package chartframe_test

import (
	"fmt"
	"io"
	"strings"

	cf "github.com/Alifarid0011/chartframe"
)

func Example() {
	p := cf.New("Experiment").Scatter("sample", []float64{1, 2, 3}, []float64{2, 4, 6})
	err := p.Write(io.Discard, "svg")
	fmt.Println(err)
	// Output: <nil>
}

func ExampleReadCSV() {
	t, err := cf.ReadCSV(strings.NewReader("x,y\n1,2\n2,4\n3,6\n"))
	if err != nil {
		panic(err)
	}
	x, y, err := t.XY("x", "y")
	if err != nil {
		panic(err)
	}
	fit, err := cf.FitLinear(x, y)
	if err != nil {
		panic(err)
	}
	fmt.Printf("slope %.1f, R² %.1f\n", fit.Slope, fit.RSquared)
	// Output: slope 2.0, R² 1.0
}
