package desktop

import "math"

const TrayWidth = 100
const TrayHeight = 284

type Point struct{ X, Y int }
type Rect struct{ X, Y, Width, Height int }

// TrayBounds uses physical pixels throughout. The anchor is the right-click
// position, including when the icon is in Explorer's overflow flyout.
func TrayBounds(anchor Point, work Rect, scale float64) Rect {
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		scale = 1
	}
	width := max(1, int(math.Round(TrayWidth*scale)))
	height := max(1, int(math.Round(TrayHeight*scale)))
	gap := max(1, int(math.Round(4*scale)))
	if work.Width <= 0 || work.Height <= 0 {
		return Rect{anchor.X + gap, anchor.Y + gap, width, height}
	}
	width = min(width, work.Width)
	height = min(height, work.Height)
	x, y := anchor.X+gap, anchor.Y+gap
	if x+width > work.X+work.Width {
		x = anchor.X - width - gap
	}
	if y+height > work.Y+work.Height {
		y = anchor.Y - height - gap
	}
	x = max(work.X, min(x, work.X+work.Width-width))
	y = max(work.Y, min(y, work.Y+work.Height-height))
	return Rect{x, y, width, height}
}
