package main

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// renderSVG renders the diagram rows as an SVG document. Each glyph of the
// chess font is converted to a vector path, so the result is self-contained
// and does not depend on an embedded font.
func renderSVG(opts *options, rows []string) ([]byte, error) {
	f, err := truetype.Parse(merida.ttf)
	if err != nil {
		return nil, err
	}

	fontSize := float64(opts.size) / 10.0
	scale := fixed.Int26_6(fontSize * 64)

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`+"\n",
		opts.size, opts.size, opts.size, opts.size)

	bgFill, bgOpacity := svgColor(opts.bg)
	if bgOpacity != 0 {
		fmt.Fprintf(&b, `  <rect width="100%%" height="100%%" fill="%s"`, bgFill)
		if bgOpacity != 1 {
			fmt.Fprintf(&b, ` fill-opacity="%s"`, f2s(bgOpacity))
		}
		b.WriteString("/>\n")
	}

	fgFill, fgOpacity := svgColor(opts.fg)
	fmt.Fprintf(&b, `  <g fill="%s"`, fgFill)
	if fgOpacity != 1 {
		fmt.Fprintf(&b, ` fill-opacity="%s"`, f2s(fgOpacity))
	}
	b.WriteString(">\n")

	var buf truetype.GlyphBuf
	for rowIdx, row := range rows {
		baseline := float64(rowIdx+1) * fontSize
		penX := 0.0
		for _, r := range row {
			i := f.Index(r)
			if err := buf.Load(f, scale, i, font.HintingNone); err != nil {
				return nil, err
			}
			d := glyphPath(&buf)
			if d != "" {
				fmt.Fprintf(&b, `    <path d="%s" transform="translate(%s %s)"/>`+"\n",
					d, f2s(penX), f2s(baseline))
			}
			penX += float64(buf.AdvanceWidth) / 64
		}
	}
	b.WriteString("  </g>\n")
	b.WriteString("</svg>\n")

	return []byte(b.String()), nil
}

func svgColor(c color.Color) (hex string, opacity float64) {
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	hex = fmt.Sprintf("#%02x%02x%02x", n.R, n.G, n.B)
	return hex, float64(n.A) / 255
}

func f2s(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func fixedF(v fixed.Int26_6) string {
	return f2s(float64(v) / 64)
}

// glyphPath converts the loaded glyph's TrueType contours to an SVG path. The
// font's Y axis points up, while SVG's points down, so Y is negated here.
func glyphPath(buf *truetype.GlyphBuf) string {
	var b strings.Builder
	start := 0
	for _, end := range buf.Ends {
		pts := buf.Points[start:end]
		start = end
		if len(pts) == 0 {
			continue
		}
		contour(&b, pts)
	}
	return b.String()
}

// contour converts a single closed TrueType contour to SVG path commands. The
// first on-curve point starts the contour; for contours made solely of
// off-curve points an implicit on-curve point between the last and the first
// point is used.
func contour(b *strings.Builder, pts []truetype.Point) {
	n := len(pts)
	isOn := func(i int) bool { return pts[i].Flags&1 != 0 }

	i0 := -1
	for i := 0; i < n; i++ {
		if isOn(i) {
			i0 = i
			break
		}
	}

	var sx, sy fixed.Int26_6
	first := 0
	if i0 == -1 {
		sx = (pts[n-1].X + pts[0].X) / 2
		sy = (pts[n-1].Y + pts[0].Y) / 2
	} else {
		sx, sy = pts[i0].X, pts[i0].Y
		first = i0 + 1
	}
	fmt.Fprintf(b, "M%s %s", fixedF(sx), fixedF(-sy))

	penX, penY := sx, sy
	count := 0
	i := first % n
	for count < n {
		p := pts[i]
		if isOn(i) {
			if p.X != penX || p.Y != penY {
				fmt.Fprintf(b, "L%s %s", fixedF(p.X), fixedF(-p.Y))
				penX, penY = p.X, p.Y
			}
			i = (i + 1) % n
			count++
		} else {
			j := (i + 1) % n
			if isOn(j) {
				fmt.Fprintf(b, "Q%s %s %s %s", fixedF(p.X), fixedF(-p.Y),
					fixedF(pts[j].X), fixedF(-pts[j].Y))
				penX, penY = pts[j].X, pts[j].Y
				i = (j + 1) % n
				count += 2
			} else {
				mx := (p.X + pts[j].X) / 2
				my := (p.Y + pts[j].Y) / 2
				fmt.Fprintf(b, "Q%s %s %s %s", fixedF(p.X), fixedF(-p.Y),
					fixedF(mx), fixedF(-my))
				penX, penY = mx, my
				i = j
				count++
			}
		}
	}
	b.WriteString("Z")
}
