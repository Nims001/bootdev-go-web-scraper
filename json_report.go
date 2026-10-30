package main

import (
	"encoding/json"
	"os"
	"sort"
)

func writeJSONReport(pages map[string]PageData, filename string) error {
	// Create a file to write the JSON report
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	keys := make([]string, 0, len(pages)) // Allocate with capacity for efficiency
	for k := range pages {
		keys = append(keys, k)
	}
	//keys := slices.Collect(maps.Keys(pages)) // get keys of the pages map
	sort.Strings(keys) // sort the keys in ascending order

	sortedPages := make([]PageData, 0, len(pages)) // create a slice to hold the sorted pages
	// for each key in the sorted keys, append the corresponding PageData to the sortedPages slice
	for _, key := range keys {
		sortedPages = append(sortedPages, pages[key]) // append the PageData to the sortedPages slice
	}
	// Marshal the pages map to JSON
	jsonData, err := json.MarshalIndent(sortedPages, "", "  ")
	if err != nil {
		return err
	}

	// Write the JSON data to the file
	_, err = file.Write(jsonData)
	if err != nil {
		return err
	}

	return nil
}
