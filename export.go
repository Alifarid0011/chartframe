package chartframe

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/text"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

// Figure is a grid of charts. Finish mutating its charts before rendering.
type Figure struct {
	title   string
	columns int
	charts  []*Chart
}

// Grid lays out charts in row-major order with one to four columns.
// A figure is limited to 16 charts, 8192 pixels per dimension and 32 megapixels.
func Grid(title string, columns int, charts ...*Chart) *Figure {
	return &Figure{title: title, columns: columns, charts: append([]*Chart(nil), charts...)}
}

// SVG renders a standalone scalable vector image.
func (c *Chart) SVG() ([]byte, error) { return c.render("svg") }

// HTML renders a self-contained page with viewport pan, zoom, reset and SVG download.
func (c *Chart) HTML() ([]byte, error) {
	b, err := c.SVG()
	if err != nil {
		return nil, err
	}
	return htmlDocument(c.title, b)
}

// Write renders svg, png, pdf or html to w. Errors, including short writes, propagate.
func (c *Chart) Write(w io.Writer, format string) error {
	if w == nil {
		return errors.New("chartframe: nil writer")
	}
	b, err := c.render(strings.ToLower(format))
	if err != nil {
		return err
	}
	return writeAll(w, b)
}

// Save renders before touching the destination, then renames a sibling temporary file.
// New files are owner-readable/writable. Replacement atomicity follows os.Rename.
func (c *Chart) Save(path string) error {
	b, err := c.render(fileFormat(path))
	if err != nil {
		return err
	}
	return saveBytes(path, b)
}

// SVG renders the complete grid as a single image.
func (f *Figure) SVG() ([]byte, error) { return f.render("svg") }

// HTML renders the grid in a standalone interactive viewport.
func (f *Figure) HTML() ([]byte, error) {
	b, err := f.SVG()
	if err != nil {
		return nil, err
	}
	return htmlDocument(f.title, b)
}

// Write renders a grid to svg, png, pdf or html.
func (f *Figure) Write(w io.Writer, format string) error {
	if w == nil {
		return errors.New("chartframe: nil writer")
	}
	b, err := f.render(strings.ToLower(format))
	if err != nil {
		return err
	}
	return writeAll(w, b)
}

// Save writes the grid using the destination's file extension.
func (f *Figure) Save(path string) error {
	b, err := f.render(fileFormat(path))
	if err != nil {
		return err
	}
	return saveBytes(path, b)
}

func fileFormat(path string) string {
	return strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
}
func writeAll(w io.Writer, b []byte) error {
	n, err := w.Write(b)
	if err != nil {
		return err
	}
	if n != len(b) {
		return io.ErrShortWrite
	}
	return nil
}
func saveBytes(path string, b []byte) (err error) {
	f, err := os.CreateTemp(filepath.Dir(path), ".chartframe-*")
	if err != nil {
		return err
	}
	defer func() { f.Close(); os.Remove(f.Name()) }()
	if err = writeAll(f, b); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
func (c *Chart) render(format string) ([]byte, error) {
	if c == nil {
		return nil, errors.New("chartframe: nil chart")
	}
	if format == "html" {
		return c.HTML()
	}
	return renderGrid("", 1, []*Chart{c}, format)
}
func (f *Figure) render(format string) ([]byte, error) {
	if f == nil {
		return nil, errors.New("chartframe: nil figure")
	}
	if format == "html" {
		return f.HTML()
	}
	return renderGrid(f.title, f.columns, f.charts, format)
}

func renderGrid(title string, columns int, charts []*Chart, format string) ([]byte, error) {
	if format != "svg" && format != "png" && format != "pdf" {
		return nil, fmt.Errorf("chartframe: unsupported output format %q", format)
	}
	if !validText(title) {
		return nil, errors.New("chartframe: invalid figure title")
	}
	if columns < 1 || columns > 4 || len(charts) == 0 || len(charts) > 16 {
		return nil, errors.New("chartframe: grid requires 1..4 columns and 1..16 charts")
	}
	rows := (len(charts) + columns - 1) / columns
	cw, ch := 0, 0
	plots := make([][]*plot.Plot, rows)
	for r := range plots {
		plots[r] = make([]*plot.Plot, columns)
	}
	for i, c := range charts {
		if err := c.Err(); err != nil {
			return nil, fmt.Errorf("chartframe: panel %d: %w", i+1, err)
		}
		p, err := c.buildPlot()
		if err != nil {
			return nil, fmt.Errorf("chartframe: panel %d: %w", i+1, err)
		}
		plots[i/columns][i%columns] = p
		cw = max(cw, c.width)
		ch = max(ch, c.height)
	}
	header := 0
	if title != "" {
		header = 44
	}
	w, h := cw*columns, ch*rows+header
	if w > 8192 || h > 8192 || int64(w)*int64(h) > 32_000_000 {
		return nil, errors.New("chartframe: figure exceeds rendering size limit")
	}
	canvas, err := draw.NewFormattedCanvas(px(w), px(h), format)
	if err != nil {
		return nil, err
	}
	dc := draw.New(canvas)
	if title != "" {
		style := text.Style{Color: plots[0][0].Title.TextStyle.Color, Font: plots[0][0].Title.TextStyle.Font, Handler: plot.DefaultTextHandler, XAlign: draw.XCenter, YAlign: draw.YCenter}
		dc.SetColor(plots[0][0].BackgroundColor)
		dc.FillPolygon(plots[0][0].BackgroundColor, []vg.Point{{X: 0, Y: 0}, {X: px(w), Y: 0}, {X: px(w), Y: px(h)}, {X: 0, Y: px(h)}})
		dc.FillText(style, vg.Point{X: px(w) / 2, Y: px(h) - px(header)/2}, title)
		dc.Max.Y -= px(header)
	}
	tiles := draw.Tiles{Rows: rows, Cols: columns}
	if len(charts) > 1 {
		tiles.PadX = vg.Points(12)
		tiles.PadY = vg.Points(12)
	}
	canvases := plot.Align(plots, tiles, dc)
	for r := range plots {
		for col, p := range plots[r] {
			if p != nil {
				p.Draw(canvases[r][col])
			}
		}
	}
	var b bytes.Buffer
	if _, err := canvas.WriteTo(&b); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
func px(n int) vg.Length { return vg.Length(n) * vg.Inch / 96 }
