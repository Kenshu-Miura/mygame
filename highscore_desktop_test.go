//go:build !js

package main

import (
	"path/filepath"
	"testing"
)

func TestDesktopHighScoreStoreRoundTrip(t *testing.T) {
	store := &platformHighScoreStore{path: filepath.Join(t.TempDir(), "mygame", "highscore")}
	if err := store.Save(123); err != nil {
		t.Fatalf("save high score: %v", err)
	}
	score, err := store.Load()
	if err != nil {
		t.Fatalf("load high score: %v", err)
	}
	if score != 123 {
		t.Fatalf("loaded high score = %d, want 123", score)
	}
}
