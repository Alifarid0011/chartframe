package chartframe

import (
	"reflect"
	"strings"
	"testing"
)

func TestCSVColumns(t *testing.T) {
	table, err := ReadCSV(strings.NewReader("x,y,note\n1,2,\"hello, world\"\n3,4,\"line\nbreak\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	h := table.Headers()
	h[0] = "changed"
	if table.Headers()[0] != "x" {
		t.Fatal("headers mutable")
	}
	c, err := table.Column("note")
	if err != nil || !reflect.DeepEqual(c, []string{"hello, world", "line\nbreak"}) {
		t.Fatalf("%v %v", c, err)
	}
	c[0] = "changed"
	again, _ := table.Column("note")
	if again[0] == "changed" {
		t.Fatal("column mutable")
	}
	x, y, err := table.XY("x", "y")
	if err != nil || !reflect.DeepEqual(x, []float64{1, 3}) || !reflect.DeepEqual(y, []float64{2, 4}) {
		t.Fatalf("%v %v %v", x, y, err)
	}
	if _, e := table.Column("missing"); e == nil {
		t.Fatal("missing column accepted")
	}
	if _, _, e := table.XY("x", "missing"); e == nil {
		t.Fatal("missing XY column accepted")
	}
}

func TestCSVValidation(t *testing.T) {
	for _, s := range []string{"", "x,x\n1,2", "x,\n1,2", "x,y\n1", "x,y\n1,2,3", "x\n\"unclosed", "\xff\n1", "x\n\xff"} {
		if _, e := ReadCSV(strings.NewReader(s)); e == nil {
			t.Errorf("accepted %q", s)
		}
	}
	for _, s := range []string{"NaN", "Inf", "-Inf", "", " 1", "1 ", "1e999", "word"} {
		table, e := ReadCSV(strings.NewReader("x,y\n1,2\n\"" + s + "\",3\n"))
		if e != nil {
			t.Fatal(e)
		}
		if _, e := table.FloatColumn("x"); e == nil || !strings.Contains(e.Error(), "row 3") || !strings.Contains(e.Error(), "x") {
			t.Errorf("value %q: expected row/column error, got %v", s, e)
		}
	}
	table, e := ReadCSV(strings.NewReader("x,y\n"))
	if e != nil {
		t.Fatal(e)
	}
	v, e := table.FloatColumn("x")
	if e != nil || len(v) != 0 {
		t.Fatal(v, e)
	}
}
