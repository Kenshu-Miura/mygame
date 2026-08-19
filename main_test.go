package main

import "testing"

func TestShouldFireWhileSpaceIsHeld(t *testing.T) {
	game := &Game{}
	firedAt := make([]int, 0, 3)

	for tick := 0; tick < shotInterval*3; tick++ {
		if game.shouldFire(true) {
			firedAt = append(firedAt, tick)
		}
	}

	want := []int{0, shotInterval, shotInterval * 2}
	if len(firedAt) != len(want) {
		t.Fatalf("fired at %v, want %v", firedAt, want)
	}
	for i := range want {
		if firedAt[i] != want[i] {
			t.Fatalf("fired at %v, want %v", firedAt, want)
		}
	}
}

func TestShouldFireImmediatelyAfterSpaceIsReleased(t *testing.T) {
	game := &Game{}
	if !game.shouldFire(true) {
		t.Fatal("first press did not fire")
	}
	if game.shouldFire(true) {
		t.Fatal("fired again before the interval elapsed")
	}
	if game.shouldFire(false) {
		t.Fatal("fired after Space was released")
	}
	if !game.shouldFire(true) {
		t.Fatal("new press did not fire immediately")
	}
}
