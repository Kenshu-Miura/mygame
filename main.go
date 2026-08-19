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
	screenWidth      = 640
	screenHeight     = 480
	windowScale      = 2
	playerScale      = 0.1
	playerSpeed      = 4
	projectileSpeed  = 2
	enemySpeed       = 2
	enemySpeedGain   = 0.25
	maxEnemySpeed    = 6
	audioSampleRate  = 48_000
	specialCost      = 20
	shotInterval     = 10 // At 60 TPS, holding Space fires about six shots per second.
	waveDuration     = 8 * 60
	bossWaveCycle    = 5
	waveBannerTime   = 90
	bossScale        = 0.22
	bossSpeed        = 1.5
	bossY            = -18
	bossMoveMinTime  = 30
	bossMoveVariance = 90
	bossBaseHP       = 30
	bossHPGrowth     = 15
	bossAttackTime   = 75
	bossSpecialHit   = 10
	bossDefeatBonus  = 25
	comboStep        = 5
	maxComboBonus    = 5
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

type ufo struct {
	horizontalEnemy
	visible bool
}

type horizontalEnemy struct {
	point
	velocityX float64
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

	projectiles []point
	ufos        []ufo
	bashiHebis  []point
	ebis        []horizontalEnemy
	boss        *boss
	debug       bool

	score           int
	combo           int
	missCount       int
	shotCooldown    int
	wave            int
	waveTicks       int
	waveBannerTicks int
	random          *rand.Rand

	playerImage   *ebiten.Image
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
}

func newGame() (*Game, error) {
	gameFont, err := loadFont()
	if err != nil {
		return nil, fmt.Errorf("load font: %w", err)
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

	g := &Game{
		debug:         debugModeEnabled(),
		playerImage:   playerImage,
		ufoImage:      ufoImage,
		projectileImg: projectileImage,
		bashiHebiImg:  bashiHebiImage,
		ebiImage:      ebiImage,
		bossImage:     bossImage,
		font:          gameFont,
		shotSound:     shotSound,
		hitSound:      hitSound,
		kieeSound:     kieeSound,
		kieeSound2:    kieeSound2,
		hoaaSound:     hoaaSound,
		bgm:           bgm,
		gameOverSE:    gameOverSE,
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
	g.boss = nil
	g.score = 0
	g.combo = 0
	g.missCount = 0
	g.shotCooldown = 0
	g.wave = 1
	g.waveTicks = 0
	g.waveBannerTicks = waveBannerTime
	g.random = rand.New(rand.NewSource(time.Now().UnixNano()))
	g.state = stateTitle
	g.bgm.Pause()
	g.gameOverSE.Pause()
}

func (g *Game) Update() error {
	switch g.state {
	case stateTitle:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.state = statePlaying
			replay(g.bgm)
		}
		return nil
	case stateGameOver:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.reset()
		}
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.reset()
		return nil
	}

	g.handleDebugInput()
	g.updateWave()
	g.handlePlayerInput()
	g.handleProjectileCollisions()
	g.handleSpecialAttack()
	g.spawnEnemies()
	g.moveEntities()
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
}

func (g *Game) handlePlayerInput() {
	playerWidth := float64(g.playerImage.Bounds().Dx()) * playerScale
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		g.player.x = max(0, g.player.x-playerSpeed)
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		g.player.x = min(float64(screenWidth)-playerWidth, g.player.x+playerSpeed)
	}
	if g.shouldFire(ebiten.IsKeyPressed(ebiten.KeySpace)) {
		projectileWidth := float64(g.projectileImg.Bounds().Dx())
		g.projectiles = append(g.projectiles, point{
			x: g.player.x + playerWidth*playerFingerTipXRatio - projectileWidth/2,
			y: g.player.y,
		})
		replay(g.shotSound)
	}
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
	g.score += baseScore * g.comboMultiplier()
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
	if g.boss != nil {
		return
	}
	g.waveTicks++
	if g.waveTicks >= waveDuration {
		g.startWave(g.wave + 1)
	}
}

func (g *Game) startWave(wave int) {
	g.wave = wave
	g.waveTicks = 0
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
	g.score += bossDefeatBonus * g.comboMultiplier()
	g.startWave(g.wave + 1)
}

func (g *Game) handleProjectileCollisions() {
	for projectileIndex := len(g.projectiles) - 1; projectileIndex >= 0; projectileIndex-- {
		projectile := g.projectiles[projectileIndex]
		projectileRect := g.projectileImg.Bounds().Add(image.Pt(int(projectile.x), int(projectile.y)))
		hit := false
		bossDefeated := false

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
				target.visible = false
				g.recordHit(1)
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
					g.score = max(0, g.score-2)
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
	}
}

func (g *Game) handleSpecialAttack() {
	if g.missCount < specialCost || !inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		return
	}

	g.missCount -= specialCost
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
				g.recordHit(1)
			}
		}
		g.ufos = nil
	}
	g.bashiHebis = nil
	g.ebis = nil
	g.projectiles = nil
	replay(g.kieeSound)
	replay(g.kieeSound2)
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
	fallingEnemySpeed := 1 + float64(g.wave-1)*0.15
	for index := range g.bashiHebis {
		g.bashiHebis[index].y += fallingEnemySpeed
	}
	for index := range g.ebis {
		g.ebis[index].x += g.ebis[index].velocityX
	}
	for index := range g.projectiles {
		g.projectiles[index].y -= projectileSpeed
	}
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
	const padding = 30
	playerRect := image.Rect(
		int(g.player.x)+padding,
		int(g.player.y)+padding,
		int(g.player.x)+int(float64(g.playerImage.Bounds().Dx())*playerScale)-padding,
		int(g.player.y)+int(float64(g.playerImage.Bounds().Dy())*playerScale)-padding,
	)

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
		if g.projectiles[index].y+float64(g.projectileImg.Bounds().Dy()) < 0 {
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
}

func removeAt[T any](values []T, index int) []T {
	copy(values[index:], values[index+1:])
	var zero T
	values[len(values)-1] = zero
	return values[:len(values)-1]
}

func (g *Game) Draw(screen *ebiten.Image) {
	switch g.state {
	case stateTitle:
		g.drawTitle(screen)
		return
	case stateGameOver:
		g.drawGame(screen)
		g.drawCenteredText(screen, "GAME OVER", screenHeight/2, color.White)
		g.drawCenteredText(screen, "Escキーでタイトルに戻る", screenHeight/2+40, color.White)
		return
	default:
		g.drawGame(screen)
	}
}

func (g *Game) drawTitle(screen *ebiten.Image) {
	g.drawCenteredText(screen, "UFO撃ち落としたことありますか？", screenHeight/2-34, color.White)
	g.drawCenteredText(screen, "Spaceキーでスタート", screenHeight/2+6, color.White)
	g.drawCenteredText(screen, "5ウェーブごとに巨大海老ボス出現！", screenHeight/2+46, color.White)
	g.drawCenteredText(screen, "連続命中でコンボ倍率アップ", screenHeight/2+86, color.White)
	if g.debug {
		g.drawCenteredText(screen, "DEBUG MODE: 無敵 / B:ボス / K:KIEE", screenHeight/2+126, color.RGBA{R: 255, G: 210, B: 60, A: 255})
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
		drawImageAt(screen, g.projectileImg, projectile)
	}

	g.drawHUD(screen)
	if g.waveBannerTicks > 0 {
		message := fmt.Sprintf("WAVE %d", g.wave)
		if isBossWave(g.wave) {
			message = fmt.Sprintf("BOSS WAVE %d", g.wave)
		}
		g.drawCenteredText(screen, message, 110, color.White)
	}
}

func (g *Game) drawHUD(screen *ebiten.Image) {
	text.Draw(screen, fmt.Sprintf("Score: %d", g.score), basicfont.Face7x13, 1, 12, color.White)
	text.Draw(screen, fmt.Sprintf("Wave: %d", g.wave), basicfont.Face7x13, 1, 25, color.White)
	text.Draw(screen, fmt.Sprintf("KIEE: %d/%d", min(g.missCount, specialCost), specialCost), basicfont.Face7x13, 1, 38, color.White)
	text.Draw(screen, fmt.Sprintf("Combo: %d  x%d", g.combo, g.comboMultiplier()), basicfont.Face7x13, 1, 51, color.White)
	if g.debug {
		text.Draw(screen, "DEBUG: INVINCIBLE  B:BOSS  K:KIEE", basicfont.Face7x13, 1, screenHeight-4, color.RGBA{R: 255, G: 210, B: 60, A: 255})
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
