package main

import (
	"math/rand"
	"testing"
)

type fakeHighScoreStore struct {
	saved []int
}

func (*fakeHighScoreStore) Load() (int, error) {
	return 0, nil
}

func (store *fakeHighScoreStore) Save(score int) error {
	store.saved = append(store.saved, score)
	return nil
}

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

func TestHorizontalEnemySpawnsFromBothSides(t *testing.T) {
	const (
		imageWidth = 32
		y          = 80
		speed      = 3.5
	)

	fromLeft := newHorizontalEnemy(imageWidth, y, speed, true)
	if fromLeft.x != -imageWidth || fromLeft.velocityX != speed {
		t.Fatalf("left spawn = %+v, want x=%d velocity=%v", fromLeft, -imageWidth, speed)
	}

	fromRight := newHorizontalEnemy(imageWidth, y, speed, false)
	if fromRight.x != screenWidth || fromRight.velocityX != -speed {
		t.Fatalf("right spawn = %+v, want x=%d velocity=%v", fromRight, screenWidth, -speed)
	}
}

func TestEnemySpeedIncreasesByWaveAndIsCapped(t *testing.T) {
	if got := enemySpeedForWave(1); got != enemySpeed {
		t.Fatalf("wave 1 speed = %v, want %v", got, enemySpeed)
	}
	if enemySpeedForWave(2) <= enemySpeedForWave(1) {
		t.Fatal("enemy speed did not increase at wave 2")
	}
	if got := enemySpeedForWave(100); got != maxEnemySpeed {
		t.Fatalf("capped speed = %v, want %v", got, maxEnemySpeed)
	}
}

func TestBossMovementUsesBothDirectionsAndBoundedTiming(t *testing.T) {
	random := rand.New(rand.NewSource(1))
	seenLeft := false
	seenRight := false
	for range 100 {
		direction := randomHorizontalDirection(random)
		seenLeft = seenLeft || direction < 0
		seenRight = seenRight || direction > 0
	}
	if !seenLeft || !seenRight {
		t.Fatalf("random boss movement directions: left=%t right=%t", seenLeft, seenRight)
	}

	game := &Game{random: rand.New(rand.NewSource(2))}
	for range 100 {
		duration := game.randomBossMoveTime()
		if duration < bossMoveMinTime || duration > bossMoveMinTime+bossMoveVariance {
			t.Fatalf("boss movement duration %d is outside the configured range", duration)
		}
	}
}

func TestPowerUpCreatesThreeShotsAndExpires(t *testing.T) {
	game := &Game{}
	game.fireProjectiles(100, 200)
	if len(game.projectiles) != 1 {
		t.Fatalf("normal shot count = %d, want 1", len(game.projectiles))
	}

	game.projectiles = nil
	game.activatePowerUp()
	if game.powerUpTicks != powerUpDuration {
		t.Fatalf("power-up duration = %d, want %d", game.powerUpTicks, powerUpDuration)
	}
	game.fireProjectiles(100, 200)
	if len(game.projectiles) != powerUpShotCount {
		t.Fatalf("powered shot count = %d, want %d", len(game.projectiles), powerUpShotCount)
	}
	if game.projectiles[0].x != 100 || game.projectiles[1].x != 100-powerUpShotGap || game.projectiles[2].x != 100+powerUpShotGap {
		t.Fatalf("powered shot positions = %+v", game.projectiles)
	}
	game.powerUpTicks = 1
	game.moveEntities()
	if game.powerUpTicks != 0 {
		t.Fatalf("expired power-up ticks = %d, want 0", game.powerUpTicks)
	}
}

func TestPowerUpDropUsesConfiguredRate(t *testing.T) {
	game := &Game{random: rand.New(rand.NewSource(1))}
	for range powerUpDropRate * 20 {
		game.maybeDropPowerUp(point{x: 12, y: 34})
	}
	if len(game.powerUps) == 0 || len(game.powerUps) == powerUpDropRate*20 {
		t.Fatalf("power-up drops = %d, want some but not all attempts", len(game.powerUps))
	}
	for _, item := range game.powerUps {
		if item.x != 12 || item.y != 34 {
			t.Fatalf("power-up position = %+v, want (12,34)", item)
		}
	}
}

func TestKIEEGaugeFillIsClamped(t *testing.T) {
	const width = 100
	if got := kieeGaugeFillWidth(-5, width); got != 0 {
		t.Fatalf("negative gauge width = %v, want 0", got)
	}
	if got := kieeGaugeFillWidth(specialCost/2, width); got != width/2 {
		t.Fatalf("half gauge width = %v, want %d", got, width/2)
	}
	if got := kieeGaugeFillWidth(specialCost+10, width); got != width {
		t.Fatalf("overfilled gauge width = %v, want %d", got, width)
	}
}

func TestHighScoreOnlySavesNewRecords(t *testing.T) {
	store := &fakeHighScoreStore{}
	game := &Game{score: 10, highScore: 10, highScoreStore: store}

	game.addScore(5)
	game.addScore(-8)
	game.addScore(2)

	if game.highScore != 15 {
		t.Fatalf("high score = %d, want 15", game.highScore)
	}
	if len(store.saved) != 1 || store.saved[0] != 15 {
		t.Fatalf("saved high scores = %v, want [15]", store.saved)
	}
}
