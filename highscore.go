package main

type highScoreStore interface {
	Load() (int, error)
	Save(score int) error
}
