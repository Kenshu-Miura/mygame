//go:build js

package main

import (
	"fmt"
	"strconv"
	"syscall/js"
)

const highScoreStorageKey = "mygame.highScore"

type platformHighScoreStore struct{}

func newHighScoreStore() highScoreStore {
	return &platformHighScoreStore{}
}

func (*platformHighScoreStore) Load() (score int, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			score = 0
			err = fmt.Errorf("read localStorage: %v", recovered)
		}
	}()

	value := js.Global().Get("localStorage").Call("getItem", highScoreStorageKey)
	if value.IsNull() || value.IsUndefined() || value.String() == "" {
		return 0, nil
	}
	score, err = strconv.Atoi(value.String())
	if err != nil {
		return 0, fmt.Errorf("parse localStorage value: %w", err)
	}
	return max(0, score), nil
}

func (*platformHighScoreStore) Save(score int) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("write localStorage: %v", recovered)
		}
	}()

	js.Global().Get("localStorage").Call("setItem", highScoreStorageKey, strconv.Itoa(max(0, score)))
	return nil
}
