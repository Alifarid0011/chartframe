# API guide

## Builder

`New(title)` creates a 900×540 chart. `.Size(w,h)` takes logical pixels at 96 dpi.
Data validation is eager, but errors accumulate as the first error on the builder:

```go
p := chartframe.New("Signal").Labels("s", "V").Line("measured", x, y)
if err := p.Err(); err != nil { return err }
return p.Save("signal.png")
```

Use `Write(writer, "png")`, `SVG()` or `HTML()` for buffers and HTTP responses.
No export starts a process or opens a browser. `Grid(title, columns, charts...)`
has the same export methods. Finish changing all panel builders before rendering.

## Distributions

`Describe(samples)` returns count, extrema, mean, median, and quartiles. Quartiles
use linear interpolation at `(n-1)*p`, commonly called R type 7. Box charts use the
same quartiles; whiskers end at observed values within 1.5 IQR of Q1 and Q3, and
observations outside those fences are rendered as points.

`Bin(samples, n)` returns `n+1` edges and `n` integer counts. Bins are left-closed
and right-open except the last, which includes its upper edge. Constant samples
receive a finite enclosing range; errors report when the requested finite distinct
edges cannot be represented. Histogram constructors require 1–10000 bins.

`ECDF` sorts a copy of the samples and draws a right-continuous step function. The
linear-Y chart includes the initial jump from zero; log-Y omits that zero boundary.

## Regression

`FitLinear(x,y)` returns slope, intercept and R² using centered, scaled ordinary
least squares. It rejects fewer than two samples, constant X, nonfinite input and
unrepresentable results. For constant Y, R² is defined as 1. `LinearFit` adds the
fitted endpoint line, using a fused multiply-add to avoid intermediate overflow.
It is a linear fit, not a statistical significance test or confidence interval.

## CSV and sheetframe

```go
table, err := chartframe.ReadCSV(reader)
if err != nil { return err }
x, y, err := table.XY("time", "voltage")
if err != nil { return err }
return chartframe.New("Voltage").Line("measured", x, y).Save("voltage.svg")
```

Headers must be nonempty and unique. Records have the header's field count.
Numeric conversion trims surrounding whitespace and rejects blanks, invalid
numbers and nonfinite values. Error rows are CSV record numbers including the
header; a quoted multiline cell is still one record. Accessors return copies.

For sheetframe data, write the selected numeric columns with `sheetframe.WriteCSV`
and pass a reader to `ReadCSV`, or extract float64 values with `Frame.At` into two
slices. Null handling stays explicit in the calling analysis. chartframe does not
import sheetframe or Excelize, keeping plotting independent of spreadsheet I/O.

## Numerical and rendering limits

Representable input does not guarantee a representable derived range. A linear
axis spanning `-MaxFloat64` to `MaxFloat64`, indistinguishable logarithms, or a
histogram with colliding float64 edges returns an error. No input point is silently
removed to force a successful rendering. CSV and statistical helpers use float64;
use another numerical representation when that precision is insufficient.
