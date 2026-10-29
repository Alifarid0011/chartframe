# chartframe implementation plan

Goal: deliver the v0.1 contract in design.md as a tested, documented Go module.
Architecture: immutable copied layer data in a sticky-error Chart builder; Gonum
rendering builds a fresh backend per export; offline SVG HTML and bounded grids.

1. chart.go/chart_test.go: implement and test the builder, copying, finite values,
   categorical compatibility, dimensions, log scales, and first-error propagation.
2. stats.go/csv.go and tests: known regression/quartile/bin results and strict CSV
   parsing, failure cases, immutable results, and numerical range boundaries.
3. render.go/export.go/html.go: test signatures/decoded dimensions of all outputs,
   XML validity, writer failures and short writes, repeatability, atomic save, grids,
   label escaping, SVG viewport controls and offline exports; implement Gonum adapters.
4. Examples/gallery and documentation: generate each chart using the real Go API,
   provide runnable examples and measured benchmarks with environment/methodology.
5. Independently review code; run race tests, vet, fuzz smoke checks, benchmarks,
   build the gallery, inspect the rendered page, then commit and publish to the user's
   GitHub repository. Report unsupported cases and any failed verification honestly.

Review focus: huge/nonfinite float ranges, constant and empty samples, zero baselines
on logarithmic axes, mutation after adding layers, escaping hostile labels in HTML/SVG,
nil/short writers and interrupted saves, filesystem permissions, and dashboard bounds.
