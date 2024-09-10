package main

import (
	"math"

	"github.com/pwiecz/go-fltk"
)

// If the screen is portrait or landscape, the window will be scaled
// accordingly.
const (
	WIDTH_PORTRAIT   = 100
	HEIGHT_PORTRAIT  = 150
	WIDTH_LANDSCAPE  = 150
	HEIGHT_LANDSCAPE = 100
)

// Positioning (x,y,w,h) for fltk elements
type Pos struct {
	X int
	Y int
	W int
	H int
}

// Translates the widget's width/height from the original 100 or 150px base
// width/height to the window's current width/height
func tr(i int, winW int, winH int, useHeight bool) int {
	if portrait {
		if useHeight {
			return int(math.Round((float64(i) / float64(HEIGHT_PORTRAIT)) * float64(winH)))
		} else {
			return int(math.Round((float64(i) / float64(WIDTH_PORTRAIT)) * float64(winW)))
		}
	} else {
		if useHeight {
			return int(math.Round((float64(i) / float64(HEIGHT_LANDSCAPE)) * float64(winH)))
		} else {
			return int(math.Round((float64(i) / float64(WIDTH_LANDSCAPE)) * float64(winW)))
		}
	}
}

// Translate converts a predefined position into a scaled position based on
// the latest width & height of the window.
func (p *Pos) Translate(winW, winH int) {
	p.X = tr(p.X, winW, winH, false)
	p.Y = tr(p.Y, winW, winH, true)
	p.W = tr(p.W, winW, winH, false)
	p.H = tr(p.H, winW, winH, true)
}

// Resizes and repositions all components based on the window's size.
func responsive(win *fltk.Window) {
	if forceLandscape || forcePortrait {
		return
	}

	winW := win.W()
	winH := win.H()

	if winW > winH {
		portrait = false
	} else {
		portrait = true
	}

	getDevsBtnPos := Pos{X: 5, Y: 85, W: 35, H: 10}
	directoryBtnPos := Pos{X: 45, Y: 85, W: 50, H: 10}
	scanBtnPos := Pos{X: 100, Y: 85, W: 45, H: 10}
	devicesChoicePos := Pos{X: 5, Y: 5, W: 140, H: 15}
	optsChoicePos := Pos{X: 5, Y: 25, W: 60, H: 15}
	constChoicePos := Pos{X: 75, Y: 25, W: 70, H: 15}
	fileTmplInputPos := Pos{X: 5, Y: 45, W: 140, H: 10}
	activityPos := Pos{X: 5, Y: 60, W: 140, H: 20}

	if portrait {
		getDevsBtnPos = Pos{X: 5, Y: 105, W: 90, H: 10}
		directoryBtnPos = Pos{X: 5, Y: 120, W: 90, H: 10}
		scanBtnPos = Pos{X: 5, Y: 135, W: 90, H: 10}
		devicesChoicePos = Pos{X: 5, Y: 5, W: 90, H: 15}
		optsChoicePos = Pos{X: 5, Y: 25, W: 90, H: 15}
		constChoicePos = Pos{X: 5, Y: 45, W: 90, H: 15}
		fileTmplInputPos = Pos{X: 5, Y: 65, W: 90, H: 10}
		activityPos = Pos{X: 5, Y: 80, W: 90, H: 20}
	}

	getDevsBtnPos.Translate(winW, winH)
	directoryBtnPos.Translate(winW, winH)
	scanBtnPos.Translate(winW, winH)
	devicesChoicePos.Translate(winW, winH)
	optsChoicePos.Translate(winW, winH)
	constChoicePos.Translate(winW, winH)
	fileTmplInputPos.Translate(winW, winH)
	activityPos.Translate(winW, winH)

	getDevicesBtn.Resize(getDevsBtnPos.X, getDevsBtnPos.Y, getDevsBtnPos.W, getDevsBtnPos.H)
	directoryBtn.Resize(directoryBtnPos.X, directoryBtnPos.Y, directoryBtnPos.W, directoryBtnPos.H)
	scanBtn.Resize(scanBtnPos.X, scanBtnPos.Y, scanBtnPos.W, scanBtnPos.H)
	devicesChoice.Resize(devicesChoicePos.X, devicesChoicePos.Y, devicesChoicePos.W, devicesChoicePos.H)
	optChoice.Resize(optsChoicePos.X, optsChoicePos.Y, optsChoicePos.W, optsChoicePos.H)
	constChoice.Resize(constChoicePos.X, constChoicePos.Y, constChoicePos.W, constChoicePos.H)
	fileTmplInput.Resize(fileTmplInputPos.X, fileTmplInputPos.Y, fileTmplInputPos.W, fileTmplInputPos.H)
	activity.Resize(activityPos.X, activityPos.Y, activityPos.W, activityPos.H)
}
