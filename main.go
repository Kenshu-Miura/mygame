package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	scale        = 0.1
	startingUfoX = 640
	startingUfoY = 0
	speed        = 4
	screenWidth  = 640
	screenHeight = 480
)

type Game struct {
	x, y            float64
	oses            []*O
	sound, hitSound *audio.Player
	ufos            []*UFO
	score           int
}

type O struct {
	x, y float64
}

type UFO struct {
	x, y    float64
	visible bool
}

func (g *Game) resetGame() {
	g.x = float64(screenWidth)/2 - float64(img.Bounds().Dx())*scale/2
	g.y = float64(screenHeight)*1.1 - float64(img.Bounds().Dy())*scale
	g.oses = []*O{}
	g.ufos = []*UFO{} // Reset the ufos slice
	g.score = 0       // Reset the score
}

var (
	img, ufoImg, oImg *ebiten.Image
	game              = &Game{}
	audioContext      = audio.NewContext(48000)
)

func init() {
	var err error
	img, err = loadImage("ebisan.png")
	if err != nil {
		log.Fatal(err)
	}

	ufoImg, err = loadImage("ufo.png")
	if err != nil {
		log.Fatal(err)
	}

	oImg, err = loadImage("o.png")
	if err != nil {
		log.Fatal(err)
	}

	game.sound, err = loadSound("shot.wav")
	if err != nil {
		log.Fatal(err)
	}

	game.hitSound, err = loadSound("hit.wav")
	if err != nil {
		log.Fatal(err)
	}

	rand.Seed(time.Now().UnixNano())

	game.resetGame()
}

func loadImage(filePath string) (*ebiten.Image, error) {
	img, _, err := ebitenutil.NewImageFromFile(filePath)
	return img, err
}

func loadSound(filePath string) (*audio.Player, error) {
	file, err := ebitenutil.OpenFile(filePath)
	if err != nil {
		return nil, err
	}

	soundStream, err := wav.DecodeWithSampleRate(48000, file)
	if err != nil {
		return nil, err
	}

	return audioContext.NewPlayer(soundStream)
}

func (g *Game) Update() error {
	const speed = 4
	if ebiten.IsKeyPressed(ebiten.KeyLeft) && g.x > 0 {
		g.x -= speed
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) && g.x < float64(640)-float64(img.Bounds().Dx())*scale {
		g.x += speed
	}

	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.sound.Rewind()
		g.sound.Play()
		g.oses = append(g.oses, &O{x: g.x + float64(img.Bounds().Dx())*scale/2, y: g.y})
	}
	for oIndex, o := range g.oses {
		for _, ufo := range g.ufos {
			if ufo.visible && o.x >= ufo.x && o.x <= ufo.x+float64(ufoImg.Bounds().Dx()) && o.y >= ufo.y && o.y <= ufo.y+float64(ufoImg.Bounds().Dy()) {
				ufo.visible = false
				g.hitSound.Rewind()
				g.hitSound.Play()
				g.score++

				// Remove the "o" object from oses slice
				g.oses = append(g.oses[:oIndex], g.oses[oIndex+1:]...)
				break // Break out of the inner loop as the o object has been removed
			}
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.resetGame()
	}

	if rand.Intn(100) < 1 {
		g.ufos = append(g.ufos, &UFO{x: startingUfoX, y: float64(rand.Intn(screenHeight / 2)), visible: true})
	}

	for _, ufo := range g.ufos {
		ufo.x -= 2
	}

	for _, o := range g.oses {
		o.y -= 2 // これにより"o.png"が上に移動します
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate(g.x, g.y)
	screen.DrawImage(img, opts)
	ebitenutil.DebugPrint(screen, fmt.Sprintf("Score: %d", g.score))
	for _, o := range g.oses {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(o.x, o.y)
		screen.DrawImage(oImg, opts)
	}

	for _, ufo := range g.ufos {
		if ufo.visible {
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Translate(ufo.x, ufo.y)
			screen.DrawImage(ufoImg, opts)
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 640, 480
}

func main() {
	ebiten.SetWindowSize(1280, 960)
	ebiten.SetWindowTitle("Hello, Ebisan!")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
