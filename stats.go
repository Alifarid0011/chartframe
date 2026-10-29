package chartframe

import (
	"fmt"
	"math"
	"sort"
)

// Regression describes an ordinary least-squares line y = Slope*x + Intercept.
// RSquared is one for a constant response fitted exactly by a horizontal line.
type Regression struct{ Slope, Intercept, RSquared float64 }

// Summary describes a sample, using R type 7 interpolated quartiles.
type Summary struct {
	Count                          int
	Min, Q1, Median, Q3, Max, Mean float64
}

// HistogramData contains left-closed bins; the final bin includes its right edge.
type HistogramData struct {
	Edges  []float64
	Counts []int
}

func sampleBounds(values []float64) (float64, float64, error) {
	if len(values) == 0 {
		return 0, 0, fmt.Errorf("sample is empty")
	}
	lo, hi := values[0], values[0]
	for i, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, 0, fmt.Errorf("sample value %d is nonfinite", i)
		}
		lo = math.Min(lo, v)
		hi = math.Max(hi, v)
	}
	return lo, hi, nil
}

// compensatedSum reduces rounding error when observations cancel.
func compensatedSum(values []float64) float64 {
	var sum, correction float64
	for _, v := range values {
		next := sum + v
		if math.Abs(sum) >= math.Abs(v) {
			correction += (sum - next) + v
		} else {
			correction += (v - next) + sum
		}
		sum = next
	}
	return sum + correction
}

func interpolate(a, b, t float64) float64 {
	if t == 0 {
		return a
	}
	if t == 1 {
		return b
	}
	if math.Signbit(a) == math.Signbit(b) {
		return a + (b-a)*t
	}
	return a*(1-t) + b*t
}

// Describe returns descriptive statistics without modifying values.
func Describe(values []float64) (Summary, error) {
	lo, hi, err := sampleBounds(values)
	if err != nil {
		return Summary{}, err
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	quantile := func(p float64) float64 {
		index := float64(len(sorted)-1) * p
		i := int(index)
		if i == len(sorted)-1 {
			return sorted[i]
		}
		return interpolate(sorted[i], sorted[i+1], index-float64(i))
	}
	scale := math.Max(math.Abs(lo), math.Abs(hi))
	mean := 0.0
	if scale != 0 {
		normalized := make([]float64, len(values))
		for i, v := range values {
			normalized[i] = v / scale
		}
		mean = (compensatedSum(normalized) / float64(len(values))) * scale
	}
	return Summary{Count: len(values), Min: lo, Q1: quantile(.25), Median: quantile(.5), Q3: quantile(.75), Max: hi, Mean: mean}, nil
}

// FitLinear fits a line with an intercept, rejecting constant predictors and
// coefficients that cannot be represented reliably as finite float64 values.
func FitLinear(x, y []float64) (Regression, error) {
	if len(x) != len(y) {
		return Regression{}, fmt.Errorf("regression lengths differ")
	}
	xlo, xhi, err := sampleBounds(x)
	if err != nil {
		return Regression{}, err
	}
	ylo, yhi, err := sampleBounds(y)
	if err != nil {
		return Regression{}, err
	}
	if xlo == xhi {
		return Regression{}, fmt.Errorf("regression requires nonconstant x")
	}
	if ylo == yhi {
		return Regression{Intercept: ylo, RSquared: 1}, nil
	}
	// Center before scaling so a large common offset does not erase variation.
	xo, yo := interpolate(xlo, xhi, .5), interpolate(ylo, yhi, .5)
	xs, ys := math.Max(math.Abs(xlo-xo), math.Abs(xhi-xo)), math.Max(math.Abs(ylo-yo), math.Abs(yhi-yo))
	nx, ny := make([]float64, len(x)), make([]float64, len(y))
	for i := range x {
		nx[i] = (x[i] - xo) / xs
		ny[i] = (y[i] - yo) / ys
	}
	mx, my := compensatedSum(nx)/float64(len(x)), compensatedSum(ny)/float64(len(y))
	var xx, yy, xy float64
	for i := range x {
		dx, dy := nx[i]-mx, ny[i]-my
		xx += dx * dx
		yy += dy * dy
		xy += dx * dy
	}
	coefficient := xy / xx
	// Frexp avoids overflow in the intermediate scale ratio.
	cm, ce := math.Frexp(coefficient)
	ym, ye := math.Frexp(ys)
	xm, xe := math.Frexp(xs)
	slope := math.Ldexp(cm*ym/xm, ce+ye-xe)
	intercept := math.FMA(-slope, xo, yo) + ys*(my-coefficient*mx)
	correlation := (xy / math.Sqrt(xx)) / math.Sqrt(yy)
	r2 := math.Min(1, math.Max(0, correlation*correlation))
	if math.IsNaN(slope) || math.IsInf(slope, 0) || math.IsNaN(intercept) || math.IsInf(intercept, 0) || math.IsNaN(r2) || (coefficient != 0 && slope == 0) {
		return Regression{}, fmt.Errorf("regression is numerically unstable")
	}
	return Regression{Slope: slope, Intercept: intercept, RSquared: r2}, nil
}

// Bin groups values into equal-width bins, preserving every observation.
// Constant samples receive an enclosing finite range. Unrepresentable edges
// produce an error instead of collapsed or nonfinite bins.
func Bin(values []float64, bins int) (HistogramData, error) {
	if bins < 1 || bins > 10000 {
		return HistogramData{}, fmt.Errorf("bins must be between 1 and 10000")
	}
	lo, hi, err := sampleBounds(values)
	if err != nil {
		return HistogramData{}, err
	}
	if lo == hi {
		padding := math.Abs(lo) * .05
		if padding == 0 {
			padding = .5
		}
		lower, upper := lo-padding, hi+padding
		if !math.IsInf(lower, 0) {
			lo = lower
		}
		if !math.IsInf(upper, 0) {
			hi = upper
		}
	}
	edges := make([]float64, bins+1)
	for i := range edges {
		edges[i] = interpolate(lo, hi, float64(i)/float64(bins))
		if math.IsNaN(edges[i]) || math.IsInf(edges[i], 0) || (i > 0 && edges[i] <= edges[i-1]) {
			return HistogramData{}, fmt.Errorf("histogram range cannot represent %d distinct finite bins", bins)
		}
	}
	counts := make([]int, bins)
	for _, v := range values {
		index := sort.Search(len(edges), func(i int) bool { return edges[i] > v }) - 1
		if index == bins {
			index--
		}
		counts[index]++
	}
	return HistogramData{Edges: edges, Counts: counts}, nil
}
