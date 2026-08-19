//go:build !js

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type platformHighScoreStore struct {
	path    string
	initErr error
}

func newHighScoreStore() highScoreStore {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return &platformHighScoreStore{initErr: err}
	}
	return &platformHighScoreStore{path: filepath.Join(configDir, "mygame", "highscore")}
}

func (store *platformHighScoreStore) Load() (int, error) {
	if store.initErr != nil {
		return 0, store.initErr
	}
	data, err := os.ReadFile(store.path)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	score, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", store.path, err)
	}
	return max(0, score), nil
}

func (store *platformHighScoreStore) Save(score int) error {
	if store.initErr != nil {
		return store.initErr
	}
	if err := os.MkdirAll(filepath.Dir(store.path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(store.path, []byte(strconv.Itoa(max(0, score))), 0o644)
}
