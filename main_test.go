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

func TestComboMultiplierAndScore(t *testing.T) {
	game := &Game{}
	for range comboStep {
		game.recordHit(1)
	}

	if game.combo != comboStep {
		t.Fatalf("combo = %d, want %d", game.combo, comboStep)
	}
	if multiplier := game.comboMultiplier(); multiplier != 2 {
		t.Fatalf("multiplier = %d, want 2", multiplier)
	}
	if game.score != 6 {
		t.Fatalf("score = %d, want 6", game.score)
	}

	game.combo = comboStep * 20
	if multiplier := game.comboMultiplier(); multiplier != maxComboBonus {
		t.Fatalf("capped multiplier = %d, want %d", multiplier, maxComboBonus)
	}
}

func TestBossWaveAndHealthScaling(t *testing.T) {
	for _, wave := range []int{bossWaveCycle, bossWaveCycle * 2, bossWaveCycle * 3} {
		if !isBossWave(wave) {
			t.Fatalf("wave %d should be a boss wave", wave)
		}
	}
	for _, wave := range []int{0, 1, bossWaveCycle - 1, bossWaveCycle + 1} {
		if isBossWave(wave) {
			t.Fatalf("wave %d should not be a boss wave", wave)
		}
	}

	if hp := bossHealthForWave(bossWaveCycle); hp != bossBaseHP {
		t.Fatalf("first boss HP = %d, want %d", hp, bossBaseHP)
	}
	wantSecondBossHP := bossBaseHP + bossHPGrowth
	if hp := bossHealthForWave(bossWaveCycle * 2); hp != wantSecondBossHP {
		t.Fatalf("second boss HP = %d, want %d", hp, wantSecondBossHP)
	}
}

func TestDebugModeEnabledFromEnvironment(t *testing.T) {
	t.Setenv("MYGAME_DEBUG", "1")
	if !debugModeEnabled() {
		t.Fatal("debug mode should be enabled when MYGAME_DEBUG=1")
	}

	t.Setenv("MYGAME_DEBUG", "0")
	if debugModeEnabled() {
		t.Fatal("debug mode should be disabled unless MYGAME_DEBUG=1")
	}
}
