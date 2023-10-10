package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	scale        = 0.1
	startingUfoX = 640 // Starting from the right edge of the screen
	startingUfoY = 0   // Starting from the top of the screen
	speed        = 4
)

var (
	img, ufoImg, oImg         *ebiten.Image
	x, y, ufoX, ufoY          float64
	oses                      []*O
	sound, hitSound           *audio.Player
	ufoVisible                bool = true
	audioContext                   = audio.NewContext(48000)
	game                      *Game
	screenWidth, screenHeight int
)

type Game struct{}

type O struct {
	x, y float64
}

func resetGame() {
	screenWidth, screenHeight = 640, 480 // これらの変数の初期値を設定します
	x = float64(screenWidth)/2 - float64(img.Bounds().Dx())*scale/2
	y = float64(screenHeight)*1.1 - float64(img.Bounds().Dy())*scale
	ufoX = float64(640)
	ufoY = 0
	oses = []*O{}
	ufoVisible = true
}

func init() {
	var err error
	img, err = loadImage("ebisan.png")
	checkError(err)

	ufoImg, err = loadImage("ufo.png")
	checkError(err)

	oImg, err = loadImage("o.png")
	checkError(err)

	sound, err = loadSound("shot.wav")
	checkError(err)

	hitSound, err = loadSound("hit.wav")
	checkError(err)

	// Set the initial position of the UFO
	ufoX = startingUfoX
	ufoY = startingUfoY

	resetGame()
}

func loadImage(filePath string) (*ebiten.Image, error) {
	img, _, err := ebitenutil.NewImageFromFile(filePath)
	return img, err
}

func loadSound(filePath string) (*audio.Player, error) {
	soundFile, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer soundFile.Close()

	soundData, err := ioutil.ReadAll(soundFile)
	if err != nil {
		return nil, err
	}

	soundBuffer := bytes.NewReader(soundData)
	soundStream, err := wav.DecodeWithSampleRate(48000, soundBuffer)
	if err != nil {
		return nil, err
	}

	return audioContext.NewPlayer(soundStream)
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func (g *Game) Update() error {
	const speed = 4
	if ebiten.IsKeyPressed(ebiten.KeyLeft) && x > 0 {
		x -= speed
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) && x < float64(640)-float64(img.Bounds().Dx())*scale {
		x += speed
	}

	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		sound.Rewind()
		sound.Play()
		oses = append(oses, &O{x: x + float64(img.Bounds().Dx())*scale/2, y: y})
	}
	for _, o := range oses {
		o.y -= 2
		// Check for collision between "o" and the UFO
		if ufoVisible && o.x >= ufoX && o.x <= ufoX+float64(ufoImg.Bounds().Dx()) && o.y >= ufoY && o.y <= ufoY+float64(ufoImg.Bounds().Dy()) {
			ufoVisible = false // Hide the UFO if a collision is detected
			hitSound.Rewind()  // Rewind the sound to the beginning
			hitSound.Play()    // Play the hit sound
		}
	}
	// Update the position of the UFO
	ufoX -= 2 // Move the UFO to the left

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		resetGame() // Escキーが押されたときにresetGame関数を呼び出します
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate(x, y)
	screen.DrawImage(img, opts)
	for _, o := range oses {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(o.x, o.y)
		screen.DrawImage(oImg, opts) // Draw the "o" image instead of using DebugPrintAt
	}
	// Draw the UFO
	if ufoVisible {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(ufoX, ufoY)
		screen.DrawImage(ufoImg, opts)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 640, 480
}

func main() {
	ebiten.SetWindowSize(1280, 960)
	ebiten.SetWindowTitle("Hello, Ebisan!")
	if err := ebiten.RunGame(game); err != nil { // gameインスタンスを使ってゲームを実行します
		log.Fatal(err)
	}
}
