//go:build js

package main

import "syscall/js"

func debugModeEnabled() bool {
	params := js.Global().Get("URLSearchParams").New(js.Global().Get("location").Get("search"))
	return params.Call("get", "debug").String() == "1"
}
