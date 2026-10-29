package chartframe

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Table owns a rectangular CSV table. Accessors return independent slices.
type Table struct {
	headers []string
	rows    [][]string
}

// ReadCSV reads a header followed by strictly rectangular UTF-8 CSV records.
func ReadCSV(reader io.Reader) (*Table, error) {
	if reader == nil {
		return nil, fmt.Errorf("CSV reader is nil")
	}
	parser := csv.NewReader(reader)
	headers, err := parser.Read()
	if err != nil {
		return nil, fmt.Errorf("CSV header: %w", err)
	}
	seen := make(map[string]bool, len(headers))
	for i, h := range headers {
		if strings.TrimSpace(h) == "" || !utf8.ValidString(h) {
			return nil, fmt.Errorf("CSV header column %d is empty or invalid UTF-8", i+1)
		}
		if seen[h] {
			return nil, fmt.Errorf("CSV duplicate header %q", h)
		}
		seen[h] = true
	}
	table := &Table{headers: append([]string(nil), headers...)}
	for row := 2; ; row++ {
		record, err := parser.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("CSV row %d: %w", row, err)
		}
		for column, value := range record {
			if !utf8.ValidString(value) {
				return nil, fmt.Errorf("CSV row %d column %q is invalid UTF-8", row, headers[column])
			}
		}
		table.rows = append(table.rows, append([]string(nil), record...))
	}
	return table, nil
}

// Headers returns the column names in their original order.
func (t *Table) Headers() []string { return append([]string(nil), t.headers...) }

func (t *Table) columnIndex(name string) (int, error) {
	for i, h := range t.headers {
		if h == name {
			return i, nil
		}
	}
	return 0, fmt.Errorf("CSV column %q not found", name)
}

// Column returns all values for the named column, preserving empty strings.
func (t *Table) Column(name string) ([]string, error) {
	index, err := t.columnIndex(name)
	if err != nil {
		return nil, err
	}
	values := make([]string, len(t.rows))
	for i, row := range t.rows {
		values[i] = row[index]
	}
	return values, nil
}

// FloatColumn parses finite decimal or Go-supported floating-point values.
// Blank values, surrounding whitespace, and nonfinite numbers are errors.
func (t *Table) FloatColumn(name string) ([]float64, error) {
	index, err := t.columnIndex(name)
	if err != nil {
		return nil, err
	}
	values := make([]float64, len(t.rows))
	for i, row := range t.rows {
		value, err := strconv.ParseFloat(row[index], 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, fmt.Errorf("CSV row %d column %q: invalid finite number %q", i+2, name, row[index])
		}
		values[i] = value
	}
	return values, nil
}

// XY returns two numerical columns without dropping or reordering records.
func (t *Table) XY(x, y string) ([]float64, []float64, error) {
	xs, err := t.FloatColumn(x)
	if err != nil {
		return nil, nil, err
	}
	ys, err := t.FloatColumn(y)
	if err != nil {
		return nil, nil, err
	}
	return xs, ys, nil
}
