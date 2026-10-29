package chartframe

import (
	"fmt"
	"io"
	"math"
	"testing"
)

func BenchmarkSVG(b *testing.B) {
	for _, n := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			x, y := make([]float64, n), make([]float64, n)
			for i := range x {
				x[i] = float64(i)
				y[i] = math.Sin(float64(i) / 50)
			}
			p := New("Signal").Line("sine", x, y)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := p.Write(io.Discard, "svg"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
