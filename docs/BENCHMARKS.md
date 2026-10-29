# Benchmarks

These measurements describe this implementation, not a comparison with Python,
Matplotlib, or other Go libraries. Rendering uses Gonum Plot v0.17.0.

Measured on Linux amd64, Intel Core i3-1215U, Go 1.26.8, GOMAXPROCS=2.
Three repetitions, 200 ms minimum per benchmark. Other applications were running;
these are development-machine measurements, not a controlled performance study.

| Operation | Input | Median time | Allocation budget |
| --- | --- | --- | --- |
| SVG export | 100 points | 1.145 ms | about 627 KB |
| SVG export | 1,000 points | 4.028 ms | about 2.84 MB |
| SVG export | 10,000 points | 38.072 ms | about 31.84 MB |
| Linear regression | 1,000 pairs | 70.556 µs | 16,384 B, 2 allocations |
| Summary statistics | 1,000 values | 108.005 µs | 16,384 B, 2 allocations |

SVG export includes constructing the rendering plot, serializing the document,
and writing to `io.Discard`. Chart construction and input generation are outside
the timed loop. Statistics include validation and internal copies. Timings exclude
compilation and startup. SVG memory use grows with point count; reduce or aggregate
large series before rendering when an overview is sufficient.

Reproduce from the module root:

```sh
GOMAXPROCS=2 go test -run '^$' -bench . -benchmem -count 3 -benchtime 200ms ./...
```

[Raw output](benchmarks-go1.26.8-linux-amd64.txt) and benchmark sources are included.
No universal speed advantage or parity with Matplotlib is claimed.
