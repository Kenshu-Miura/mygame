//go:build !js

package main

import "os"

func debugModeEnabled() bool {
	return os.Getenv("MYGAME_DEBUG") == "1"
}
