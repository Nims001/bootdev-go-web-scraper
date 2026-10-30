package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
)

type config struct {
	pages              map[string]PageData
	baseURL            *url.URL
	mu                 *sync.Mutex
	concurrencyControl chan struct{}
	wg                 *sync.WaitGroup
	maxPages           int
}

func main() {
	arguments := os.Args[1:] // get the command line arguments excluding the program name as os.Args[0] is the program name

	if len(arguments) == 0 {
		fmt.Println("no website provided")
		os.Exit(1)
	}

	// we give arguments of maxpages and maxconcurrency
	//if more than 3 arguments then the other arguments are invalid
	if len(arguments) > 3 {
		fmt.Println("too many arguments provided")
		//exit with a non-zero status code to indicate an error
		os.Exit(1)
	}
	// make default maxpages
	maxConcurrency := 2
	if len(arguments) > 1 {
		fmt.Sscanf(arguments[1], "%d", &maxConcurrency)
	}
	// make default max concurrency if argument given then use it
	maxPages := 5
	if len(arguments) > 2 {
		fmt.Sscanf(arguments[2], "%d", &maxPages)
	}

	baseURl := arguments[0]
	parsedURL, err := url.Parse(baseURl)
	if err != nil {
		log.Fatalf("Failed to parse URL: %v", err)
	}

	pages := make(map[string]PageData)
	cfg := &config{
		pages:              pages,
		baseURL:            parsedURL,
		mu:                 &sync.Mutex{},
		concurrencyControl: make(chan struct{}, maxConcurrency),
		wg:                 &sync.WaitGroup{},
		maxPages:           maxPages,
	}

	fmt.Println("starting crawl of:", baseURl)
	cfg.wg.Add(1)

	// Start the crawl in a separate goroutine to allow for concurrent crawling of pages
	// defer the call to cfg.wg.Done() to ensure that it is called when the goroutine completes, even if an error occurs
	go func() {
		defer cfg.wg.Done()
		cfg.crawlPage(baseURl)
	}()

	cfg.wg.Wait() // wait for all goroutines to complete
	fmt.Println("crawl finished. found", len(cfg.pages), "pages")
	for normalizedURL, pageData := range cfg.pages {
		// print the pages map data in a readable format using the %+v verb,
		// which prints the struct fields and their values
		fmt.Printf("URL: %s, Data: %+v\n", normalizedURL, pageData)
	}
	writeJSONReport(cfg.pages, "report.json")
	fmt.Println("report written to report.json")
}

func getHTML(rawURL string) (string, error) {

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "BootCrawler/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close() // close the response body when the function returns

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error: status code %d", resp.StatusCode)
	}

	//contentType := resp.Header.Get("Content-Type")
	//if contentType != "text/html; charset=utf-8" {
	//return "", fmt.Errorf("error: content type is not text/html")
	//}
	// resp.Body is an io.ReadCloser, which is a combination of an io.Reader and an io.Closer. We can read the response body using io.ReadAll,
	// which reads from the reader until EOF and returns the data it read. We also need to handle any errors that may occur during reading.
	// We can use the io.ReadAll function to read the entire response body into a byte slice, and then convert it to a string.
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(bodyBytes), nil
}

func (cfg *config) crawlPage(rawCurrentURL string) {
	cfg.mu.Lock()
	if len(cfg.pages) >= cfg.maxPages {
		cfg.mu.Unlock()
		return
	}
	cfg.mu.Unlock()

	normalizedURL, err := normalizeURL(rawCurrentURL)
	if err != nil {
		fmt.Println("couldn't normalize url")
		return
	}

	currentURL, err := url.Parse(rawCurrentURL)
	if err != nil {
		fmt.Println("couldn't parse url")
		return
	}
	// if the hostname of the base URL and the current URL are not the same, we don't want to crawl the page (outside the domain), so we return early
	if cfg.baseURL.Hostname() != currentURL.Hostname() {
		return
	}

	if !cfg.addPageVisit(normalizedURL) {
		return
	}

	// acquire token only after checking if url has same base url
	// to avoid blocking other goroutines from acquiring a token and crawling pages within the same domain
	cfg.concurrencyControl <- struct{}{}        // acquire a token to start the crawl
	defer func() { <-cfg.concurrencyControl }() // release the token when the function returns

	html, err := getHTML(rawCurrentURL)
	fmt.Println("crawling:", rawCurrentURL)
	if err != nil {
		fmt.Printf("error getting HTML: %v", err)
		return
	}

	// get page data for the extracted HTML and the normalized URL, and store it in the pages map
	extractedPageData := extractPageData(html, rawCurrentURL)

	// update the pages while making sure to lock the mutex to avoid race conditions when
	// multiple goroutines are trying to update the pages map concurrently
	cfg.mu.Lock()
	cfg.pages[normalizedURL] = extractedPageData
	cfg.mu.Unlock()

	// for each outgoing link on the page, we add a new goroutine to crawl that page
	//reading from cfg.pages is unsafe here so we use extractedPageData.OutgoingLinks instead of cfg.pages[normalizedURL].OutgoingLinks to avoid race conditions
	for _, link := range extractedPageData.OutgoingLinks {
		cfg.wg.Add(1)
		go func() {
			defer cfg.wg.Done()
			cfg.crawlPage(link)
		}()
	}

	// wait for all goroutines to complete before returning from the function

}

func (cfg *config) addPageVisit(normalizedURL string) (shouldVisit bool) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	if _, ok := cfg.pages[normalizedURL]; ok {
		return false
	}

	cfg.pages[normalizedURL] = PageData{
		URL: normalizedURL,
	}

	return true
}
