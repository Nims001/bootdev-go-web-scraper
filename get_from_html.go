package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func getHeadingFromHTML(html string) string {

	// create a document reader
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return ""
	}
	// find the first h1 element and get its text

	heading := doc.Find("h1")

	if heading.Length() == 0 {
		// if no h1 is found, find the first h2 element and get its text
		heading = doc.Find("h2")

	}
	return heading.Text()
}

func getFirstParagraphFromHTML(html string) string {

	// create a document reader
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return ""
	}
	// find the first p element inside main and get its text
	paragraph := doc.Find("main p")

	if paragraph.First().Length() == 0 {
		// if no p is found inside main, find the first p element and get its text
		paragraph = doc.Find("p")
	}

	return paragraph.First().Text()
}

func getURLsFromHTML(htmlBody string, baseURL *url.URL) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return nil, err
	}

	var urls []string
	var hrefErr error

	// finding the a elements with href attributes and extracting the URLs
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		// s.Attr("href") returns the value of the href attribute and a boolean indicating if it exists
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		linkURL, err := url.Parse(href)
		if err != nil {
			hrefErr = err
			return
		}

		urls = append(urls, baseURL.ResolveReference(linkURL).String())
	})
	if hrefErr != nil {
		return nil, hrefErr
	}

	return urls, nil
}

func getImagesFromHTML(htmlBody string, baseURL *url.URL) ([]string, error) {
	// setup html reader
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return nil, err
	}

	var images []string
	var srcErr error

	// finding the img elements with src attributes and extracting the image URLs,.Each method iterates over each matched element and applies the provided function to it.
	doc.Find("img[src]").Each(func(_ int, s *goquery.Selection) {
		// s.Attr("src") returns the value of the src attribute and a boolean indicating if it exists
		src, exists := s.Attr("src")
		if !exists {
			return
		}
		// This turns the raw string (which might be relative like /logo.png or absolute like https://crawler-test.com)
		// into a structured *url.URL broken into scheme, host, path, etc.
		imgURL, err := url.Parse(src)
		if err != nil {
			srcErr = err
			return
		}
		// ResolveReference is the key trick: given the "root" baseURL, it combines it with imgURL.
		// If imgUrl is already absolute, it's returned basically unchanged. If imgURL is relative (just a path),
		// it gets stitched onto baseURL's scheme and host.ResolveReference is the key trick: given the "root" baseURL, it combines it with imgURL. If linkURL is already absolute, it's returned basically unchanged.
		// If linkURL is relative (just a path), it gets stitched onto baseURL's scheme and host.
		images = append(images, baseURL.ResolveReference(imgURL).String())
	})
	if srcErr != nil {
		return nil, srcErr
	}

	return images, nil
}

type PageData struct {
	URL            string   `json:"url"`
	Heading        string   `json:"heading"`
	FirstParagraph string   `json:"first_paragraph"`
	OutgoingLinks  []string `json:"outgoing_links"`
	ImageURLs      []string `json:"image_urls"`
}

func extractPageData(html, pageURL string) PageData {
	var pageData PageData
	pageData.URL = pageURL
	pageData.Heading = getHeadingFromHTML(html)
	pageData.FirstParagraph = getFirstParagraphFromHTML(html)

	absURL, err := url.Parse(pageURL)
	if err != nil {
		fmt.Println("Couldn't parse url")
		return PageData{}
	}

	pageData.OutgoingLinks, err = getURLsFromHTML(html, absURL)
	if err != nil {
		fmt.Println("Couldn't get URLs from HTML")
		return PageData{}
	}

	pageData.ImageURLs, err = getImagesFromHTML(html, absURL)
	if err != nil {
		fmt.Println("Couldn't get image URLs from HTML")
		return PageData{}
	}

	return pageData

}
