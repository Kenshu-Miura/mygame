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
	screenWidth     = 640
	screenHeight    = 480
	windowScale     = 2
	playerScale     = 0.1
	playerSpeed     = 4
	projectileSpeed = 2
	enemySpeed      = 2
	audioSampleRate = 48_000
	specialCost     = 20
)

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
	point
	visible bool
}

type Game struct {
	player point
	state  gameState

	projectiles []point
	ufos        []ufo
	bashiHebis  []point
	ebis        []point

	score          int
	missCount      int
	bashiHebiSpeed float64
	spawnThreshold float64
	random         *rand.Rand

	playerImage   *ebiten.Image
	ufoImage      *ebiten.Image
	projectileImg *ebiten.Image
	bashiHebiImg  *ebiten.Image
	ebiImage      *ebiten.Image
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
		playerImage:   playerImage,
		ufoImage:      ufoImage,
		projectileImg: projectileImage,
		bashiHebiImg:  bashiHebiImage,
		ebiImage:      ebiImage,
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
	g.score = 0
	g.missCount = 0
	g.bashiHebiSpeed = 1
	g.spawnThreshold = 0.001
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

	g.handlePlayerInput()
	g.handleProjectileCollisions()
	g.handleSpecialAttack()
	g.spawnEnemies()
	g.moveEntities()
	g.handlePlayerCollision()
	g.removeOffscreenEntities()
	return nil
}

func (g *Game) handlePlayerInput() {
	playerWidth := float64(g.playerImage.Bounds().Dx()) * playerScale
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		g.player.x = max(0, g.player.x-playerSpeed)
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		g.player.x = min(float64(screenWidth)-playerWidth, g.player.x+playerSpeed)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		projectileWidth := float64(g.projectileImg.Bounds().Dx())
		g.projectiles = append(g.projectiles, point{
			x: g.player.x + playerWidth/2 - projectileWidth/2,
			y: g.player.y,
		})
		replay(g.shotSound)
	}
}

func (g *Game) handleProjectileCollisions() {
	for projectileIndex := len(g.projectiles) - 1; projectileIndex >= 0; projectileIndex-- {
		projectile := g.projectiles[projectileIndex]
		projectileRect := g.projectileImg.Bounds().Add(image.Pt(int(projectile.x), int(projectile.y)))
		hit := false

		for ufoIndex := range g.ufos {
			target := &g.ufos[ufoIndex]
			if !target.visible {
				continue
			}
			targetRect := g.ufoImage.Bounds().Add(image.Pt(int(target.x), int(target.y)))
			if projectileRect.Overlaps(targetRect) {
				target.visible = false
				g.score++
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
	}
}

func (g *Game) handleSpecialAttack() {
	if g.missCount < specialCost || !inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		return
	}

	g.missCount -= specialCost
	for _, target := range g.ufos {
		if target.visible && target.x+float64(g.ufoImage.Bounds().Dx()) >= 0 {
			g.score++
		}
	}
	g.ufos = nil
	g.bashiHebis = nil
	g.ebis = nil
	g.projectiles = nil
	replay(g.kieeSound)
	replay(g.kieeSound2)
}

func (g *Game) spawnEnemies() {
	if g.random.Intn(120) < 2 {
		g.ufos = append(g.ufos, ufo{
			point:   point{x: screenWidth, y: float64(g.random.Intn(screenHeight / 2))},
			visible: true,
		})
	}

	g.spawnThreshold += 0.0005
	if g.random.Intn(165) < int(g.spawnThreshold) {
		g.bashiHebis = append(g.bashiHebis, point{x: float64(g.random.Intn(screenWidth)), y: 0})
	}

	if g.random.Intn(130) < 1 {
		g.ebis = append(g.ebis, point{x: screenWidth, y: float64(g.random.Intn(screenHeight / 2))})
	}
}

func (g *Game) moveEntities() {
	g.bashiHebiSpeed += 0.001
	for index := range g.ufos {
		g.ufos[index].x -= enemySpeed
	}
	for index := range g.bashiHebis {
		g.bashiHebis[index].y += g.bashiHebiSpeed
	}
	for index := range g.ebis {
		g.ebis[index].x -= enemySpeed
	}
	for index := range g.projectiles {
		g.projectiles[index].y -= projectileSpeed
	}
}

func (g *Game) handlePlayerCollision() {
	const padding = 30
	playerRect := image.Rect(
		int(g.player.x)+padding,
		int(g.player.y)+padding,
		int(g.player.x)+int(float64(g.playerImage.Bounds().Dx())*playerScale)-padding,
		int(g.player.y)+int(float64(g.playerImage.Bounds().Dy())*playerScale)-padding,
	)

	for _, enemy := range g.bashiHebis {
		enemyRect := g.bashiHebiImg.Bounds().Add(image.Pt(int(enemy.x), int(enemy.y)))
		if enemyRect.Overlaps(playerRect) {
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
		if !target.visible || target.x+float64(g.ufoImage.Bounds().Dx()) < 0 {
			g.ufos = removeAt(g.ufos, index)
		}
	}

	for index := len(g.ebis) - 1; index >= 0; index-- {
		if g.ebis[index].x+float64(g.ebiImage.Bounds().Dx()) < 0 {
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
	g.drawCenteredText(screen, "KIEE Countが20以上の時に↑を押すと…", screenHeight/2+46, color.White)
}

func (g *Game) drawGame(screen *ebiten.Image) {
	playerOptions := &ebiten.DrawImageOptions{}
	playerOptions.GeoM.Scale(playerScale, playerScale)
	playerOptions.GeoM.Translate(g.player.x, g.player.y)
	screen.DrawImage(g.playerImage, playerOptions)

	ebitenutil.DebugPrint(screen, fmt.Sprintf("Score: %d", g.score))
	for _, projectile := range g.projectiles {
		drawImageAt(screen, g.projectileImg, projectile)
	}
	for _, target := range g.ufos {
		if target.visible {
			drawImageAt(screen, g.ufoImage, target.point)
		}
	}
	for _, enemy := range g.bashiHebis {
		drawImageAt(screen, g.bashiHebiImg, enemy)
	}
	for _, target := range g.ebis {
		drawImageAt(screen, g.ebiImage, target)
	}

	text.Draw(screen, fmt.Sprintf("KIEE Count: %d", g.missCount), basicfont.Face7x13, 1, 23, color.White)
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
