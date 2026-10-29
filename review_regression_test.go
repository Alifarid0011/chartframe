package chartframe

import "testing"

func TestLinearFitFiniteEndpoints(t *testing.T) {
	c := New("large offset").LinearFit("fit", []float64{8e307, 9e307}, []float64{0, 2e307})
	if err := c.Err(); err != nil {
		t.Fatalf("finite fit rejected: %v", err)
	}
	if _, err := c.SVG(); err != nil {
		t.Fatal(err)
	}
}

func TestECDFStartsAtZero(t *testing.T) {
	for _, values := range [][]float64{{2}, {2, 2, 2}} {
		p, err := New("point mass").ECDF("sample", values).buildPlot()
		if err != nil {
			t.Fatal(err)
		}
		if p.Y.Min != 0 || p.Y.Max < 1 {
			t.Fatalf("ECDF jump must span zero to one, range [%g,%g]", p.Y.Min, p.Y.Max)
		}
	}
}
