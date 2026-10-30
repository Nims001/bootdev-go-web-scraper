package main

import (
	"net/url"
)

func normalizeURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	// Remove the scheme (http:// or https://)
	// normalized url will be in the form of "www.boot.dev/blog/path"

	return parsedURL.Host + parsedURL.Path, nil
}
