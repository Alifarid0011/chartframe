package chartframe

import (
	"math"
	"reflect"
	"testing"
)

func TestBuilderValidCharts(t *testing.T) {
	for name, c := range map[string]*Chart{
		"numeric":   New("Trends").Line("line", []float64{1, 2}, []float64{2, 4}).Scatter("dots", []float64{1}, []float64{3}).ECDF("cdf", []float64{3, 1, 2}),
		"bars":      New("Categories").Bar("a", []string{"A", "B"}, []float64{1, -2}).Bar("b", []string{"A", "B"}, []float64{2, 3}),
		"boxes":     New("Boxes").Box("a", []float64{1, 2}).Box("b", []float64{3, 4}),
		"heatmap":   New("Heat").Heatmap("heat", []string{"X"}, []string{"Y"}, [][]float64{{1}}),
		"histogram": New("Samples").Histogram("a", []float64{1, 2, 3}, 2),
		"fit":       New("Fit").LinearFit("fit", []float64{1, 2, 3}, []float64{3, 5, 7}).Labels("x", "y").Size(240, 4096).Dark().LogX().LogY(),
	} {
		t.Run(name, func(t *testing.T) {
			if err := c.Err(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestBuilderRejectsInvalidData(t *testing.T) {
	for name, c := range map[string]*Chart{
		"empty":        New("").Line("", nil, nil),
		"mismatch":     New("").Scatter("", []float64{1}, []float64{1, 2}),
		"nan":          New("").Line("", []float64{math.NaN()}, []float64{1}),
		"infinity":     New("").Box("", []float64{math.Inf(1)}),
		"bins":         New("").Histogram("", []float64{1}, 0),
		"bad matrix":   New("").Heatmap("", []string{"a"}, []string{"b"}, [][]float64{{1, 2}}),
		"empty matrix": New("").Heatmap("", nil, nil, nil),
		"size":         New("").Size(239, 540),
		"title":        New("bad\x00"),
		"name":         New("").Box("bad\xff", []float64{1}),
		"label":        New("").Labels("bad\x01", "y"),
		"category":     New("").Bar("", []string{"bad\x00"}, []float64{1}),
		"fit":          New("").LinearFit("", []float64{1, 1}, []float64{2, 3}),
	} {
		t.Run(name, func(t *testing.T) {
			if c.Err() == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestBuilderLogValidationOrder(t *testing.T) {
	for _, before := range []bool{false, true} {
		for name, add := range map[string]func(*Chart) *Chart{
			"line":    func(c *Chart) *Chart { return c.Line("", []float64{0}, []float64{1}) },
			"bar":     func(c *Chart) *Chart { return c.Bar("", []string{"a"}, []float64{1}) },
			"hist":    func(c *Chart) *Chart { return c.Histogram("", []float64{1}, 2) },
			"box":     func(c *Chart) *Chart { return c.Box("", []float64{1}) },
			"heatmap": func(c *Chart) *Chart { return c.Heatmap("", []string{"a"}, []string{"b"}, [][]float64{{1}}) },
		} {
			t.Run(name+map[bool]string{true: "before", false: "after"}[before], func(t *testing.T) {
				c := New("")
				if before {
					c.LogX()
				}
				add(c)
				if !before {
					c.LogX()
				}
				if c.Err() == nil {
					t.Fatal("expected log validation error")
				}
			})
		}
	}
	if New("").Line("", []float64{1}, []float64{0}).LogY().Err() == nil {
		t.Fatal("zero y accepted")
	}
}

func TestBuilderOverlayCompatibility(t *testing.T) {
	for _, c := range []*Chart{
		New("").Bar("", []string{"a"}, []float64{1}).Bar("", []string{"b"}, []float64{2}),
		New("").Box("", []float64{1}).Line("", []float64{1}, []float64{2}),
		New("").Line("", []float64{1}, []float64{2}).Bar("", []string{"a"}, []float64{1}),
		New("").Heatmap("", []string{"a"}, []string{"b"}, [][]float64{{1}}).Box("", []float64{1}),
	} {
		if c.Err() == nil {
			t.Fatal("incompatible layers accepted")
		}
	}
}

func TestBuilderCopiesInputs(t *testing.T) {
	x, y := []float64{1, 2}, []float64{3, 4}
	labels := []string{"a", "b"}
	rows := []string{"r"}
	z := [][]float64{{1, 2}}
	c := New("").Line("", x, y)
	b := New("").Bar("", labels, y)
	h := New("").Heatmap("", labels, rows, z)
	x[0] = 99
	y[0] = 99
	labels[0] = "changed"
	rows[0] = "changed"
	z[0][0] = 99
	if c.layers[0].x[0] != 1 || c.layers[0].y[0] != 3 || b.layers[0].labels[0] != "a" || b.layers[0].y[0] != 3 || h.layers[0].z[0][0] != 1 || h.layers[0].yLabels[0] != "r" {
		t.Fatal("input aliases retained")
	}
	samples := []float64{3, 1, 2}
	ecdf := New("").ECDF("", samples)
	hist := New("").Histogram("", samples, 2)
	box := New("").Box("", samples)
	samples[0] = 100
	if !reflect.DeepEqual(ecdf.layers[0].x, []float64{1, 2, 3}) || !reflect.DeepEqual(ecdf.layers[0].y, []float64{1.0 / 3, 2.0 / 3, 1}) || hist.layers[0].y[0] != 3 || box.layers[0].y[0] != 3 {
		t.Fatal("samples not prepared/copied")
	}
}

func TestBuilderFirstErrorIsSticky(t *testing.T) {
	c := New("bad\x00")
	first := c.Err()
	c.Size(-1, -1).Labels("bad\x00", "").Line("", nil, nil).Dark().LogX()
	if c.Err() != first {
		t.Fatal("first error replaced")
	}
	var nilChart *Chart
	if nilChart.Err() == nil {
		t.Fatal("nil chart has no error")
	}
}

func TestBuilderValidXMLText(t *testing.T) {
	c := New("Δ <data> & values\n").Labels("\t時間", "🙂").Line("a & b", []float64{1}, []float64{2})
	if err := c.Err(); err != nil {
		t.Fatal(err)
	}
}

func TestBuilderNonfiniteHeatmapAndInvalidYLog(t *testing.T) {
	if New("").Heatmap("", []string{"x"}, []string{"y"}, [][]float64{{math.NaN()}}).Err() == nil {
		t.Fatal("nonfinite heatmap accepted")
	}
	for _, add := range []func(*Chart) *Chart{
		func(c *Chart) *Chart { return c.Bar("", []string{"a"}, []float64{1}) },
		func(c *Chart) *Chart { return c.Histogram("", []float64{1}, 1) },
		func(c *Chart) *Chart { return c.Box("", []float64{1}) },
	} {
		if add(New("").LogY()).Err() == nil {
			t.Fatal("unsupported log y layer accepted")
		}
	}
}
