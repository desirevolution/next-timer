package client

import (
	"math"
	"strings"
)

// This is an original vector-stroke clock inspired by Termdown's large
// terminal typography. It does not use Termdown code, font files or FIGlet
// assets. Unlike a seven-segment display, its digits have genuine curved
// contours (notably 2, 3, 6, 7 and 9). Terminal quadrant characters give
// the strokes sub-cell resolution and soft, rounded ends.
//
// Colors and UI state are supplied by the calling Bubble Tea/Lip Gloss view.
func largeClockAdaptive(clock string, maxWidth, maxHeight int) string {
	if maxWidth < 28 || maxHeight < 8 {
		return clock
	}
	digits, colons := 0, 0
	for _, r := range clock {
		switch {
		case '0' <= r && r <= '9':
			digits++
		case r == ':':
			colons++
		default:
			return clock
		}
	}
	if digits == 0 || colons > 2 {
		return clock
	}

	// Cell dimensions are intentionally different from pixel dimensions:
	// most terminals have cells about twice as tall as they are wide.
	width, height, colonWidth, gap := 0, 0, 0, 0
	for h := maxHeight; h >= 8; h-- {
		w := max(7, int(math.Round(float64(h)*0.95)))
		c := max(3, w/4)
		g := max(1, w/14)
		if digits*w+colons*c+(digits+colons-1)*g <= maxWidth {
			width, height, colonWidth, gap = w, h, c, g
			break
		}
	}
	if height == 0 {
		return clock
	}

	glyphs := make([][]string, 0, digits+colons)
	for _, r := range clock {
		if r == ':' {
			glyphs = append(glyphs, termdownGlyph(r, colonWidth, height))
		} else {
			glyphs = append(glyphs, termdownGlyph(r, width, height))
		}
	}
	rows := make([]string, height)
	separator := strings.Repeat(" ", gap)
	for y := range rows {
		var row strings.Builder
		for i, glyph := range glyphs {
			if i > 0 {
				row.WriteString(separator)
			}
			row.WriteString(glyph[y])
		}
		rows[y] = row.String()
	}
	return strings.Join(rows, "\n")
}

// A stroke is a line segment in glyph coordinates, where 0..1 spans the
// glyph's drawing rectangle. Curves are approximated by short segments.
type stroke struct{ x0, y0, x1, y1 float64 }
type glyphPath struct {
	x, y     float64
	segments []stroke
}

func (p *glyphPath) move(x, y float64) { p.x, p.y = x, y }
func (p *glyphPath) line(x, y float64) {
	p.segments = append(p.segments, stroke{p.x, p.y, x, y})
	p.x, p.y = x, y
}
func (p *glyphPath) cubic(x1, y1, x2, y2, x3, y3 float64) {
	x0, y0 := p.x, p.y
	for i := 1; i <= 18; i++ {
		t := float64(i) / 18
		u := 1 - t
		x := u*u*u*x0 + 3*u*u*t*x1 + 3*u*t*t*x2 + t*t*t*x3
		y := u*u*u*y0 + 3*u*u*t*y1 + 3*u*t*t*y2 + t*t*t*y3
		p.line(x, y)
	}
}

// Each character is drawn as strokes rather than a fixed-sized bitmap. This
// retains curved contours when resized and keeps the terminal executable
// independent of installed fonts.
func termdownPaths(r rune) []stroke {
	p := &glyphPath{}
	switch r {
	case '0':
		p.move(.50, .10)
		p.cubic(.28, .10, .19, .25, .19, .50)
		p.cubic(.19, .74, .28, .90, .50, .90)
		p.cubic(.72, .90, .81, .74, .81, .50)
		p.cubic(.81, .25, .72, .10, .50, .10)
	case '1':
		p.move(.27, .28)
		p.line(.54, .11)
		p.line(.54, .90)
		p.move(.27, .90)
		p.line(.80, .90)
	case '2':
		p.move(.18, .28)
		p.cubic(.20, .12, .36, .10, .51, .10)
		p.cubic(.73, .10, .83, .21, .81, .35)
		p.cubic(.79, .48, .58, .63, .20, .90)
		p.line(.82, .90)
	case '3':
		p.move(.19, .19)
		p.cubic(.36, .07, .68, .07, .78, .22)
		p.cubic(.87, .36, .73, .49, .52, .50)
		p.cubic(.74, .49, .87, .62, .79, .78)
		p.cubic(.70, .94, .36, .95, .19, .82)
	case '4':
		p.move(.68, .11)
		p.line(.68, .90)
		p.move(.16, .64)
		p.line(.83, .64)
		p.move(.16, .64)
		p.line(.59, .11)
	case '5':
		p.move(.82, .11)
		p.line(.28, .11)
		p.line(.22, .47)
		p.cubic(.38, .39, .72, .41, .79, .59)
		p.cubic(.92, .83, .64, .97, .41, .89)
		p.cubic(.30, .87, .21, .81, .18, .77)
	case '6':
		p.move(.78, .18)
		p.cubic(.48, .04, .21, .35, .20, .61)
		p.cubic(.19, .83, .34, .90, .54, .90)
		p.cubic(.76, .90, .86, .73, .77, .56)
		p.cubic(.66, .40, .39, .43, .20, .61)
	case '7':
		p.move(.18, .11)
		p.line(.84, .11)
		p.cubic(.67, .36, .49, .64, .39, .90)
	case '8':
		p.move(.50, .50)
		p.cubic(.30, .47, .22, .35, .24, .24)
		p.cubic(.27, .06, .73, .06, .77, .24)
		p.cubic(.81, .39, .70, .47, .50, .50)
		p.cubic(.29, .54, .17, .64, .24, .78)
		p.cubic(.34, .96, .69, .96, .78, .78)
		p.cubic(.85, .64, .72, .55, .50, .50)
	case '9':
		p.move(.79, .40)
		p.cubic(.64, .58, .36, .56, .26, .42)
		p.cubic(.15, .25, .29, .10, .49, .10)
		p.cubic(.72, .10, .84, .31, .79, .56)
		p.cubic(.74, .81, .58, .94, .24, .83)
	}
	return p.segments
}

var termdownQuadrants = [16]rune{' ', '▘', '▝', '▀', '▖', '▌', '▞', '▛', '▗', '▚', '▐', '▜', '▄', '▙', '▟', '█'}

func termdownGlyph(r rune, width, height int) []string {
	segments := termdownPaths(r)
	w, h := float64(width), float64(height)
	// Capsule stroke radii scale with the character width and height.
	// The thinner strokes more closely resemble FIGlet/Termdown typography
	// than a block-style seven-segment display.
	thickness := math.Max(.85, math.Min(w*.068, h*.077))
	radiusSquared := thickness * thickness
	lines := make([]string, height)
	for y := 0; y < height; y++ {
		var line strings.Builder
		for x := 0; x < width; x++ {
			mask := 0
			for sy := 0; sy < 2; sy++ {
				for sx := 0; sx < 2; sx++ {
					px, py := float64(x)+(float64(sx)+.5)/2, float64(y)+(float64(sy)+.5)/2
					on := false
					if r == ':' {
						dx := px - w*.50
						dotR := math.Max(.75, math.Min(w*.23, h*.055))
						for _, yy := range []float64{h * .37, h * .65} {
							dy := py - yy
							if dx*dx+dy*dy <= dotR*dotR {
								on = true
								break
							}
						}
					} else {
						for _, s := range segments {
							// Distance is evaluated in terminal-cell space,
							// not normalized space, to keep circular caps.
							if pointToStrokeSquared(px, py, stroke{s.x0 * w, s.y0 * h, s.x1 * w, s.y1 * h}) <= radiusSquared {
								on = true
								break
							}
						}
					}
					if on {
						mask |= 1 << (sx + sy*2)
					}
				}
			}
			line.WriteRune(termdownQuadrants[mask])
		}
		lines[y] = line.String()
	}
	return lines
}
func pointToStrokeSquared(x, y float64, s stroke) float64 {
	dx, dy := s.x1-s.x0, s.y1-s.y0
	d := dx*dx + dy*dy
	t := 0.0
	if d > 0 {
		t = math.Max(0, math.Min(1, ((x-s.x0)*dx+(y-s.y0)*dy)/d))
	}
	ex, ey := x-(s.x0+t*dx), y-(s.y0+t*dy)
	return ex*ex + ey*ey
}
