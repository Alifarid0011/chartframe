# chartframe

**A small Go API for scientific charts.**

Build a plot from slices, inspect a distribution, fit a line, and export a report.
Native Go calculations and [Gonum Plot](https://github.com/gonum/plot) rendering;
no Python runtime, GUI toolkit, browser service, or CDN required.

Built by [alifarid](https://github.com/Alifarid0011).

[Gallery](https://alifarid0011.github.io/chartframe/) · [API guide](docs/API.md) ·
[Benchmarks](docs/BENCHMARKS.md) · [GoBook notebooks](https://github.com/Alifarid0011/gobook)

## Install

Go 1.25 or newer:

```sh
go get github.com/Alifarid0011/chartframe@v0.1.0
```

```go
package main

import (
    "log"
    cf "github.com/Alifarid0011/chartframe"
)

func main() {
    x := []float64{1, 2, 3, 4, 5}
    y := []float64{2.1, 3.8, 6.2, 7.9, 10.1}
    p := cf.New("Experiment 01").
        Labels("Concentration", "Response").
        Scatter("observations", x, y).
        LinearFit("least-squares fit", x, y)
    if err := p.Save("experiment.svg"); err != nil { log.Fatal(err) }
    if err := p.Save("experiment.html"); err != nil { log.Fatal(err) }
}
```

The HTML file is self-contained. Use buttons or the wheel to zoom, drag to pan,
reset the view, and download SVG. This is **viewport navigation**, not interactive
recomputation of axes, tooltips, or a live plotting server.

## Charts and analysis

| Task | API |
| --- | --- |
| Curves and measurements | `Line`, `Scatter`, `LinearFit` |
| Compare categories | `Bar`, including grouped bars and negative values |
| Distributions | `Histogram`, `Box`, `ECDF` |
| Matrices | `Heatmap`, first matrix row displayed at the top |
| Reports | `Grid`, `Save`, `Write` — SVG, PNG, PDF, HTML |
| Presentation | `Labels`, `Size`, `Dark`, `LogX`, `LogY` |
| Numeric helpers | `Describe`, `FitLinear`, `Bin` |
| CSV input | `ReadCSV`, `FloatColumn`, `XY` |

```go
hist := cf.New("Latency distribution").Histogram("latency", samples, 20)
box := cf.New("Two cohorts").Box("control", control).Box("treatment", treatment)
report := cf.Grid("Study results", 2, hist, box)
if err := report.Save("report.pdf"); err != nil { return err }
```

Statistics use type 7 quartiles, ordinary least squares, and explicit fixed-width
histogram bins. The last histogram bin includes the maximum. Invalid or
unrepresentable numeric results return errors instead of silently dropping samples.

## Data without surprises

Inputs are copied. A builder remembers its first error, reported by `Err` or the
next export. Empty data, malformed shapes, nonfinite values, and invalid text are
rejected. CSV conversion is strict: blanks and nonnumeric cells report their row
and column. No implicit missing-value deletion changes the population you plot.

Numeric line/scatter/histogram/ECDF layers can overlay. Bars require matching
categories; box plots combine with other boxes; heatmaps stand alone. Categorical
charts, boxes, and histograms currently reject log axes. Numeric logarithmic
coordinates must be positive and distinguishable at float64 precision.

`Save` renders before touching the destination, then renames a sibling temporary
file. New files are private (`0600`); replacement behavior follows `os.Rename` on
the host platform. `Write` propagates errors and short writes. Construct a chart
before rendering it concurrently; simultaneous builder mutation needs a lock.

## Scope

This is an **early v0.1 plotting package**, not a Matplotlib replacement or Python
compatibility layer. Gonum supplies the drawing backends and fonts; chartframe
adds the builder, validation, statistics, CSV workflow, offline HTML, and grids.

There is no 3D, animation, GPU engine, GUI event loop, LaTeX engine, notebook
kernel, arbitrary annotations, or automatic downsampling. Very large datasets
produce correspondingly large vector files. Single charts are 240–4096 pixels
per dimension. Grids are limited to 16 panels, 8192 pixels per dimension and
32 megapixels. Export errors explain unsupported numeric ranges.

Native Go can suit concurrent simulations and typed data pipelines. This package
does not claim to outperform Matplotlib, NumPy, pandas, or Gonum. Published Go
benchmarks report the operations actually measured, including allocations.

## Develop

```sh
go test -race ./...
go vet ./...
go run ./cmd/gallery       # regenerate the actual Go-rendered gallery
go test -run '^$' -bench . -benchmem
```

MIT license for chartframe. Dependencies keep their own licenses; Gonum Plot is
BSD-3-Clause. See [NOTICE](NOTICE).
