package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"
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
	bashiHebis      []*BashiHebi
	gameOver        bool
}

type O struct {
	x, y float64
}

type UFO struct {
	x, y    float64
	visible bool
}

type BashiHebi struct {
	x, y float64
}

func (g *Game) resetGame() {
	g.x = float64(screenWidth)/2 - float64(img.Bounds().Dx())*scale/2
	g.y = float64(screenHeight)*1.1 - float64(img.Bounds().Dy())*scale
	g.oses = []*O{}
	g.ufos = []*UFO{} // Reset the ufos slice
	g.score = 0       // Reset the score
	g.bashiHebis = []*BashiHebi{}
}

var (
	img, ufoImg, oImg, bashiHebiImg *ebiten.Image
	game                            = &Game{}
	audioContext                    = audio.NewContext(48000)
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

	bashiHebiImg, err = loadImage("bashihebi.png")
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
	if !g.gameOver {
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
		g.gameOver = false // 追加
	}

	if rand.Intn(100) < 2 {
		g.ufos = append(g.ufos, &UFO{x: startingUfoX, y: float64(rand.Intn(screenHeight / 2)), visible: true})
	}

	// UFOとbashihebiの動きを更新する前にgameOverフラグをチェック
	if !g.gameOver {
		for _, ufo := range g.ufos {
			ufo.x -= 2
		}
		for _, bashihebi := range g.bashiHebis {
			bashihebi.y += 2
		}
	}

	for _, o := range g.oses {
		o.y -= 2 // これにより"o.png"が上に移動します
	}

	if !g.gameOver && rand.Intn(200) < 1 { // 1%の確率で新しいBashiHebiを生成
		g.bashiHebis = append(g.bashiHebis, &BashiHebi{x: float64(rand.Intn(screenWidth)), y: 0})
	}

	for _, bh := range g.bashiHebis {
		if bh.x >= g.x && bh.x <= g.x+float64(img.Bounds().Dx())*scale &&
			bh.y >= g.y && bh.y <= g.y+float64(img.Bounds().Dy())*scale {
			g.gameOver = true
		}
	}

	for i := len(g.bashiHebis) - 1; i >= 0; i-- {
		if g.bashiHebis[i].y > float64(screenHeight) {
			g.bashiHebis = append(g.bashiHebis[:i], g.bashiHebis[i+1:]...)
		}
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

	for _, bh := range g.bashiHebis {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(bh.x, bh.y)
		screen.DrawImage(bashiHebiImg, opts)
	}

	if g.gameOver {
		text.Draw(screen, "GAME OVER", basicfont.Face7x13, screenWidth/2-50, screenHeight/2, color.White)
		return
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
