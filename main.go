package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"io"
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
)

const (
	screenWidth          = 640
	screenHeight         = 480
	windowScale          = 2
	playerScale          = 0.1
	playerSpeed          = 4
	projectileSpeed      = 2
	enemySpeed           = 2
	enemySpeedGain       = 0.25
	maxEnemySpeed        = 6
	audioSampleRate      = 48_000
	specialCost          = 20
	shotInterval         = 10 // At 60 TPS, holding Space fires about six shots per second.
	bossWaveCycle        = 5
	waveBannerTime       = 90
	ufoBaseTarget        = 5
	ufoMaxTarget         = 20
	bossScale            = 0.22
	bossSpeed            = 1.5
	bossY                = -18
	bossMoveMinTime      = 30
	bossMoveVariance     = 90
	bossBaseHP           = 30
	bossHPGrowth         = 15
	bossAttackTime       = 75
	bossSpecialHit       = 10
	bossDefeatBonus      = 25
	comboStep            = 5
	maxComboBonus        = 5
	powerUpSize          = 20
	powerUpSpeed         = 1.5
	powerUpDropRate      = 5 // One in five defeated UFOs drops an item.
	powerUpDuration      = 10 * 60
	powerUpShotCount     = 3
	powerUpDiagonalSpeed = 1.4
	fallingSpeedBase     = 1.2
	fallingSpeedGain     = 0.25
	maxFallingSpeed      = 5
	touchTapDistance     = 14
	touchSpecialDistance = 60
)

// The raised fingertip is about 11% of the way across ebisan.png.
const playerFingerTipXRatio = 0.11

type gameState uint8

const (
	stateTitle gameState = iota
	statePlaying
	stateGameOver
)

type point struct {
	x float64
	y float64
}

type projectile struct {
	point
	velocityX float64
	velocityY float64
}

type ufo struct {
	horizontalEnemy
	visible bool
}

type horizontalEnemy struct {
	point
	velocityX float64
}

type powerUp struct {
	point
}

type boss struct {
	point
	hp             int
	maxHP          int
	direction      float64
	attackCooldown int
	moveCooldown   int
}

type Game struct {
	player point
	state  gameState

	projectiles []projectile
	ufos        []ufo
	bashiHebis  []point
	ebis        []horizontalEnemy
	powerUps    []powerUp
	boss        *boss
	debug       bool

	score           int
	highScore       int
	combo           int
	missCount       int
	shotCooldown    int
	powerUpTicks    int
	wave            int
	ufoKills        int
	waveBannerTicks int
	random          *rand.Rand
	touch           touchGesture
	touchShot       bool
	touchSpecial    bool

	playerImage   *ebiten.Image
	backgroundImg *ebiten.Image
	ufoImage      *ebiten.Image
	projectileImg *ebiten.Image
	bashiHebiImg  *ebiten.Image
	ebiImage      *ebiten.Image
	bossImage     *ebiten.Image
	font          font.Face

	shotSound  *audio.Player
	hitSound   *audio.Player
	kieeSound  *audio.Player
	kieeSound2 *audio.Player
	hoaaSound  *audio.Player
	bgm        *audio.Player
	gameOverSE *audio.Player

	highScoreStore highScoreStore
}

func newGame() (*Game, error) {
	gameFont, err := loadFont()
	if err != nil {
		return nil, fmt.Errorf("load font: %w", err)
	}

	backgroundImage, err := loadImage("space_background.png")
	if err != nil {
		return nil, err
	}
	playerImage, err := loadImage("ebisan.png")
	if err != nil {
		return nil, err
	}
	ufoImage, err := loadImage("ufo.png")
	if err != nil {
		return nil, err
	}
	projectileImage, err := loadImage("o.png")
	if err != nil {
		return nil, err
	}
	bashiHebiImage, err := loadImage("bashihebi.png")
	if err != nil {
		return nil, err
	}
	ebiImage, err := loadImage("ebi.png")
	if err != nil {
		return nil, err
	}
	bossImage, err := loadImage("boss_ebi.png")
	if err != nil {
		return nil, err
	}

	audioContext := audio.NewContext(audioSampleRate)
	shotSound, err := loadWAV(audioContext, "shot.wav")
	if err != nil {
		return nil, err
	}
	hitSound, err := loadWAV(audioContext, "hit.wav")
	if err != nil {
		return nil, err
	}
	kieeSound, err := loadWAV(audioContext, "kiee.wav")
	if err != nil {
		return nil, err
	}
	kieeSound2, err := loadWAV(audioContext, "kiee2.wav")
	if err != nil {
		return nil, err
	}
	hoaaSound, err := loadWAV(audioContext, "hoaa.wav")
	if err != nil {
		return nil, err
	}
	gameOverSE, err := loadWAV(audioContext, "majide.wav")
	if err != nil {
		return nil, err
	}
	bgm, err := loadLoopingVorbis(audioContext, "BGM.ogg")
	if err != nil {
		return nil, err
	}

	store := newHighScoreStore()
	highScore, err := store.Load()
	if err != nil {
		log.Printf("load high score: %v", err)
	}

	g := &Game{
		debug:          debugModeEnabled(),
		highScore:      max(0, highScore),
		highScoreStore: store,
		backgroundImg:  backgroundImage,
		playerImage:    playerImage,
		ufoImage:       ufoImage,
		projectileImg:  projectileImage,
		bashiHebiImg:   bashiHebiImage,
		ebiImage:       ebiImage,
		bossImage:      bossImage,
		font:           gameFont,
		shotSound:      shotSound,
		hitSound:       hitSound,
		kieeSound:      kieeSound,
		kieeSound2:     kieeSound2,
		hoaaSound:      hoaaSound,
		bgm:            bgm,
		gameOverSE:     gameOverSE,
	}
	g.reset()
	return g, nil
}

func loadFont() (font.Face, error) {
	parsed, err := opentype.Parse(fonts.MPlus1pRegular_ttf)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    24,
		DPI:     72,
		Hinting: font.HintingVertical,
	})
}

func loadImage(path string) (*ebiten.Image, error) {
	file, err := openAsset(path)
	if err != nil {
		return nil, fmt.Errorf("open image %q: %w", path, err)
	}
	defer file.Close()

	source, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("decode image %q: %w", path, err)
	}
	return ebiten.NewImageFromImage(source), nil
}

func loadWAV(context *audio.Context, path string) (*audio.Player, error) {
	data, err := readAsset(path)
	if err != nil {
		return nil, fmt.Errorf("open sound %q: %w", path, err)
	}

	stream, err := wav.DecodeWithSampleRate(audioSampleRate, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode sound %q: %w", path, err)
	}
	player, err := context.NewPlayer(stream)
	if err != nil {
		return nil, fmt.Errorf("create player for %q: %w", path, err)
	}
	return player, nil
}

func loadLoopingVorbis(context *audio.Context, path string) (*audio.Player, error) {
	data, err := readAsset(path)
	if err != nil {
		return nil, fmt.Errorf("open music %q: %w", path, err)
	}

	stream, err := vorbis.DecodeWithSampleRate(audioSampleRate, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode music %q: %w", path, err)
	}
	player, err := context.NewPlayer(audio.NewInfiniteLoop(stream, stream.Length()))
	if err != nil {
		return nil, fmt.Errorf("create player for %q: %w", path, err)
	}
	return player, nil
}

func readAsset(path string) ([]byte, error) {
	file, err := openAsset(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

func replay(player *audio.Player) {
	if err := player.Rewind(); err != nil {
		log.Printf("rewind audio: %v", err)
		return
	}
	player.Play()
}

func (g *Game) reset() {
	g.player.x = float64(screenWidth)/2 - float64(g.playerImage.Bounds().Dx())*playerScale/2
	g.player.y = float64(screenHeight) - float64(g.playerImage.Bounds().Dy())*playerScale
	g.projectiles = nil
	g.ufos = nil
	g.bashiHebis = nil
	g.ebis = nil
	g.powerUps = nil
	g.boss = nil
	g.score = 0
	g.combo = 0
	g.missCount = 0
	g.shotCooldown = 0
	g.powerUpTicks = 0
	g.wave = 1
	g.ufoKills = 0
	g.waveBannerTicks = waveBannerTime
	g.touch = touchGesture{}
	g.touchShot = false
	g.touchSpecial = false
	g.random = rand.New(rand.NewSource(time.Now().UnixNano()))
	g.state = stateTitle
	g.bgm.Pause()
	g.gameOverSE.Pause()
}

func (g *Game) Update() error {
	switch g.state {
	case stateTitle:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) || touchJustPressed() {
			g.state = statePlaying
			replay(g.bgm)
		}
		return nil
	case stateGameOver:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || touchJustPressed() {
			g.reset()
		}
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.reset()
		return nil
	}

	g.handleTouchInput()
	g.handleDebugInput()
	g.updateWave()
	g.handlePlayerInput()
	g.handleProjectileCollisions()
	g.handleSpecialAttack()
	g.spawnEnemies()
	g.moveEntities()
	g.handlePowerUpCollisions()
	g.handlePlayerCollision()
	g.removeOffscreenEntities()
	return nil
}

func (g *Game) handleDebugInput() {
	if !g.debug {
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		nextBossWave := (g.wave/bossWaveCycle + 1) * bossWaveCycle
		if isBossWave(g.wave) {
			nextBossWave = g.wave
		}
		g.startWave(nextBossWave)
		log.Printf("debug: jumped to boss wave %d", nextBossWave)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyK) {
		g.missCount = max(g.missCount, specialCost)
		log.Printf("debug: filled KIEE gauge to %d", g.missCount)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.powerUps = append(g.powerUps, powerUp{point: point{
			x: g.player.x + float64(g.playerImage.Bounds().Dx())*playerScale/2 - powerUpSize/2,
			y: g.player.y - powerUpSize - 8,
		}})
		log.Printf("debug: spawned power-up above player")
	}
}

func (g *Game) handlePlayerInput() {
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		g.movePlayerHorizontally(-playerSpeed)
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		g.movePlayerHorizontally(playerSpeed)
	}
	fire := g.shouldFire(ebiten.IsKeyPressed(ebiten.KeySpace)) || g.touchShot
	g.touchShot = false
	if fire {
		g.firePlayerShot()
	}
}

func (g *Game) firePlayerShot() {
	playerWidth := float64(g.playerImage.Bounds().Dx()) * playerScale
	projectileWidth := float64(g.projectileImg.Bounds().Dx())
	shotX := g.player.x + playerWidth*playerFingerTipXRatio - projectileWidth/2
	g.fireProjectiles(shotX, g.player.y)
	replay(g.shotSound)
}

func (g *Game) movePlayerHorizontally(distance float64) {
	playerWidth := float64(g.playerImage.Bounds().Dx()) * playerScale
	g.player.x = min(float64(screenWidth)-playerWidth, max(0, g.player.x+distance))
}

func (g *Game) fireProjectiles(x, y float64) {
	g.projectiles = append(g.projectiles, projectile{
		point:     point{x: x, y: y},
		velocityY: -projectileSpeed,
	})
	if g.powerUpTicks <= 0 {
		return
	}
	g.projectiles = append(g.projectiles,
		projectile{point: point{x: x, y: y}, velocityX: -powerUpDiagonalSpeed, velocityY: -powerUpDiagonalSpeed},
		projectile{point: point{x: x, y: y}, velocityX: powerUpDiagonalSpeed, velocityY: -powerUpDiagonalSpeed},
	)
}

func (g *Game) shouldFire(spacePressed bool) bool {
	if !spacePressed {
		g.shotCooldown = 0
		return false
	}
	if g.shotCooldown > 0 {
		g.shotCooldown--
		return false
	}
	g.shotCooldown = shotInterval - 1
	return true
}

func (g *Game) recordHit(baseScore int) {
	g.combo++
	g.addScore(baseScore * g.comboMultiplier())
}

func (g *Game) recordUFODefeat() bool {
	g.recordHit(1)
	g.ufoKills++
	target := ufoTargetForWave(g.wave)
	return target > 0 && g.ufoKills >= target
}

func (g *Game) addScore(points int) {
	g.score = max(0, g.score+points)
	if g.score <= g.highScore {
		return
	}
	g.highScore = g.score
	if g.highScoreStore != nil {
		if err := g.highScoreStore.Save(g.highScore); err != nil {
			log.Printf("save high score: %v", err)
		}
	}
}

func (g *Game) comboMultiplier() int {
	return min(maxComboBonus, 1+g.combo/comboStep)
}

func isBossWave(wave int) bool {
	return wave > 0 && wave%bossWaveCycle == 0
}

func bossHealthForWave(wave int) int {
	return bossBaseHP + max(0, wave/bossWaveCycle-1)*bossHPGrowth
}

func enemySpeedForWave(wave int) float64 {
	return min(float64(maxEnemySpeed), float64(enemySpeed)+float64(max(0, wave-1))*enemySpeedGain)
}

func ufoTargetForWave(wave int) int {
	if isBossWave(wave) {
		return 0
	}
	return min(ufoMaxTarget, ufoBaseTarget+max(0, wave-1))
}

func fallingEnemySpeedForWave(wave int) float64 {
	return min(float64(maxFallingSpeed), fallingSpeedBase+float64(max(0, wave-1))*fallingSpeedGain)
}

func newHorizontalEnemy(imageWidth int, y, speed float64, fromLeft bool) horizontalEnemy {
	if fromLeft {
		return horizontalEnemy{point: point{x: -float64(imageWidth), y: y}, velocityX: speed}
	}
	return horizontalEnemy{point: point{x: screenWidth, y: y}, velocityX: -speed}
}

func horizontalEnemyOffscreen(enemy horizontalEnemy, imageWidth int) bool {
	return enemy.x+float64(imageWidth) < 0 || enemy.x > screenWidth
}

func horizontalSpawnSide(velocity float64) string {
	if velocity > 0 {
		return "left"
	}
	return "right"
}

func horizontalMovementDirection(velocity float64) string {
	if velocity > 0 {
		return "right"
	}
	return "left"
}

func randomHorizontalDirection(random *rand.Rand) float64 {
	if random.Intn(2) == 0 {
		return -1
	}
	return 1
}

func (g *Game) randomBossMoveTime() int {
	return bossMoveMinTime + g.random.Intn(bossMoveVariance+1)
}

func (g *Game) updateWave() {
	if g.waveBannerTicks > 0 {
		g.waveBannerTicks--
	}
}

func (g *Game) startWave(wave int) {
	g.wave = wave
	g.ufoKills = 0
	g.waveBannerTicks = waveBannerTime
	g.projectiles = nil
	g.ufos = nil
	g.bashiHebis = nil
	g.ebis = nil
	g.boss = nil

	if !isBossWave(wave) {
		return
	}

	hp := bossHealthForWave(wave)
	bossWidth := float64(g.bossImage.Bounds().Dx()) * bossScale
	g.boss = &boss{
		point:          point{x: (screenWidth - bossWidth) / 2, y: bossY},
		hp:             hp,
		maxHP:          hp,
		direction:      randomHorizontalDirection(g.random),
		attackCooldown: bossAttackTime,
		moveCooldown:   g.randomBossMoveTime(),
	}
}

func (g *Game) finishBossWave() {
	g.addScore(bossDefeatBonus * g.comboMultiplier())
	g.startWave(g.wave + 1)
}

func (g *Game) handleProjectileCollisions() {
	for projectileIndex := len(g.projectiles) - 1; projectileIndex >= 0; projectileIndex-- {
		projectile := g.projectiles[projectileIndex]
		projectileRect := g.projectileImg.Bounds().Add(image.Pt(int(projectile.x), int(projectile.y)))
		hit := false
		bossDefeated := false
		waveComplete := false

		if g.boss != nil && projectileRect.Overlaps(g.bossRect()) {
			g.boss.hp--
			g.recordHit(1)
			replay(g.hitSound)
			hit = true
			bossDefeated = g.boss.hp <= 0
		}

		for ufoIndex := range g.ufos {
			if hit {
				break
			}
			target := &g.ufos[ufoIndex]
			if !target.visible {
				continue
			}
			targetRect := g.ufoImage.Bounds().Add(image.Pt(int(target.x), int(target.y)))
			if projectileRect.Overlaps(targetRect) {
				dropPosition := point{
					x: target.x + float64(g.ufoImage.Bounds().Dx()-powerUpSize)/2,
					y: target.y + float64(g.ufoImage.Bounds().Dy()-powerUpSize)/2,
				}
				target.visible = false
				g.maybeDropPowerUp(dropPosition)
				waveComplete = g.recordUFODefeat()
				replay(g.hitSound)
				hit = true
				break
			}
		}

		if !hit {
			for ebiIndex := len(g.ebis) - 1; ebiIndex >= 0; ebiIndex-- {
				target := g.ebis[ebiIndex]
				targetRect := g.ebiImage.Bounds().Add(image.Pt(int(target.x), int(target.y)))
				if projectileRect.Overlaps(targetRect) {
					g.ebis = removeAt(g.ebis, ebiIndex)
					g.addScore(-2)
					g.combo = 0
					replay(g.hoaaSound)
					replay(g.hitSound)
					hit = true
					break
				}
			}
		}

		if hit {
			g.projectiles = removeAt(g.projectiles, projectileIndex)
		}
		if bossDefeated {
			g.finishBossWave()
			return
		}
		if waveComplete {
			g.startWave(g.wave + 1)
			return
		}
	}
}

func (g *Game) maybeDropPowerUp(position point) {
	if g.random.Intn(powerUpDropRate) != 0 {
		return
	}
	g.powerUps = append(g.powerUps, powerUp{point: position})
	if g.debug {
		log.Printf("debug: power-up dropped at (%.1f,%.1f)", position.x, position.y)
	}
}

func (g *Game) handleSpecialAttack() {
	requested := inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || g.touchSpecial
	g.touchSpecial = false
	if g.missCount < specialCost || !requested {
		return
	}

	g.missCount -= specialCost
	waveComplete := false
	if g.boss != nil {
		damage := min(bossSpecialHit, g.boss.hp)
		for range damage {
			g.recordHit(1)
		}
		g.boss.hp -= damage
		if g.boss.hp <= 0 {
			g.finishBossWave()
		}
	} else {
		for _, target := range g.ufos {
			if target.visible && target.x+float64(g.ufoImage.Bounds().Dx()) >= 0 {
				waveComplete = g.recordUFODefeat() || waveComplete
			}
		}
		g.ufos = nil
	}
	g.bashiHebis = nil
	g.ebis = nil
	g.projectiles = nil
	replay(g.kieeSound)
	replay(g.kieeSound2)
	if waveComplete {
		g.startWave(g.wave + 1)
	}
}

func (g *Game) spawnEnemies() {
	if g.boss != nil {
		g.boss.attackCooldown--
		if g.boss.attackCooldown <= 0 {
			bossWidth := float64(g.bossImage.Bounds().Dx()) * bossScale
			attackX := g.boss.x + bossWidth/2 - float64(g.bashiHebiImg.Bounds().Dx())/2
			attackY := g.boss.y + float64(g.bossImage.Bounds().Dy())*bossScale*0.7
			g.bashiHebis = append(g.bashiHebis, point{x: attackX, y: attackY})
			g.boss.attackCooldown = bossAttackTime
		}
		return
	}

	ufoChance := min(6, 2+g.wave/2)
	if g.random.Intn(120) < ufoChance {
		movement := newHorizontalEnemy(
			g.ufoImage.Bounds().Dx(),
			float64(g.random.Intn(screenHeight/2)),
			enemySpeedForWave(g.wave),
			g.random.Intn(2) == 0,
		)
		g.ufos = append(g.ufos, ufo{
			horizontalEnemy: movement,
			visible:         true,
		})
		if g.debug {
			log.Printf("debug: spawned UFO from %s (wave=%d speed=%.2f)", horizontalSpawnSide(movement.velocityX), g.wave, enemySpeedForWave(g.wave))
		}
	}

	fallingEnemyChance := min(7, 1+g.wave/2)
	if g.random.Intn(165) < fallingEnemyChance {
		g.bashiHebis = append(g.bashiHebis, point{x: float64(g.random.Intn(screenWidth)), y: 0})
		if g.debug {
			log.Printf("debug: spawned falling enemy (wave=%d speed=%.2f)", g.wave, fallingEnemySpeedForWave(g.wave))
		}
	}

	if g.random.Intn(130) < 1 {
		movement := newHorizontalEnemy(
			g.ebiImage.Bounds().Dx(),
			float64(g.random.Intn(screenHeight/2)),
			enemySpeedForWave(g.wave),
			g.random.Intn(2) == 0,
		)
		g.ebis = append(g.ebis, movement)
		if g.debug {
			log.Printf("debug: spawned shrimp from %s (wave=%d speed=%.2f)", horizontalSpawnSide(movement.velocityX), g.wave, enemySpeedForWave(g.wave))
		}
	}
}

func (g *Game) moveEntities() {
	if g.powerUpTicks > 0 {
		g.powerUpTicks--
	}
	if g.boss != nil {
		bossWidth := float64(g.bossImage.Bounds().Dx()) * bossScale
		g.boss.moveCooldown--
		if g.boss.moveCooldown <= 0 {
			g.boss.direction = randomHorizontalDirection(g.random)
			g.boss.moveCooldown = g.randomBossMoveTime()
			if g.debug {
				log.Printf("debug: boss moving %s for %d ticks", horizontalMovementDirection(g.boss.direction), g.boss.moveCooldown)
			}
		}
		g.boss.x += bossSpeed * g.boss.direction
		if g.boss.x <= 0 {
			g.boss.x = 0
			g.boss.direction = 1
		} else if g.boss.x+bossWidth >= screenWidth {
			g.boss.x = screenWidth - bossWidth
			g.boss.direction = -1
		}
	}
	for index := range g.ufos {
		g.ufos[index].x += g.ufos[index].velocityX
	}
	fallingEnemySpeed := fallingEnemySpeedForWave(g.wave)
	for index := range g.bashiHebis {
		g.bashiHebis[index].y += fallingEnemySpeed
	}
	for index := range g.ebis {
		g.ebis[index].x += g.ebis[index].velocityX
	}
	for index := range g.projectiles {
		g.projectiles[index].x += g.projectiles[index].velocityX
		g.projectiles[index].y += g.projectiles[index].velocityY
	}
	for index := range g.powerUps {
		g.powerUps[index].y += powerUpSpeed
	}
}

func (g *Game) playerRect() image.Rectangle {
	const padding = 30
	return image.Rect(
		int(g.player.x)+padding,
		int(g.player.y)+padding,
		int(g.player.x)+int(float64(g.playerImage.Bounds().Dx())*playerScale)-padding,
		int(g.player.y)+int(float64(g.playerImage.Bounds().Dy())*playerScale)-padding,
	)
}

func (g *Game) handlePowerUpCollisions() {
	playerRect := g.playerRect()
	for index := len(g.powerUps) - 1; index >= 0; index-- {
		item := g.powerUps[index]
		itemRect := image.Rect(int(item.x), int(item.y), int(item.x)+powerUpSize, int(item.y)+powerUpSize)
		if !itemRect.Overlaps(playerRect) {
			continue
		}
		g.powerUps = removeAt(g.powerUps, index)
		g.activatePowerUp()
		if g.debug {
			log.Printf("debug: power-up collected (%d ticks)", powerUpDuration)
		}
	}
}

func (g *Game) activatePowerUp() {
	g.powerUpTicks = powerUpDuration
}

func (g *Game) bossRect() image.Rectangle {
	if g.boss == nil {
		return image.Rectangle{}
	}
	const padding = 18
	width := int(float64(g.bossImage.Bounds().Dx()) * bossScale)
	height := int(float64(g.bossImage.Bounds().Dy()) * bossScale)
	return image.Rect(
		int(g.boss.x)+padding,
		int(g.boss.y)+padding,
		int(g.boss.x)+width-padding,
		int(g.boss.y)+height-padding,
	)
}

func (g *Game) handlePlayerCollision() {
	playerRect := g.playerRect()

	for index := len(g.bashiHebis) - 1; index >= 0; index-- {
		enemy := g.bashiHebis[index]
		enemyRect := g.bashiHebiImg.Bounds().Add(image.Pt(int(enemy.x), int(enemy.y)))
		if enemyRect.Overlaps(playerRect) {
			if g.debug {
				log.Printf("debug: collision ignored (wave=%d score=%d combo=%d player=(%.1f,%.1f) enemy=(%.1f,%.1f))", g.wave, g.score, g.combo, g.player.x, g.player.y, enemy.x, enemy.y)
				g.bashiHebis = removeAt(g.bashiHebis, index)
				continue
			}
			log.Printf("game over: player collided with enemy (wave=%d score=%d combo=%d player=(%.1f,%.1f) enemy=(%.1f,%.1f))", g.wave, g.score, g.combo, g.player.x, g.player.y, enemy.x, enemy.y)
			g.combo = 0
			g.state = stateGameOver
			g.bgm.Pause()
			replay(g.gameOverSE)
			return
		}
	}
}

func (g *Game) removeOffscreenEntities() {
	for index := len(g.bashiHebis) - 1; index >= 0; index-- {
		if g.bashiHebis[index].y > screenHeight {
			g.bashiHebis = removeAt(g.bashiHebis, index)
		}
	}

	for index := len(g.projectiles) - 1; index >= 0; index-- {
		if projectileOffscreen(g.projectiles[index], g.projectileImg.Bounds().Dx(), g.projectileImg.Bounds().Dy()) {
			g.missCount++
			g.projectiles = removeAt(g.projectiles, index)
		}
	}

	for index := len(g.ufos) - 1; index >= 0; index-- {
		target := g.ufos[index]
		if !target.visible || horizontalEnemyOffscreen(target.horizontalEnemy, g.ufoImage.Bounds().Dx()) {
			g.ufos = removeAt(g.ufos, index)
		}
	}

	for index := len(g.ebis) - 1; index >= 0; index-- {
		if horizontalEnemyOffscreen(g.ebis[index], g.ebiImage.Bounds().Dx()) {
			g.ebis = removeAt(g.ebis, index)
		}
	}

	for index := len(g.powerUps) - 1; index >= 0; index-- {
		if g.powerUps[index].y > screenHeight {
			g.powerUps = removeAt(g.powerUps, index)
		}
	}
}

func removeAt[T any](values []T, index int) []T {
	copy(values[index:], values[index+1:])
	var zero T
	values[len(values)-1] = zero
	return values[:len(values)-1]
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.DrawImage(g.backgroundImg, nil)
	switch g.state {
	case stateTitle:
		g.drawTitle(screen)
		return
	case stateGameOver:
		g.drawGame(screen)
		g.drawCenteredText(screen, "GAME OVER", screenHeight/2, color.White)
		g.drawCenteredText(screen, "Escキーまたはタップでタイトルに戻る", screenHeight/2+40, color.White)
		return
	default:
		g.drawGame(screen)
	}
}

func (g *Game) drawTitle(screen *ebiten.Image) {
	g.drawCenteredText(screen, "UFO撃ち落としたことありますか？", screenHeight/2-34, color.White)
	g.drawCenteredText(screen, "Spaceキーでスタート", screenHeight/2+6, color.White)
	g.drawCenteredText(screen, "UFO撃破ノルマ達成で次のウェーブへ", screenHeight/2+46, color.White)
	g.drawCenteredText(screen, "連続命中でコンボ倍率アップ", screenHeight/2+86, color.White)
	g.drawCenteredText(screen, fmt.Sprintf("HIGH SCORE: %d", g.highScore), screenHeight/2+126, color.RGBA{R: 255, G: 220, B: 70, A: 255})
	g.drawCenteredText(screen, "スマホ: タップ発射 / 横スライド移動 / 上スワイプ必殺", screenHeight/2+158, color.RGBA{R: 130, G: 220, B: 255, A: 255})
	if g.debug {
		g.drawCenteredText(screen, "DEBUG MODE: 無敵 / B:ボス / K:KIEE / P:強化", screenHeight/2+190, color.RGBA{R: 255, G: 210, B: 60, A: 255})
	}
}

func (g *Game) drawGame(screen *ebiten.Image) {
	playerOptions := &ebiten.DrawImageOptions{}
	playerOptions.GeoM.Scale(playerScale, playerScale)
	playerOptions.GeoM.Translate(g.player.x, g.player.y)
	screen.DrawImage(g.playerImage, playerOptions)

	for _, target := range g.ufos {
		if target.visible {
			drawImageAt(screen, g.ufoImage, target.point)
		}
	}
	for _, enemy := range g.bashiHebis {
		drawImageAt(screen, g.bashiHebiImg, enemy)
	}
	for _, target := range g.ebis {
		drawImageAt(screen, g.ebiImage, target.point)
	}
	if g.boss != nil {
		bossOptions := &ebiten.DrawImageOptions{}
		bossOptions.GeoM.Scale(bossScale, bossScale)
		bossOptions.GeoM.Translate(g.boss.x, g.boss.y)
		screen.DrawImage(g.bossImage, bossOptions)
	}
	for _, projectile := range g.projectiles {
		drawImageAt(screen, g.projectileImg, projectile.point)
	}
	for _, item := range g.powerUps {
		g.drawPowerUp(screen, item)
	}

	g.drawHUD(screen)
	if g.waveBannerTicks > 0 {
		message := fmt.Sprintf("WAVE %d: UFOを%d体倒せ！", g.wave, ufoTargetForWave(g.wave))
		if isBossWave(g.wave) {
			message = fmt.Sprintf("BOSS WAVE %d: ボスを倒せ！", g.wave)
		}
		g.drawCenteredText(screen, message, 110, color.White)
	}
}

func projectileOffscreen(projectile projectile, width, height int) bool {
	return projectile.y+float64(height) < 0 ||
		projectile.x+float64(width) < 0 ||
		projectile.x > screenWidth
}

func (g *Game) drawHUD(screen *ebiten.Image) {
	text.Draw(screen, fmt.Sprintf("Score: %d  High: %d", g.score, g.highScore), basicfont.Face7x13, 1, 12, color.White)
	waveStatus := fmt.Sprintf("Wave: %d  UFO: %d/%d", g.wave, g.ufoKills, ufoTargetForWave(g.wave))
	if isBossWave(g.wave) {
		waveStatus = fmt.Sprintf("Wave: %d  Defeat BOSS", g.wave)
	}
	text.Draw(screen, waveStatus, basicfont.Face7x13, 1, 25, color.White)
	text.Draw(screen, "KIEE", basicfont.Face7x13, 1, 39, color.White)
	const (
		gaugeX      = 38
		gaugeY      = 30
		gaugeWidth  = 112
		gaugeHeight = 10
	)
	charge := kieeCharge(g.missCount)
	ebitenutil.DrawRect(screen, gaugeX, gaugeY, gaugeWidth, gaugeHeight, color.RGBA{R: 45, G: 45, B: 60, A: 255})
	fillColor := color.RGBA{R: 55, G: 190, B: 255, A: 255}
	if charge >= specialCost {
		fillColor = color.RGBA{R: 255, G: 215, B: 55, A: 255}
	}
	ebitenutil.DrawRect(screen, gaugeX+1, gaugeY+1, kieeGaugeFillWidth(charge, gaugeWidth-2), gaugeHeight-2, fillColor)
	text.Draw(screen, fmt.Sprintf("%d/%d", charge, specialCost), basicfont.Face7x13, gaugeX+gaugeWidth+5, 39, color.White)
	text.Draw(screen, fmt.Sprintf("Combo: %d  x%d", g.combo, g.comboMultiplier()), basicfont.Face7x13, 1, 54, color.White)
	if g.powerUpTicks > 0 {
		seconds := float64(g.powerUpTicks) / 60
		text.Draw(screen, fmt.Sprintf("POWER x%d  %.1fs", powerUpShotCount, seconds), basicfont.Face7x13, 1, 68, color.RGBA{R: 255, G: 225, B: 70, A: 255})
	}
	if g.debug {
		text.Draw(screen, "DEBUG: INVINCIBLE  B:BOSS  K:KIEE  P:POWER", basicfont.Face7x13, 1, screenHeight-4, color.RGBA{R: 255, G: 210, B: 60, A: 255})
	}

	if g.boss == nil {
		return
	}
	const (
		barX      = 238
		barY      = 10
		barWidth  = 220
		barHeight = 10
	)
	text.Draw(screen, "BOSS", basicfont.Face7x13, barX-38, barY+9, color.White)
	ebitenutil.DrawRect(screen, barX, barY, barWidth, barHeight, color.RGBA{R: 60, G: 20, B: 20, A: 255})
	hpWidth := barWidth * float64(max(0, g.boss.hp)) / float64(g.boss.maxHP)
	ebitenutil.DrawRect(screen, barX, barY, hpWidth, barHeight, color.RGBA{R: 230, G: 45, B: 35, A: 255})
}

func kieeCharge(missCount int) int {
	return min(specialCost, max(0, missCount))
}

func kieeGaugeFillWidth(charge, width int) float64 {
	return float64(width) * float64(kieeCharge(charge)) / specialCost
}

func (g *Game) drawPowerUp(screen *ebiten.Image, item powerUp) {
	border := color.RGBA{R: 255, G: 120, B: 35, A: 255}
	inside := color.RGBA{R: 255, G: 225, B: 65, A: 255}
	ebitenutil.DrawRect(screen, item.x, item.y, powerUpSize, powerUpSize, border)
	ebitenutil.DrawRect(screen, item.x+3, item.y+3, powerUpSize-6, powerUpSize-6, inside)
	text.Draw(screen, "P", basicfont.Face7x13, int(item.x)+7, int(item.y)+15, color.RGBA{R: 120, G: 35, B: 15, A: 255})
}

func drawImageAt(screen, img *ebiten.Image, position point) {
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Translate(position.x, position.y)
	screen.DrawImage(img, options)
}

func (g *Game) drawCenteredText(screen *ebiten.Image, message string, y int, clr color.Color) {
	_, advance := font.BoundString(g.font, message)
	x := (screenWidth - advance.Ceil()) / 2
	text.Draw(screen, message, g.font, x, y, clr)
}

func (g *Game) Layout(_, _ int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	game, err := newGame()
	if err != nil {
		log.Fatal(err)
	}

	ebiten.SetWindowSize(screenWidth*windowScale, screenHeight*windowScale)
	ebiten.SetWindowTitle("UFO撃ち落としたことありますか？")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
