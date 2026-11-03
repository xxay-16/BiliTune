package desktop

import "testing"

func TestTrayMenuAnchorsToClick(t *testing.T) {
	work := Rect{0, 0, 1920, 1040}
	tests := []struct {
		name   string
		anchor Point
		want   Rect
	}{
		{"overflow flyout", Point{1600, 900}, Rect{1604, 612, 100, 284}},
		{"bottom right taskbar", Point{1900, 1060}, Rect{1796, 756, 100, 284}},
		{"top taskbar", Point{700, 10}, Rect{704, 14, 100, 284}},
		{"left flyout", Point{63, 900}, Rect{67, 612, 100, 284}},
		{"room below cursor", Point{800, 300}, Rect{804, 304, 100, 284}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := TrayBounds(test.anchor, work, 1); got != test.want {
				t.Fatalf("got %+v, want %+v", got, test.want)
			}
		})
	}
}
func TestTrayMenuMixedDPIAndNegativeMonitor(t *testing.T) {
	for _, scale := range []float64{1, 1.25, 1.5, 1.75, 2} {
		work := Rect{-2560, -360, 2560, 1400}
		anchor := Point{-1500, 900}
		got := TrayBounds(anchor, work, scale)
		if got.X >= 0 || got.Y < work.Y || got.X < work.X || got.X+got.Width > work.X+work.Width || got.Y+got.Height > work.Y+work.Height {
			t.Fatalf("scale %v leaves target monitor: %+v", scale, got)
		}
		if got.Width != int(TrayWidth*scale) {
			t.Fatalf("scale %v width %d", scale, got.Width)
		}
		if got.X != anchor.X+int(4*scale) {
			t.Fatalf("scale %v lost cursor anchor: %+v", scale, got)
		}
	}
}
func TestTrayMenuClampsEveryWorkAreaEdge(t *testing.T) {
	work := Rect{50, 40, 400, 450}
	for _, anchor := range []Point{{0, 0}, {500, 0}, {0, 600}, {500, 600}, {100, 100}} {
		bounds := TrayBounds(anchor, work, 1.5)
		if bounds.X < work.X || bounds.Y < work.Y || bounds.X+bounds.Width > work.X+work.Width || bounds.Y+bounds.Height > work.Y+work.Height {
			t.Fatal(anchor, bounds)
		}
	}
}
