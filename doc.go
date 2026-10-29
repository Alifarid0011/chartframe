// Package chartframe provides Go charts for exploratory data analysis and reports.
//
// Build plots with New and its chainable methods, then export SVG, PNG, PDF or
// self-contained HTML. Builders copy all input slices and retain their first
// validation error. Call Err to validate construction before exporting.
//
// Charts support concurrent reads and renders after construction. Mutations
// require external synchronization. Numeric series can be overlaid; categorical
// bars require matching labels, box plots combine only with boxes, and heatmaps
// occupy a chart on their own. Logarithmic axes require positive coordinates and
// are supported for line, scatter, ECDF, and fitted-line series.
package chartframe
