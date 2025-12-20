package window

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// ============ TAPPABLE RECTANGLE WIDGET ============

type tappableRect struct {
	widget.BaseWidget
	rect  *canvas.Rectangle
	onTap func()
}

func newTappableRect(rect *canvas.Rectangle, onTap func()) *tappableRect {
	t := &tappableRect{rect: rect, onTap: onTap}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tappableRect) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.rect)
}

func (t *tappableRect) Tapped(_ *fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}

func (t *tappableRect) TappedSecondary(_ *fyne.PointEvent) {}
