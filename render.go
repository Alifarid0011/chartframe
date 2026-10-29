package chartframe

import (
	"fmt"
	"image/color"
	"math"
	"strconv"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

var seriesColors = []color.NRGBA{
	{R: 52, G: 120, B: 246, A: 255}, {R: 13, G: 157, B: 144, A: 255},
	{R: 231, G: 157, B: 41, A: 255}, {R: 143, G: 101, B: 216, A: 255},
	{R: 222, G: 92, B: 115, A: 255},
}

// buildPlot creates a separate backend for every export, retaining no rendering
// state in the builder. The caller may safely render an unchanged chart in parallel.
func (c *Chart) buildPlot() (*plot.Plot, error) {
	if err := c.Err(); err != nil {
		return nil, err
	}
	if len(c.layers) == 0 {
		return nil, fmt.Errorf("chartframe: chart has no layers")
	}
	p := plot.New()
	p.Title.Text = c.title
	p.X.Label.Text = c.xLabel
	p.Y.Label.Text = c.yLabel
	p.Title.TextStyle.Font.Size = vg.Points(17)
	p.Legend.Top = true
	p.Legend.Left = true
	p.Legend.TextStyle.Font.Size = vg.Points(9)
	foreground := color.NRGBA{R: 38, G: 51, B: 69, A: 255}
	background := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	gridColor := color.NRGBA{R: 226, G: 232, B: 240, A: 255}
	if c.dark {
		foreground = color.NRGBA{R: 226, G: 232, B: 240, A: 255}
		background = color.NRGBA{R: 18, G: 27, B: 43, A: 255}
		gridColor = color.NRGBA{R: 50, G: 65, B: 85, A: 255}
	}
	p.BackgroundColor = background
	p.Title.TextStyle.Color = foreground
	p.Legend.TextStyle.Color = foreground
	for _, a := range []*plot.Axis{&p.X, &p.Y} {
		a.Color = foreground
		a.Label.TextStyle.Color = foreground
		a.Tick.Label.Color = foreground
		a.Tick.Color = foreground
		a.Label.TextStyle.Font.Size = vg.Points(11)
		a.Tick.Label.Font.Size = vg.Points(9)
	}
	if c.layers[0].kind != "heatmap" {
		grid := plotter.NewGrid()
		grid.Horizontal.Color = gridColor
		grid.Vertical.Color = gridColor
		p.Add(grid)
	}
	nominalX, nominalY := false, false
	for i, l := range c.layers {
		col := seriesColors[i%len(seriesColors)]
		switch l.kind {
		case "line", "ecdf", "scatter":
			points := make(plotter.XYs, len(l.x))
			for j := range points {
				points[j].X = l.x[j]
				points[j].Y = l.y[j]
			}
			if l.kind == "scatter" {
				dots, err := plotter.NewScatter(points)
				if err != nil {
					return nil, err
				}
				dots.Color = col
				dots.Radius = vg.Points(3)
				p.Add(dots)
				if l.name != "" {
					p.Legend.Add(l.name, dots)
				}
			} else {
				if l.kind == "ecdf" && !c.logY {
					// Include the initial jump, including a one-observation ECDF.
					points = append(plotter.XYs{{X: l.x[0], Y: 0}}, points...)
				}
				line, err := plotter.NewLine(points)
				if err != nil {
					return nil, err
				}
				line.Color = col
				line.Width = vg.Points(1.8)
				if l.kind == "ecdf" {
					line.StepStyle = plotter.PostStep
				}
				p.Add(line)
				if l.kind == "ecdf" && len(points) == 1 {
					dots, err := plotter.NewScatter(points)
					if err != nil {
						return nil, err
					}
					dots.Color = col
					dots.Radius = vg.Points(3)
					p.Add(dots)
				}
				if l.name != "" {
					p.Legend.Add(l.name, line)
				}
			}
		case "bar":
			groupWidth := math.Min(28, float64(c.width)*.7*.75/float64(len(l.labels)))
			width := vg.Points(groupWidth / float64(len(c.layers)))
			bars, err := plotter.NewBarChart(plotter.Values(l.y), width)
			if err != nil {
				return nil, err
			}
			bars.Color = col
			bars.LineStyle.Width = 0
			bars.Offset = (vg.Length(i) - vg.Length(len(c.layers)-1)/2) * width
			p.Add(bars)
			if l.name != "" {
				p.Legend.Add(l.name, bars)
			}
			p.NominalX(l.labels...)
			p.X.Min = -.5
			p.X.Max = float64(len(l.labels)) - .5
			nominalX = true
		case "histogram":
			binned, err := Bin(l.y, l.bins)
			if err != nil {
				return nil, err
			}
			hist := &plotter.Histogram{Bins: make([]plotter.HistogramBin, len(binned.Counts)), Width: binned.Edges[1] - binned.Edges[0], FillColor: color.NRGBA{R: col.R, G: col.G, B: col.B, A: 155}, LineStyle: draw.LineStyle{Color: col, Width: vg.Points(.5)}}
			for j, count := range binned.Counts {
				hist.Bins[j] = plotter.HistogramBin{Min: binned.Edges[j], Max: binned.Edges[j+1], Weight: float64(count)}
			}
			p.Add(zeroHistogram{hist})
			if l.name != "" {
				p.Legend.Add(l.name, hist)
			}
		case "box":
			box, err := type7Box(l.y, float64(i))
			if err != nil {
				return nil, err
			}
			box.FillColor = color.NRGBA{R: col.R, G: col.G, B: col.B, A: 100}
			box.BoxStyle.Color = col
			box.MedianStyle.Color = foreground
			box.WhiskerStyle.Color = col
			box.GlyphStyle.Color = col
			box.GlyphStyle.Radius = vg.Points(2.5)
			p.Add(box)
			nominalX = true
		case "heatmap":
			grid := newHeatGrid(l.z)
			hm := plotter.NewHeatMap(grid, heatColors())
			hm.Min = 0
			hm.Max = 1
			p.Add(hm)
			p.NominalX(l.labels...)
			yt := make(plot.ConstantTicks, len(l.yLabels))
			for j, label := range l.yLabels {
				yt[j] = plot.Tick{Value: float64(len(yt) - 1 - j), Label: label}
			}
			p.Y.Tick.Marker = yt
			p.X.Min = -.5
			p.X.Max = float64(len(l.labels)) - .5
			p.Y.Min = -.5
			p.Y.Max = float64(len(l.yLabels)) - .5
			nominalX = true
			nominalY = true
			lo, hi := grid.bounds()
			label := fmt.Sprintf("%s [%g, %g]", l.name, lo, hi)
			p.Legend.Add(label, gradientKey{heatColors()})
		default:
			return nil, fmt.Errorf("chartframe: unsupported layer kind %q", l.kind)
		}
	}
	if c.layers[0].kind == "box" {
		labels := make([]string, len(c.layers))
		for i, l := range c.layers {
			labels[i] = l.name
		}
		p.NominalX(labels...)
		p.X.Min = -.5
		p.X.Max = float64(len(labels)) - .5
	}
	if err := prepareAxis(&p.X, c.logX, nominalX); err != nil {
		return nil, fmt.Errorf("chartframe: x axis: %w", err)
	}
	if err := prepareAxis(&p.Y, c.logY, nominalY); err != nil {
		return nil, fmt.Errorf("chartframe: y axis: %w", err)
	}
	return p, nil
}

func prepareAxis(a *plot.Axis, logarithmic, nominal bool) error {
	lo, hi := a.Min, a.Max
	if !finiteValues([]float64{lo, hi}) || lo > hi {
		return fmt.Errorf("invalid finite range")
	}
	if logarithmic && lo <= 0 {
		return fmt.Errorf("logarithmic range must be positive")
	}
	if lo == hi {
		delta := math.Abs(lo) * .05
		if delta == 0 {
			delta = .5
		}
		lower, upper := lo-delta, hi+delta
		if logarithmic {
			lower, upper = lo/2, hi*2
		}
		if !math.IsInf(lower, 0) && (!logarithmic || lower > 0) {
			lo = lower
		}
		if !math.IsInf(upper, 0) {
			hi = upper
		}
	}
	if lo >= hi || math.IsInf(hi-lo, 0) {
		return fmt.Errorf("range has no representable finite span")
	}
	a.Min = lo
	a.Max = hi
	if nominal {
		return nil
	}
	tickLo, tickHi := lo, hi
	if logarithmic {
		tickLo, tickHi = math.Log(lo), math.Log(hi)
		if tickLo >= tickHi {
			return fmt.Errorf("logarithmic coordinates are indistinguishable")
		}
		a.Scale = plot.LogScale{}
	}
	ticks := make(plot.ConstantTicks, 0, 5)
	for i := 0; i < 5; i++ {
		v := interpolate(tickLo, tickHi, float64(i)/4)
		if logarithmic {
			v = math.Exp(v)
		}
		if i == 0 {
			v = lo
		}
		if i == 4 {
			v = hi
		}
		if !finiteValues([]float64{v}) || (len(ticks) > 0 && v <= ticks[len(ticks)-1].Value) {
			continue
		}
		ticks = append(ticks, plot.Tick{Value: v, Label: strconv.FormatFloat(v, 'g', 5, 64)})
	}
	// Close coordinates need enough digits to distinguish their labels.
	for i := 1; i < len(ticks); i++ {
		if ticks[i].Label == ticks[i-1].Label {
			for j := range ticks {
				ticks[j].Label = strconv.FormatFloat(ticks[j].Value, 'g', -1, 64)
			}
			break
		}
	}
	a.Tick.Marker = ticks
	return nil
}

func type7Box(values []float64, location float64) (*plotter.BoxPlot, error) {
	summary, err := Describe(values)
	if err != nil {
		return nil, err
	}
	b, err := plotter.NewBoxPlot(vg.Points(25), location, plotter.Values(values))
	if err != nil {
		return nil, err
	}
	b.Quartile1 = summary.Q1
	b.Quartile3 = summary.Q3
	b.Median = summary.Median
	b.Min = summary.Min
	b.Max = summary.Max
	b.AdjLow = math.Inf(1)
	b.AdjHigh = math.Inf(-1)
	b.Outside = nil
	// Infinite fences are harmless: they mean every finite sample on that side
	// is inside the whisker. The plotted quartiles and adjacent values stay finite.
	low := summary.Q1 - 1.5*(summary.Q3-summary.Q1)
	high := summary.Q3 + 1.5*(summary.Q3-summary.Q1)
	for i, v := range values {
		if v < low || v > high {
			b.Outside = append(b.Outside, i)
		} else {
			b.AdjLow = math.Min(b.AdjLow, v)
			b.AdjHigh = math.Max(b.AdjHigh, v)
		}
	}
	return b, nil
}

// Gonum's histogram draws its baseline at the bottom of the viewport. Numeric
// overlays can extend below zero, so use the same bins with a zero baseline.
type zeroHistogram struct{ *plotter.Histogram }

func (h zeroHistogram) Plot(c draw.Canvas, p *plot.Plot) {
	tx, ty := p.Transforms(&c)
	for _, b := range h.Bins {
		points := []vg.Point{{X: tx(b.Min), Y: ty(0)}, {X: tx(b.Max), Y: ty(0)}, {X: tx(b.Max), Y: ty(b.Weight)}, {X: tx(b.Min), Y: ty(b.Weight)}}
		c.FillPolygon(h.FillColor, c.ClipPolygonXY(points))
		points = append(points, points[0])
		c.StrokeLines(h.LineStyle, c.ClipLinesXY(points)...)
	}
}

type heatGrid struct {
	values [][]float64
	lo, hi float64
}

func newHeatGrid(values [][]float64) heatGrid {
	g := heatGrid{values: values, lo: math.Inf(1), hi: math.Inf(-1)}
	for _, row := range values {
		for _, v := range row {
			g.lo = math.Min(g.lo, v)
			g.hi = math.Max(g.hi, v)
		}
	}
	return g
}
func (g heatGrid) Dims() (int, int)           { return len(g.values[0]), len(g.values) }
func (g heatGrid) X(i int) float64            { return float64(i) }
func (g heatGrid) Y(i int) float64            { return float64(i) }
func (g heatGrid) bounds() (float64, float64) { return g.lo, g.hi }
func (g heatGrid) Z(c, r int) float64 {
	if g.lo == g.hi {
		return .5
	}
	v := g.values[len(g.values)-1-r][c]
	span := g.hi - g.lo
	if !math.IsInf(span, 0) {
		return (v - g.lo) / span
	}
	scale := math.Max(math.Abs(g.lo), math.Abs(g.hi))
	return (v/scale - g.lo/scale) / (g.hi/scale - g.lo/scale)
}

type chartPalette []color.Color

func (p chartPalette) Colors() []color.Color { return p }
func heatColors() chartPalette {
	colors := make(chartPalette, 64)
	for i := range colors {
		t := float64(i) / float64(len(colors)-1)
		colors[i] = color.NRGBA{R: uint8(238 - 217*t), G: uint8(246 - 96*t), B: uint8(255 - 113*t), A: 255}
	}
	return colors
}

type gradientKey struct{ colors chartPalette }

func (g gradientKey) Thumbnail(c *draw.Canvas) {
	for i, col := range g.colors {
		x0 := c.Min.X + (c.Max.X-c.Min.X)*vg.Length(i)/vg.Length(len(g.colors))
		x1 := c.Min.X + (c.Max.X-c.Min.X)*vg.Length(i+1)/vg.Length(len(g.colors))
		c.FillPolygon(col, []vg.Point{{X: x0, Y: c.Min.Y}, {X: x1, Y: c.Min.Y}, {X: x1, Y: c.Max.Y}, {X: x0, Y: c.Max.Y}})
	}
}
