package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"

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
	x, y                                              float64
	oses                                              []*O
	sound, hitSound, kieeSound, kieeSound2, hoaaSound *audio.Player
	ufos                                              []*UFO
	score                                             int
	bashiHebis                                        []*BashiHebi
	gameOver                                          bool
	oOutsideCount                                     int
	state                                             string
	bashiHebiSpeed                                    float64
	ebis                                              []*Ebi
	spawnRate                                         float64
	r                                                 *rand.Rand
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

type Ebi struct {
	x, y float64
}

func (g *Game) resetGame() {
	g.x = float64(screenWidth)/2 - float64(img.Bounds().Dx())*scale/2
	g.y = float64(screenHeight)*1.1 - float64(img.Bounds().Dy())*scale
	g.oses = []*O{}
	g.ufos = []*UFO{} // Reset the ufos slice
	g.score = 0       // Reset the score
	g.bashiHebis = []*BashiHebi{}
	g.state = "title"
	g.ebis = []*Ebi{}
	g.bashiHebiSpeed = 1
	g.spawnRate = 0.001
	g.r = rand.New(rand.NewSource(time.Now().UnixNano()))
}

var (
	img, ufoImg, oImg, bashiHebiImg, ebiImg *ebiten.Image
	game                                    = &Game{}
	audioContext                            = audio.NewContext(48000)
	mplusNormalFont                         font.Face
	bgmPlayer                               *audio.Player
	majidePlayer                            *audio.Player
	majidePlayed                            bool
)

func init() {
	tt, err := opentype.Parse(fonts.MPlus1pRegular_ttf)
	if err != nil {
		log.Fatal(err)
	}

	const dpi = 72
	mplusNormalFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    24,
		DPI:     dpi,
		Hinting: font.HintingVertical,
	})
	if err != nil {
		log.Fatal(err)
	}

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

	ebiImg, err = loadImage("ebi.png")
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

	game.kieeSound, err = loadSound("kiee.wav")
	if err != nil {
		log.Fatal(err)
	}

	game.kieeSound2, err = loadSound("kiee2.wav")
	if err != nil {
		log.Fatal(err)
	}

	game.hoaaSound, err = loadSound("hoaa.wav")
	if err != nil {
		log.Fatal(err)
	}

	bgmPlayer, err = loadSound("BGM.wav")
	if err != nil {
		log.Fatal(err)
	}

	majidePlayer, err = loadSound("majide.wav") // majide.wavを読み込む
	if err != nil {
		log.Fatal(err)
	}

	game.resetGame()
}

func loadImage(filePath string) (*ebiten.Image, error) {
	img, _, err := ebitenutil.NewImageFromFile(filePath)
	return img, err
}

func loadSound(filePath string) (*audio.Player, error) {
	file, err := os.Open(filePath)
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
	if g.state == "title" {
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.state = "game"
			bgmPlayer.Rewind() // BGMを最初から再生する
			bgmPlayer.Play()   // BGMを再生する
		}
		return nil
	}

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
	if !g.gameOver && g.oOutsideCount >= 20 && inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		g.oOutsideCount -= 20
		ufoCount := 0
		for _, ufo := range g.ufos {
			if ufo.visible && ufo.x+float64(ufoImg.Bounds().Dx()) >= 0 { // Check if UFO is on screen
				ufo.visible = false
				ufoCount++
			}
		}
		g.kieeSound.Rewind()
		g.kieeSound.Play()
		g.kieeSound2.Rewind()
		g.kieeSound2.Play()
		g.score += ufoCount
		g.ufos = nil       // Clear the UFO slice
		g.bashiHebis = nil // Clear the BashiHebi slice
		g.ebis = nil       // Clear the Ebi slice
		g.oses = nil       // Clear the O slice
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
		bgmPlayer.Pause()
		g.gameOver = false // 追加
		g.oOutsideCount = 0
	}

	if g.r.Intn(120) < 2 {
		g.ufos = append(g.ufos, &UFO{x: startingUfoX, y: float64(g.r.Intn(screenHeight / 2)), visible: true})
	}

	// UFOとbashihebiの動きを更新する前にgameOverフラグをチェック
	if !g.gameOver {
		for _, ufo := range g.ufos {
			ufo.x -= 2
		}
		if g.state == "game" {
			g.bashiHebiSpeed += 0.001 // あるいは適切な値に設定
		}

		for _, bashihebi := range g.bashiHebis {
			bashihebi.y += g.bashiHebiSpeed
		}
	}

	for _, o := range g.oses {
		if !g.gameOver {
			o.y -= 2 // これにより"o.png"が上に移動します
		}
	}

	if !g.gameOver {
		g.spawnRate += 0.0005 // あるいは適切な値に設定
		if g.r.Intn(165) < int(g.spawnRate) {
			g.bashiHebis = append(g.bashiHebis, &BashiHebi{x: float64(g.r.Intn(screenWidth)), y: 0})
		}
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

	for i := len(g.oses) - 1; i >= 0; i-- {
		if g.oses[i].y < 0 {
			g.oOutsideCount++
			g.oses = append(g.oses[:i], g.oses[i+1:]...)
		}
	}

	if g.gameOver {
		bgmPlayer.Pause() // BGMの再生を停止する
		if !majidePlayed {
			majidePlayer.Rewind() // majidePlayerを巻き戻す
			majidePlayer.Play()   // majide.wavを再生する
			majidePlayed = true   // majide.wavが再生されたことを記録する
		}
		return nil
	} else {
		majidePlayed = false // gameOverがfalseの場合、majidePlayedをリセットする
	}

	for i := len(g.ufos) - 1; i >= 0; i-- {
		if g.ufos[i].x+float64(ufoImg.Bounds().Dx()) < 0 || g.ufos[i].x > float64(screenWidth) {
			g.ufos = append(g.ufos[:i], g.ufos[i+1:]...)
		}
	}

	for i := len(g.oses) - 1; i >= 0; i-- {
		if g.oses[i].y+float64(oImg.Bounds().Dy()) < 0 || g.oses[i].y > float64(screenHeight) {
			g.oses = append(g.oses[:i], g.oses[i+1:]...)
		}
	}

	for i := len(g.bashiHebis) - 1; i >= 0; i-- {
		if g.bashiHebis[i].y+float64(bashiHebiImg.Bounds().Dy()) < 0 || g.bashiHebis[i].y > float64(screenHeight) {
			g.bashiHebis = append(g.bashiHebis[:i], g.bashiHebis[i+1:]...)
		}
	}

	if g.r.Intn(130) < 1 {
		g.ebis = append(g.ebis, &Ebi{x: float64(screenWidth), y: float64(g.r.Intn(screenHeight / 2))})
	}
	for _, ebi := range g.ebis {
		ebi.x -= 2
	}
	for i := len(g.ebis) - 1; i >= 0; i-- {
		if g.ebis[i].x+float64(ebiImg.Bounds().Dx()) < 0 {
			g.ebis = append(g.ebis[:i], g.ebis[i+1:]...)
		}
	}
	for oIndex, o := range g.oses {
		for ebiIndex, ebi := range g.ebis {
			if o.x >= ebi.x && o.x <= ebi.x+float64(ebiImg.Bounds().Dx()) &&
				o.y >= ebi.y && o.y <= ebi.y+float64(ebiImg.Bounds().Dy()) {
				// 当たり判定
				g.oses = append(g.oses[:oIndex], g.oses[oIndex+1:]...)
				g.ebis = append(g.ebis[:ebiIndex], g.ebis[ebiIndex+1:]...)
				g.hoaaSound.Rewind()
				g.hoaaSound.Play()
				g.hitSound.Rewind()
				g.hitSound.Play()
				g.score -= 2 // スコアを減点
				if g.score < 0 {
					g.score = 0
				}
				break
			}
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.state == "title" {
		textHeight := 24                   // または適切なテキストの高さに設定
		y := screenHeight/2 + textHeight/2 // これでyは "Press Enter to Start" のテキストのy座標になります。

		japaneseText := "UFO撃ち落としたことありますか？"
		_, advance := font.BoundString(mplusNormalFont, japaneseText)
		japaneseTextWidth := advance.Ceil()
		xJapanese := (screenWidth - japaneseTextWidth) / 2
		yJapanese := y - textHeight - 10 // 10 is a padding between the two texts
		text.Draw(screen, japaneseText, mplusNormalFont, xJapanese, yJapanese, color.White)

		titleText := "Press Space to Start"
		_, advance = font.BoundString(mplusNormalFont, titleText)
		textWidth := advance.Ceil()
		textHeight = mplusNormalFont.Metrics().Height.Ceil()
		x := (screenWidth - textWidth) / 2
		y = (screenHeight-textHeight)/2 + textHeight // textHeight is added to y to align the text properly
		text.Draw(screen, titleText, mplusNormalFont, x, y, color.White)

		infoText := "KIEE Countが20以上の時に↑を押すと…"
		_, advanceInfo := font.BoundString(mplusNormalFont, infoText)
		infoTextWidth := advanceInfo.Ceil()
		xInfo := (screenWidth - infoTextWidth) / 2
		yInfo := y + textHeight + 10 // 10 is a padding between the texts
		text.Draw(screen, infoText, mplusNormalFont, xInfo, yInfo, color.White)

		return
	}

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

	for _, ebi := range g.ebis {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(ebi.x, ebi.y)
		screen.DrawImage(ebiImg, op)
	}

	if g.gameOver {
		text.Draw(screen, "GAME OVER", mplusNormalFont, screenWidth/2-50, screenHeight/2, color.White)
		instructionText := "To restart the game, press the Esc key."
		_, advanceInstruction := font.BoundString(mplusNormalFont, instructionText)
		instructionWidth := advanceInstruction.Ceil()
		xInstruction := (screenWidth - instructionWidth) / 2
		yInstruction := screenHeight/2 + 40 // 40はテキスト間のパディングです
		text.Draw(screen, instructionText, mplusNormalFont, xInstruction, yInstruction, color.White)
		return
	}

	countText := fmt.Sprintf("KIEE Count: %d", g.oOutsideCount)
	text.Draw(screen, countText, basicfont.Face7x13, 1, 23, color.White)
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
