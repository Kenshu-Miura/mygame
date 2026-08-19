//go:build !js

package main

import (
	"io"
	"os"
)

func openAsset(path string) (io.ReadCloser, error) {
	return os.Open(path)
}
