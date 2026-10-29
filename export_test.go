package chartframe

import (
	"bytes"
	"encoding/xml"
	"errors"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOutputFormats(t *testing.T) {
	c := New("Measured <signal>").Size(640, 400).Labels("time", "value").
		Line("observed", []float64{0, 1, 2}, []float64{1, 3, 2})
	for _, format := range []string{"svg", "png", "pdf", "html"} {
		t.Run(format, func(t *testing.T) {
			var b bytes.Buffer
			if err := c.Write(&b, format); err != nil {
				t.Fatal(err)
			}
			switch format {
			case "svg":
				d := xml.NewDecoder(bytes.NewReader(b.Bytes()))
				for {
					_, err := d.Token()
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Fatal(err)
					}
				}
				if !bytes.Contains(b.Bytes(), []byte("&lt;signal&gt;")) {
					t.Fatal("title not escaped")
				}
			case "png":
				cfg, err := png.DecodeConfig(bytes.NewReader(b.Bytes()))
				if err != nil || cfg.Width != 640 || cfg.Height != 400 {
					t.Fatalf("PNG config: %+v, %v", cfg, err)
				}
			case "pdf":
				if !bytes.HasPrefix(b.Bytes(), []byte("%PDF-")) {
					t.Fatal("missing PDF signature")
				}
			case "html":
				if strings.Contains(b.String(), "https://") || !strings.Contains(b.String(), "viewBox") {
					t.Fatal("HTML must be self-contained and zoomable")
				}
			}
		})
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return len(p) / 2, nil }

func TestWriteAndSaveFailures(t *testing.T) {
	c := New("test").Line("x", []float64{1, 2}, []float64{2, 4})
	want := errors.New("write failed")
	if err := c.Write(failingWriter{want}, "svg"); !errors.Is(err, want) {
		t.Fatalf("lost writer error: %v", err)
	}
	if err := c.Write(shortWriter{}, "svg"); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short write: %v", err)
	}
	if err := c.Write(nil, "svg"); err == nil {
		t.Fatal("nil writer accepted")
	}
	if err := c.Write(io.Discard, "exe"); err == nil {
		t.Fatal("unsupported format accepted")
	}
	path := filepath.Join(t.TempDir(), "existing.svg")
	if err := os.WriteFile(path, []byte("keep me"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := New("invalid").Save(path); err == nil {
		t.Fatal("empty chart saved")
	}
	got, _ := os.ReadFile(path)
	if string(got) != "keep me" {
		t.Fatal("failed render overwrote destination")
	}
	if err := c.Save(path); err != nil {
		t.Fatal(err)
	}
	got, _ = os.ReadFile(path)
	if !bytes.Contains(got, []byte("<svg")) {
		t.Fatal("saved SVG missing")
	}
}

func TestAllChartKindsAndGrid(t *testing.T) {
	charts := []*Chart{
		New("scatter + fit").Scatter("a", []float64{1, 2, 3}, []float64{2, 4, 6}).LinearFit("fit", []float64{1, 2, 3}, []float64{2, 4, 6}),
		New("bars").Bar("a", []string{"A", "B"}, []float64{-1, 3}).Bar("b", []string{"A", "B"}, []float64{2, 4}),
		New("distribution").Histogram("x", []float64{1, 2, 2, 3}, 3),
		New("boxes").Box("A", []float64{1, 2, 3, 4, 20}).Box("B", []float64{2, 3, 4, 5}),
		New("heatmap").Heatmap("matrix", []string{"a", "b"}, []string{"x", "y"}, [][]float64{{1, 2}, {3, 4}}),
		New("ECDF").ECDF("x", []float64{1, 2, 2, 3}).Dark(),
		New("log").Line("x", []float64{1, 10, 100}, []float64{1, 100, 10000}).LogX().LogY(),
		New("constant").Scatter("x", []float64{1}, []float64{2}),
	}
	for _, c := range charts {
		if _, err := c.SVG(); err != nil {
			t.Fatalf("%s: %v", c.title, err)
		}
	}
	if _, err := Grid("Research", 2, charts[:6]...).SVG(); err != nil {
		t.Fatal(err)
	}
	if _, err := Grid("bad", 0, charts[0]).SVG(); err == nil {
		t.Fatal("zero columns accepted")
	}
	if _, err := Grid("bad", 2, nil).SVG(); err == nil {
		t.Fatal("nil chart accepted")
	}
}

func TestExtremeRangeReturnsError(t *testing.T) {
	c := New("overflow").Line("x", []float64{-math.MaxFloat64, math.MaxFloat64}, []float64{1, 2})
	if _, err := c.SVG(); err == nil {
		t.Fatal("unrepresentable axis span accepted")
	}
	lo, hi := math.SmallestNonzeroFloat64, 2*math.SmallestNonzeroFloat64
	_, err := New("subnormal").Line("x", []float64{lo, hi}, []float64{1, 2}).LogX().SVG()
	// Go's architecture-specific Log implementations can round these subnormal
	// inputs to the same coordinate. Reject a collapsed span, but allow it when
	// the platform preserves two distinct logarithms.
	if math.Log(lo) >= math.Log(hi) {
		if err == nil {
			t.Fatal("indistinguishable log coordinates accepted")
		}
	} else if err != nil {
		t.Fatalf("representable subnormal log range: %v", err)
	}
}

func TestHTMLLabelEscaping(t *testing.T) {
	attack := `</script><script>alert("x")</script>`
	b, err := New(attack).Line("x", []float64{1, 2}, []float64{3, 4}).HTML()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), attack) {
		t.Fatal("unescaped label in HTML")
	}
}
