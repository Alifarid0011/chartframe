// Command gallery regenerates the static gallery and downloadable chart assets.
// Run it from the repository root: go run ./cmd/gallery.
package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"html/template"
	"os"
	"path/filepath"

	cf "github.com/Alifarid0011/chartframe"
)

//go:embed page.html
var page string

type example struct {
	ID, Title, Description, Kind, Code string
	Chart                              *cf.Chart
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if _, err := os.Stat("go.mod"); err != nil {
		return fmt.Errorf("run gallery from the repository root: %w", err)
	}
	examples := gallery()
	charts := make([]*cf.Chart, 0, len(examples))
	output := filepath.Join("docs", "charts")
	if err := os.MkdirAll(output, 0755); err != nil {
		return err
	}
	for i := range examples {
		e := &examples[i]
		source, err := format.Source([]byte("package main\n\nimport (\n\"log\"\ncf \"github.com/Alifarid0011/chartframe\"\n)\n\nfunc main() {\n" + e.Code + "\nif err := c.Save(\"" + e.ID + ".svg\"); err != nil { log.Fatal(err) }\n}\n"))
		if err != nil {
			return fmt.Errorf("format %s example: %w", e.ID, err)
		}
		e.Code = string(source)
		for _, extension := range []string{"svg", "png", "html"} {
			if err := e.Chart.Save(filepath.Join(output, e.ID+"."+extension)); err != nil {
				return fmt.Errorf("render %s.%s: %w", e.ID, extension, err)
			}
		}
		charts = append(charts, e.Chart)
	}
	report := cf.Grid("Chartframe / synthetic research notebook", 2, charts...)
	for _, extension := range []string{"svg", "pdf", "html"} {
		if err := report.Save(filepath.Join("docs", "research."+extension)); err != nil {
			return fmt.Errorf("render research.%s: %w", extension, err)
		}
	}
	tmpl, err := template.New("gallery").Parse(page)
	if err != nil {
		return err
	}
	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, examples); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join("docs", "index.html"), rendered.Bytes(), 0644); err != nil {
		return err
	}
	fmt.Printf("Generated docs/index.html, %d chart downloads, and 3 research reports.\n", len(examples)*3)
	return nil
}

func gallery() []example {
	x := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	y := []float64{12, 16, 15, 22, 24, 23, 30, 32, 31, 38, 40, 43}
	labels := []string{"North", "South", "East", "West"}
	baseline := []float64{42, 35, 48, 38}
	current := []float64{54, 44, 57, 51}
	samples := []float64{18, 22, 25, 26, 27, 28, 29, 30, 30, 31, 32, 33, 34, 34, 35, 36, 36, 37, 38, 38, 39, 40, 40, 41, 42, 42, 43, 44, 45, 46, 47, 48, 49, 50, 52, 56}
	groupA := []float64{20, 22, 23, 24, 24, 25, 26, 27, 29, 32}
	groupB := []float64{24, 25, 27, 28, 29, 30, 31, 33, 35, 39}
	groupC := []float64{19, 21, 22, 23, 24, 25, 26, 28, 31, 36}
	days := []string{"Mon", "Tue", "Wed", "Thu", "Fri"}
	periods := []string{"09:00", "12:00", "15:00", "18:00"}
	activity := [][]float64{{12, 18, 15, 22, 19}, {32, 41, 35, 44, 38}, {24, 30, 28, 36, 32}, {16, 22, 18, 25, 21}}
	return []example{
		{ID: "trend", Title: "Find the signal", Kind: "Line · Scatter · Linear fit", Description: "Monthly observations, their connected path, and an ordinary least-squares fit. Synthetic values show a gradual upward trend.",
			Chart: cf.New("Monthly observations").Line("observed", x, y).Scatter("samples", x, y).LinearFit("least-squares", x, y).Labels("Month", "Response").Size(720, 430),
			Code:  fmt.Sprintf("x := %#v\ny := %#v\nc := cf.New(\"Monthly observations\").\nLine(\"observed\", x, y).\nScatter(\"samples\", x, y).\nLinearFit(\"least-squares\", x, y).\nLabels(\"Month\", \"Response\")", x, y)},
		{ID: "categories", Title: "Compare the groups", Kind: "Grouped bar", Description: "Two measurement periods across four regions. Matching category labels keep the series aligned; the vertical axis includes zero.",
			Chart: cf.New("Regional comparison").Bar("baseline", labels, baseline).Bar("current", labels, current).Labels("Region", "Units").Size(720, 430),
			Code:  fmt.Sprintf("labels := %#v\nc := cf.New(\"Regional comparison\").\nBar(\"baseline\", labels, %#v).\nBar(\"current\", labels, %#v).\nLabels(\"Region\", \"Units\")", labels, baseline, current)},
		{ID: "distribution", Title: "See the distribution", Kind: "Histogram", Description: "Thirty-six synthetic measurements in eight equal-width bins. Every sample is counted, including observations on the final edge.",
			Chart: cf.New("Measurement distribution").Histogram("samples", samples, 8).Labels("Measurement", "Count").Size(720, 430),
			Code:  fmt.Sprintf("values := %#v\nc := cf.New(\"Measurement distribution\").\nHistogram(\"samples\", values, 8).\nLabels(\"Measurement\", \"Count\")", samples)},
		{ID: "summary", Title: "Put spread in context", Kind: "Box plot", Description: "Three small cohorts compared by median, quartiles, and spread. Quartiles use sorted linear interpolation (R type 7).",
			Chart: cf.New("Cohort spread").Box("A", groupA).Box("B", groupB).Box("C", groupC).Labels("Cohort", "Measurement").Size(720, 430),
			Code:  fmt.Sprintf("c := cf.New(\"Cohort spread\").\nBox(\"A\", %#v).\nBox(\"B\", %#v).\nBox(\"C\", %#v).\nLabels(\"Cohort\", \"Measurement\")", groupA, groupB, groupC)},
		{ID: "matrix", Title: "Spot the pattern", Kind: "Heatmap", Description: "A labeled weekday-by-time matrix. Color encodes relative activity; the color legend gives the numerical range.",
			Chart: cf.New("Activity by weekday and time").Heatmap("Activity", days, periods, activity).Labels("Weekday", "Time").Size(720, 430),
			Code:  fmt.Sprintf("days := %#v\ntimes := %#v\nvalues := %#v\nc := cf.New(\"Activity by weekday and time\").\nHeatmap(\"Activity\", days, times, values).\nLabels(\"Weekday\", \"Time\")", days, periods, activity)},
		{ID: "cumulative", Title: "Read the whole sample", Kind: "Empirical CDF", Description: "The same measurements as a cumulative step curve. Read the fraction of observations at or below a chosen measurement.",
			Chart: cf.New("Cumulative measurements").ECDF("samples", samples).Labels("Measurement", "Cumulative fraction").Size(720, 430),
			Code:  fmt.Sprintf("values := %#v\nc := cf.New(\"Cumulative measurements\").\nECDF(\"samples\", values).\nLabels(\"Measurement\", \"Cumulative fraction\")", samples)},
	}
}
