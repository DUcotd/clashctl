package ui

import "github.com/charmbracelet/bubbles/viewport"

const (
	minViewportWidth  = 24
	minViewportHeight = 6
	viewportPadX      = 4
	viewportPadY      = 2
)

func calcViewportSize(width, height, topChrome int) (int, int) {
	innerWidth := max(minViewportWidth, width-BoxStyle.GetHorizontalFrameSize()-viewportPadX)
	innerHeight := max(minViewportHeight, height-topChrome-BoxStyle.GetVerticalFrameSize()-viewportPadY)
	return innerWidth, innerHeight
}

// viewportState holds shared scrollable viewport state.
type viewportState struct {
	vp            viewport.Model
	vpReady       bool
	screenOffsets map[int]int
}

func (v *viewportState) initViewport() {
	if !v.vpReady {
		v.vp = viewport.New(1, 1)
		v.vpReady = true
	}
}

func (v *viewportState) switchScreen(from, to int, width, height, topChrome int) {
	if v.vpReady {
		v.screenOffsets[from] = v.vp.YOffset
	}
	v.initViewport()
	innerWidth, innerHeight := calcViewportSize(width, height, topChrome)
	v.vp.Width = innerWidth
	v.vp.Height = innerHeight
	if off, ok := v.screenOffsets[to]; ok {
		v.vp.SetYOffset(off)
	}
}

func (v *viewportState) ensureSize(width, height, topChrome int) {
	v.initViewport()
	innerWidth, innerHeight := calcViewportSize(width, height, topChrome)
	v.vp.Width = innerWidth
	v.vp.Height = innerHeight
}

func (v *viewportState) scroll(key string) {
	if !v.vpReady {
		return
	}
	switch key {
	case "up", "k":
		v.vp.LineUp(1)
	case "down", "j":
		v.vp.LineDown(1)
	case "pgup":
		v.vp.HalfViewUp()
	case "pgdown":
		v.vp.HalfViewDown()
	case "home":
		v.vp.GotoTop()
	case "end":
		v.vp.GotoBottom()
	}
}

func (v *viewportState) followSelected(selectedIndex int) {
	if !v.vpReady {
		return
	}
	if selectedIndex < v.vp.YOffset {
		v.vp.SetYOffset(selectedIndex)
	} else if selectedIndex >= v.vp.YOffset+v.vp.Height {
		v.vp.SetYOffset(selectedIndex - v.vp.Height + 1)
	}
}
