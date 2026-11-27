package costs

import (
	"strings"
	"testing"
)

const inputsHeader = `levels:
  - {id: base, name: Base, sqft: 1000, reserve_months: {low: 1, expected: 1, high: 1}}
  - {id: plus, name: Plus, sqft: 1500, reserve_months: {low: 1, expected: 1, high: 1}}
categories:
  - {id: a, name: Alpha}
`

const inputsFixture = inputsHeader + `inputs:
  - {id: rent-year, name: Rent per sq ft per year, value: {low: 17.50, expected: 21, high: 30}}
  - {id: build-months, name: Build-out months, value: {low: 3, expected: 6, high: 12}}
  - {id: free-months, name: Rent-free months, value: 3}
items:
  - id: deposit
    category: a
    level: base
    name: Deposit
    status: estimate
    lines:
      - {quantity: {low: 0, expected: 1, high: 3}, unit_price: item.rent}
  - id: build-rent
    category: a
    level: base
    name: Build-out rent
    status: estimate
    lines:
      - {quantity: "input.build-months - input.free-months", unit_price: item.rent}
  - id: counter
    category: a
    level: base
    name: Counter
    status: estimate
    lines:
      - {quantity: 12, unit_price: {low: 100, expected: 150, high: 500}}
  - id: mural
    category: a
    level: plus
    name: Mural
    status: estimate
    lines:
      - {quantity: "40 * 8", unit_price: {low: 20, expected: 25, high: 40}}
  - id: seating
    category: a
    level: base
    name: Seating
    status: estimate
    lines:
      - {label: tables, quantity: 9, unit_price: {low: 50, expected: 150, high: 300}}
      - {label: seats, quantity: 24, unit_price: {low: 50, expected: 250, high: 400}}
  - id: typed
    category: a
    level: base
    name: Typed
    low: 100
    expected: 200
    high: 300
    status: estimate
monthly:
  - id: rent
    level: base
    name: Rent
    status: estimate
    lines:
      - {quantity: level.base.sqft, unit_price: input.rent-year, divisor: 12}
  - id: rent-plus
    level: plus
    name: Added rent
    status: estimate
    lines:
      - {quantity: level.plus.added_sqft, unit_price: input.rent-year, divisor: 12}
`

func loadErr(t *testing.T, body string) error {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "costs.yaml", body)
	_, err := LoadIn(dir, "costs.yaml")
	return err
}

func wantLoadErr(t *testing.T, body, want string) {
	t.Helper()
	err := loadErr(t, body)
	if err == nil {
		t.Fatalf("load succeeded; want error containing %q", want)
	}
	if !strings.HasPrefix(err.Error(), "costs.yaml: ") || !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q; want prefix %q and %q", err, "costs.yaml: ", want)
	}
}

// oneItem builds a data file whose only item is "x" with the given body lines.
func oneItem(levels, extra, body string) string {
	return levels + "categories:\n  - {id: a, name: Alpha}\n" + extra +
		"items:\n  - {id: x, category: a, level: base, name: X, status: estimate" + body + "}\n"
}

const lvBase = "levels:\n  - {id: base, name: Base, sqft: 1000}\n"

func TestLoadInputsFixture(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "costs.yaml", inputsFixture)
	d, err := LoadIn(dir, "costs.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Inputs) != 3 || d.Inputs[0].ID != "rent-year" {
		t.Errorf("inputs = %+v", d.Inputs)
	}
	seating := d.Items[4]
	if len(seating.Lines) != 2 || seating.Lines[0].Label != "tables" || seating.Lines[0].Divisor != 1 {
		t.Errorf("seating lines = %+v", seating.Lines)
	}
	if d.Monthly[0].Lines[0].Divisor != 12 {
		t.Errorf("rent divisor = %d", d.Monthly[0].Lines[0].Divisor)
	}
	if d.Items[5].Lines != nil || d.Items[5].Expected != 20000 {
		t.Errorf("typed item = %+v", d.Items[5])
	}
}

func TestLoadInputsErrors(t *testing.T) {
	line := ", lines: [{quantity: 1, unit_price: 1}]"
	cases := []struct{ name, body, want string }{
		{"duplicate id", oneItem(lvBase, "inputs:\n  - {id: a, name: A, value: 1}\n  - {id: a, name: B, value: 2}\n", line), `input "a": duplicate input id`},
		{"non-slug id", oneItem(lvBase, "inputs:\n  - {id: Bad_Id, name: A, value: 1}\n", line), `input "Bad_Id": id must be a slug`},
		{"no name", oneItem(lvBase, "inputs:\n  - {id: a, value: 1}\n", line), `input "a": name is required`},
		{"no value", oneItem(lvBase, "inputs:\n  - {id: a, name: A}\n", line), `input "a": value is required`},
		{"unknown input ref", oneItem(lvBase, "inputs:\n  - {id: a, name: A, value: 1}\n", ", lines: [{quantity: input.b, unit_price: 1}]"), `item "x": lines[0].quantity: unknown input "b"`},
		{"unknown ref in input", oneItem(lvBase, "inputs:\n  - {id: a, name: A, value: input.zz}\n", line), `input "a": value: unknown input "zz"`},
		{"cycle among inputs", oneItem(lvBase, "inputs:\n  - {id: a, name: A, value: input.b}\n  - {id: b, name: B, value: input.a}\n", line), `input "a" -> input "b" -> input "a"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { wantLoadErr(t, c.body, c.want) })
	}
}

func TestLoadLinesErrors(t *testing.T) {
	cases := []struct{ name, body, want string }{
		{"typed amounts", oneItem(lvBase, "", ", low: 1, expected: 2, high: 3, lines: [{quantity: 1, unit_price: 1}]"), `item "x": lines and typed amounts cannot be combined`},
		{"one typed amount", oneItem(lvBase, "", ", expected: 2, lines: [{quantity: 1, unit_price: 1}]"), `item "x": lines and typed amounts cannot be combined`},
		{"empty lines", oneItem(lvBase, "", ", lines: []"), `item "x": lines must not be empty`},
		{"unparsable", oneItem(lvBase, "", `, lines: [{quantity: "1 +", unit_price: 1}]`), "costs.yaml: "},
		{"unknown item", oneItem(lvBase, "", ", lines: [{quantity: 1, unit_price: item.rentt}]"), `item "x": lines[0].unit_price: unknown item "rentt"`},
		{"unknown level", oneItem(lvBase, "", ", lines: [{quantity: level.nope.sqft, unit_price: 1}]"), `item "x": lines[0].quantity: unknown level "nope"`},
		{"level without sqft", oneItem("levels:\n  - {id: base, name: Base}\n", "", ", lines: [{quantity: level.base.sqft, unit_price: 1}]"), `item "x": lines[0].quantity: level "base" has no sqft`},
		{"added_sqft previous level without sqft", oneItem("levels:\n  - {id: base, name: Base}\n  - {id: plus, name: Plus, sqft: 1500}\n", "", ", lines: [{quantity: level.plus.added_sqft, unit_price: 1}]"), `item "x": lines[0].quantity: level "base" has no sqft`},
		{"divisor zero", oneItem(lvBase, "", ", lines: [{quantity: 1, unit_price: 1, divisor: 0}]"), `item "x": lines[0].divisor: must be at least 1`},
		{"missing quantity", oneItem(lvBase, "", ", lines: [{unit_price: 1}]"), `item "x": lines[0].quantity: is required`},
		{"unknown line field", oneItem(lvBase, "", ", lines: [{quantity: 1, unit_price: 1, price: 2}]"), "field price not found"},
		{"two-item cycle", lvBase + "categories:\n  - {id: a, name: Alpha}\nitems:\n" +
			"  - {id: a, category: a, level: base, name: A, status: estimate, lines: [{quantity: 1, unit_price: item.b}]}\n" +
			"  - {id: b, category: a, level: base, name: B, status: estimate, lines: [{quantity: 1, unit_price: item.a}]}\n",
			`item "a" -> item "b" -> item "a"`},
		{"self-reference", oneItem(lvBase, "", ", lines: [{quantity: 1, unit_price: item.x}]"), `item "x" -> item "x"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { wantLoadErr(t, c.body, c.want) })
	}
}
