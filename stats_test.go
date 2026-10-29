package chartframe

import (
	"math"
	"reflect"
	"testing"
)

func TestFitLinear(t *testing.T) {
	for _, tt := range []struct {
		name                 string
		x, y                 []float64
		slope, intercept, r2 float64
	}{
		{"line", []float64{1, 2, 3, 4}, []float64{3, 5, 7, 9}, 2, 1, 1},
		{"constant response", []float64{1, 2, 3}, []float64{7, 7, 7}, 0, 7, 1},
		{"no correlation", []float64{-1, 0, 1}, []float64{1, 0, 1}, 0, 2.0 / 3, 0},
		{"large offset", []float64{1e15, 1e15 + 1, 1e15 + 2}, []float64{1, 3, 5}, 2, 1 - 2e15, 1},
		{"large magnitude", []float64{-1e200, 0, 1e200}, []float64{-2e200, 0, 2e200}, 2, 0, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FitLinear(tt.x, tt.y)
			if err != nil {
				t.Fatal(err)
			}
			for _, v := range [][2]float64{{got.Slope, tt.slope}, {got.Intercept, tt.intercept}, {got.RSquared, tt.r2}} {
				if math.Abs(v[0]-v[1]) > 1e-12*math.Max(1, math.Abs(v[1])) {
					t.Errorf("got %+v", got)
				}
			}
		})
	}
	for _, tt := range []struct{ x, y []float64 }{{nil, nil}, {[]float64{1}, []float64{2}}, {[]float64{1, 1}, []float64{2, 3}}, {[]float64{1, 2}, []float64{1}}, {[]float64{1, math.Inf(1)}, []float64{1, 2}}, {[]float64{0, math.SmallestNonzeroFloat64}, []float64{0, math.MaxFloat64}}} {
		if _, err := FitLinear(tt.x, tt.y); err == nil {
			t.Errorf("accepted %+v", tt)
		}
	}
}

func TestDescribe(t *testing.T) {
	values := []float64{4, 1, 3, 2}
	before := append([]float64(nil), values...)
	got, err := Describe(values)
	if err != nil {
		t.Fatal(err)
	}
	want := Summary{Count: 4, Min: 1, Q1: 1.75, Median: 2.5, Q3: 3.25, Max: 4, Mean: 2.5}
	if got != want {
		t.Errorf("got %+v want %+v", got, want)
	}
	if !reflect.DeepEqual(values, before) {
		t.Fatal("input modified")
	}
	for _, v := range [][]float64{{9}, {math.MaxFloat64, math.MaxFloat64}, {-math.MaxFloat64, math.MaxFloat64}} {
		s, e := Describe(v)
		if e != nil {
			t.Fatal(e)
		}
		if math.IsInf(s.Mean, 0) || math.IsNaN(s.Median) {
			t.Errorf("unstable %+v", s)
		}
	}
	for _, v := range [][]float64{nil, {math.NaN()}, {math.Inf(-1)}} {
		if _, e := Describe(v); e == nil {
			t.Errorf("accepted %v", v)
		}
	}
}

func TestBin(t *testing.T) {
	got, err := Bin([]float64{0, 1, 2, 3, 4}, 4)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Edges, []float64{0, 1, 2, 3, 4}) || !reflect.DeepEqual(got.Counts, []int{1, 1, 1, 2}) {
		t.Errorf("got %+v", got)
	}
	for _, v := range [][]float64{{0, 0}, {3, 3}, {math.MaxFloat64, math.MaxFloat64}, {-math.MaxFloat64, math.MaxFloat64}} {
		h, e := Bin(v, 10)
		if e != nil {
			t.Fatal(e)
		}
		n := 0
		for i, c := range h.Counts {
			n += c
			if !(h.Edges[i] < h.Edges[i+1]) || math.IsInf(h.Edges[i], 0) {
				t.Fatalf("invalid %+v", h)
			}
		}
		if n != len(v) || len(h.Edges) != 11 {
			t.Fatalf("invalid %+v", h)
		}
	}
	for _, tt := range []struct {
		v []float64
		b int
	}{{nil, 1}, {[]float64{1}, 0}, {[]float64{1}, 10001}, {[]float64{math.NaN()}, 1}, {[]float64{1, math.Nextafter(1, 2)}, 2}} {
		if _, e := Bin(tt.v, tt.b); e == nil {
			t.Errorf("accepted %+v", tt)
		}
	}
}

func FuzzBinConservesSamples(f *testing.F) {
	f.Add(uint64(0), math.Float64bits(1), uint16(4))
	f.Add(math.Float64bits(-math.MaxFloat64), math.Float64bits(math.MaxFloat64), uint16(2))
	f.Add(math.Float64bits(1), math.Float64bits(math.Nextafter(1, 2)), uint16(2))
	f.Fuzz(func(t *testing.T, a, b uint64, bins uint16) {
		values := []float64{math.Float64frombits(a), math.Float64frombits(b)}
		count := int(bins%64) + 1
		histogram, err := Bin(values, count)
		if err != nil {
			return
		}
		if len(histogram.Edges) != count+1 || len(histogram.Counts) != count {
			t.Fatal("incorrect output dimensions")
		}
		total := 0
		for i, n := range histogram.Counts {
			total += n
			if n < 0 || !(histogram.Edges[i] < histogram.Edges[i+1]) {
				t.Fatal("invalid bins")
			}
		}
		if total != len(values) {
			t.Fatal("samples lost")
		}
		for _, v := range values {
			if math.IsNaN(v) || math.IsInf(v, 0) || v < histogram.Edges[0] || v > histogram.Edges[count] {
				t.Fatal("sample outside histogram")
			}
		}
	})
}

func BenchmarkFitLinear(b *testing.B) {
	x, y := make([]float64, 1000), make([]float64, 1000)
	for i := range x {
		x[i] = float64(i)
		y[i] = 2*x[i] + 1
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := FitLinear(x, y); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDescribe(b *testing.B) {
	values := make([]float64, 1000)
	for i := range values {
		values[i] = float64((i * 37) % 1000)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := Describe(values); err != nil {
			b.Fatal(err)
		}
	}
}
