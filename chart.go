package chartframe

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"sort"
	"unicode/utf8"
)

// Chart builds a plot. Input slices are copied. Once construction is complete,
// concurrent rendering is safe; concurrent mutation requires synchronization.
type Chart struct {
	title, xLabel, yLabel string
	width, height         int
	dark, logX, logY      bool
	layers                []layer
	err                   error
}

type layer struct {
	kind, name      string
	x, y            []float64
	labels, yLabels []string
	z               [][]float64
	bins            int
}

// New creates a chart with a default size of 900 by 540 pixels.
func New(title string) *Chart {
	c := &Chart{title: title, width: 900, height: 540}
	c.checkText(title)
	return c
}

// Err returns the first validation error, or nil for a valid builder.
func (c *Chart) Err() error {
	if c == nil {
		return errors.New("chartframe: nil chart")
	}
	return c.err
}

func (c *Chart) fail(err error) *Chart {
	if c != nil && c.err == nil {
		c.err = err
	}
	return c
}

func validText(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if r != '\t' && r != '\n' && r != '\r' && (r < 0x20 || r == 0xfffe || r == 0xffff) {
			return false
		}
	}
	return true
}

func (c *Chart) checkText(ss ...string) bool {
	for _, s := range ss {
		if !validText(s) {
			c.fail(errors.New("chartframe: text must contain valid UTF-8 and XML characters"))
			return false
		}
	}
	return true
}

func finiteValues(v []float64) bool {
	for _, f := range v {
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return false
		}
	}
	return true
}

func (c *Chart) add(l layer) *Chart {
	if c == nil || c.err != nil {
		return c
	}
	if !c.checkText(l.name) || !c.checkText(l.labels...) || !c.checkText(l.yLabels...) {
		return c
	}
	if len(l.y) == 0 && len(l.z) == 0 {
		return c.fail(errors.New("chartframe: data must not be empty"))
	}
	if !finiteValues(l.x) || !finiteValues(l.y) {
		return c.fail(errors.New("chartframe: data must be finite"))
	}
	if len(c.layers) > 0 {
		first := c.layers[0]
		if l.kind == "heatmap" || first.kind == "heatmap" || ((l.kind == "box") != (first.kind == "box")) || ((l.kind == "bar") != (first.kind == "bar")) {
			return c.fail(errors.New("chartframe: incompatible chart layers"))
		}
		if l.kind == "bar" && !slices.Equal(l.labels, first.labels) {
			return c.fail(errors.New("chartframe: bar categories must match"))
		}
	}
	if err := c.checkLog(l); err != nil {
		return c.fail(err)
	}
	l.x = slices.Clone(l.x)
	l.y = slices.Clone(l.y)
	l.labels = slices.Clone(l.labels)
	l.yLabels = slices.Clone(l.yLabels)
	if l.z != nil {
		z := make([][]float64, len(l.z))
		for i := range l.z {
			z[i] = slices.Clone(l.z[i])
		}
		l.z = z
	}
	c.layers = append(c.layers, l)
	return c
}

func (c *Chart) checkLog(l layer) error {
	if !c.logX && !c.logY {
		return nil
	}
	switch l.kind {
	case "bar", "histogram", "box", "heatmap":
		return fmt.Errorf("chartframe: %s does not support logarithmic axes", l.kind)
	}
	for _, axis := range []struct {
		enabled bool
		values  []float64
	}{{c.logX, l.x}, {c.logY, l.y}} {
		if axis.enabled {
			for _, v := range axis.values {
				if v <= 0 {
					return errors.New("chartframe: logarithmic coordinates must be positive")
				}
			}
		}
	}
	return nil
}

func (c *Chart) xy(kind, name string, x, y []float64) *Chart {
	if c == nil || c.err != nil {
		return c
	}
	if len(x) != len(y) {
		return c.fail(errors.New("chartframe: x and y lengths must match"))
	}
	return c.add(layer{kind: kind, name: name, x: x, y: y})
}

// Line adds a connected series in the supplied order.
func (c *Chart) Line(name string, x, y []float64) *Chart { return c.xy("line", name, x, y) }

// Scatter adds a series of points.
func (c *Chart) Scatter(name string, x, y []float64) *Chart { return c.xy("scatter", name, x, y) }

// Bar adds categorical values. All bar layers must use identical labels.
func (c *Chart) Bar(name string, labels []string, values []float64) *Chart {
	if c == nil || c.err != nil {
		return c
	}
	if len(labels) != len(values) {
		return c.fail(errors.New("chartframe: labels and values lengths must match"))
	}
	return c.add(layer{kind: "bar", name: name, labels: labels, y: values})
}

// Histogram adds samples split into bins of equal width.
func (c *Chart) Histogram(name string, values []float64, bins int) *Chart {
	if c == nil || c.err != nil {
		return c
	}
	if _, err := Bin(values, bins); err != nil {
		return c.fail(err)
	}
	return c.add(layer{kind: "histogram", name: name, y: values, bins: bins})
}

// Box adds a box plot of samples.
func (c *Chart) Box(name string, values []float64) *Chart {
	return c.add(layer{kind: "box", name: name, y: values})
}

// Heatmap adds a rectangular matrix, with rows corresponding to yLabels.
func (c *Chart) Heatmap(name string, xLabels, yLabels []string, z [][]float64) *Chart {
	if c == nil || c.err != nil {
		return c
	}
	if len(xLabels) == 0 || len(yLabels) == 0 || len(z) != len(yLabels) {
		return c.fail(errors.New("chartframe: heatmap dimensions must match labels"))
	}
	for _, row := range z {
		if len(row) != len(xLabels) || !finiteValues(row) {
			return c.fail(errors.New("chartframe: heatmap rows must match labels and contain finite values"))
		}
	}
	return c.add(layer{kind: "heatmap", name: name, labels: xLabels, yLabels: yLabels, z: z})
}

// ECDF adds the empirical cumulative distribution of samples.
func (c *Chart) ECDF(name string, values []float64) *Chart {
	if c == nil || c.err != nil {
		return c
	}
	if len(values) == 0 || !finiteValues(values) {
		return c.fail(errors.New("chartframe: ECDF requires finite, nonempty samples"))
	}
	x := slices.Clone(values)
	sort.Float64s(x)
	y := make([]float64, len(x))
	for i := range y {
		y[i] = float64(i+1) / float64(len(y))
	}
	return c.add(layer{kind: "ecdf", name: name, x: x, y: y})
}

// LinearFit adds a least-squares fitted line spanning the supplied x values.
func (c *Chart) LinearFit(name string, x, y []float64) *Chart {
	if c == nil || c.err != nil {
		return c
	}
	fit, err := FitLinear(x, y)
	if err != nil {
		return c.fail(err)
	}
	lo, hi := slices.Min(x), slices.Max(x)
	return c.Line(name, []float64{lo, hi}, []float64{math.FMA(fit.Slope, lo, fit.Intercept), math.FMA(fit.Slope, hi, fit.Intercept)})
}

// Labels sets the axis labels.
func (c *Chart) Labels(x, y string) *Chart {
	if c != nil && c.err == nil && c.checkText(x, y) {
		c.xLabel = x
		c.yLabel = y
	}
	return c
}

// Size sets logical pixel dimensions, each between 240 and 4096 inclusive.
func (c *Chart) Size(width, height int) *Chart {
	if c == nil || c.err != nil {
		return c
	}
	if width < 240 || width > 4096 || height < 240 || height > 4096 {
		return c.fail(errors.New("chartframe: dimensions must be between 240 and 4096 pixels"))
	}
	c.width = width
	c.height = height
	return c
}

// Dark enables the dark theme.
func (c *Chart) Dark() *Chart {
	if c != nil && c.err == nil {
		c.dark = true
	}
	return c
}

// LogX enables a logarithmic x axis and validates all existing layers.
func (c *Chart) LogX() *Chart { return c.log(true) }

// LogY enables a logarithmic y axis and validates all existing layers.
func (c *Chart) LogY() *Chart { return c.log(false) }
func (c *Chart) log(x bool) *Chart {
	if c == nil || c.err != nil {
		return c
	}
	if x {
		c.logX = true
	} else {
		c.logY = true
	}
	for _, l := range c.layers {
		if err := c.checkLog(l); err != nil {
			return c.fail(err)
		}
	}
	return c
}
