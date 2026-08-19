package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type touchAction uint8

const (
	touchActionNone touchAction = iota
	touchActionShot
	touchActionSpecial
)

type touchGesture struct {
	active             bool
	id                 ebiten.TouchID
	startX             int
	startY             int
	lastX              int
	lastY              int
	maxDistanceSquared int
}

func (gesture *touchGesture) begin(id ebiten.TouchID, x, y int) {
	*gesture = touchGesture{
		active: true,
		id:     id,
		startX: x,
		startY: y,
		lastX:  x,
		lastY:  y,
	}
}

func (gesture *touchGesture) track(x, y int) int {
	deltaX := x - gesture.lastX
	gesture.lastX = x
	gesture.lastY = y
	distanceX := x - gesture.startX
	distanceY := y - gesture.startY
	gesture.maxDistanceSquared = max(gesture.maxDistanceSquared, distanceX*distanceX+distanceY*distanceY)
	return deltaX
}

func (gesture *touchGesture) finish(x, y int) (int, touchAction) {
	deltaX := gesture.track(x, y)
	totalX := x - gesture.startX
	upwardDistance := gesture.startY - y
	gesture.active = false

	if upwardDistance >= touchSpecialDistance && upwardDistance > intAbs(totalX) {
		return deltaX, touchActionSpecial
	}
	if gesture.maxDistanceSquared <= touchTapDistance*touchTapDistance {
		return deltaX, touchActionShot
	}
	return deltaX, touchActionNone
}

func (g *Game) handleTouchInput() {
	if !g.touch.active {
		ids := inpututil.AppendJustPressedTouchIDs(nil)
		if len(ids) == 0 {
			return
		}
		x, y := ebiten.TouchPosition(ids[0])
		g.touch.begin(ids[0], x, y)
		return
	}

	if inpututil.IsTouchJustReleased(g.touch.id) {
		x, y := inpututil.TouchPositionInPreviousTick(g.touch.id)
		deltaX, action := g.touch.finish(x, y)
		g.movePlayerHorizontally(float64(deltaX))
		g.touchShot = action == touchActionShot
		g.touchSpecial = action == touchActionSpecial
		return
	}

	x, y := ebiten.TouchPosition(g.touch.id)
	g.movePlayerHorizontally(float64(g.touch.track(x, y)))
}

func touchJustPressed() bool {
	return len(inpututil.AppendJustPressedTouchIDs(nil)) > 0
}

func intAbs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
