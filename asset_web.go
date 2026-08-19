//go:build js

package main

import (
	"fmt"
	"io"
	"net/http"
)

func openAsset(path string) (io.ReadCloser, error) {
	response, err := http.Get(path)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		response.Body.Close()
		return nil, fmt.Errorf("GET %s: %s", path, response.Status)
	}
	return response.Body, nil
}
