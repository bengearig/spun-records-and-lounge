package costs

import (
	"strings"
	"testing"
)

func contains(s, sub string) bool { return strings.Contains(s, sub) }

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "costs.yaml", exampleYAML)
	d, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Levels) != 2 || len(d.Categories) != 2 || len(d.Items) != 3 || len(d.Monthly) != 2 {
		t.Fatalf("counts: %d levels, %d categories, %d items, %d monthly",
			len(d.Levels), len(d.Categories), len(d.Items), len(d.Monthly))
	}
	if d.Levels[0].ID != "base" || d.Levels[0].ReserveMonths != (ReserveMonths{1, 1, 2}) {
		t.Errorf("base level = %+v", d.Levels[0])
	}
	if d.Levels[1].ID != "plus" || d.Levels[1].ReserveMonths != (ReserveMonths{0, 1, 1}) {
		t.Errorf("plus level = %+v", d.Levels[1])
	}
	if d.Categories[0] != (Category{ID: "a", Name: "Alpha"}) || d.Categories[1] != (Category{ID: "b", Name: "Beta"}) {
		t.Errorf("categories = %+v", d.Categories)
	}
	wantItems := []Item{
		{ID: "a1", Name: "A one", Category: "a", Level: "base", Low: 10000, Expected: 20000, High: 30000, Status: StatusEstimate},
		{ID: "a2", Name: "A two", Category: "a", Level: "plus", Low: 5000, Expected: 6000, High: 7000, Status: StatusEstimate, OwnerInput: true},
		{ID: "b1", Name: "B one", Category: "b", Level: "base", Low: 100050, Expected: 150000, High: 200000, Status: StatusEstimate},
	}
	for i, w := range wantItems {
		if d.Items[i] != w {
			t.Errorf("item %d = %+v; want %+v", i, d.Items[i], w)
		}
	}
	wantMonthly := []Item{
		{ID: "m1", Name: "M one", Level: "base", Low: 1000, Expected: 2000, High: 3000, Status: StatusEstimate},
		{ID: "m2", Name: "M two", Level: "plus", Low: 500, Expected: 500, High: 500, Status: StatusEstimate},
	}
	for i, w := range wantMonthly {
		if d.Monthly[i] != w {
			t.Errorf("monthly %d = %+v; want %+v", i, d.Monthly[i], w)
		}
	}

	if _, err := LoadIn(dir, "costs.yaml"); err != nil {
		t.Errorf("LoadIn: %v", err)
	}
	_, err = LoadIn(dir, "data/missing.yaml")
	if err == nil || !strings.HasPrefix(err.Error(), "data/missing.yaml: ") || strings.Contains(err.Error(), dir) {
		t.Errorf("LoadIn missing error = %v", err)
	}
	writeFile(t, dir, "data/bad.yaml", "levels: []\n")
	_, err = LoadIn(dir, "data/bad.yaml")
	if err == nil || !strings.HasPrefix(err.Error(), "data/bad.yaml: ") || strings.Contains(err.Error(), dir) {
		t.Errorf("LoadIn invalid error = %v", err)
	}
}

func TestLoadInvalid(t *testing.T) {
	edit := func(old, repl string) string {
		if !strings.Contains(exampleYAML, old) {
			t.Fatalf("exampleYAML lacks %q", old)
		}
		return strings.Replace(exampleYAML, old, repl, 1)
	}
	cases := []struct {
		name string
		body string
		want string
	}{
		{"unknown field", edit("name: Alpha}", "name: Alpha, color: red}"), "color"},
		{"no levels", "levels: []\n" + exampleYAML[strings.Index(exampleYAML, "categories:"):], "at least one level"},
		{"no categories", strings.Replace(exampleYAML, "categories:\n  - {id: a, name: Alpha}\n  - {id: b, name: Beta}\n", "categories: []\n", 1), "at least one category"},
		{"no items", exampleYAML[:strings.Index(exampleYAML, "items:")] + "items: []\n", "at least one item"},
		{"non-slug level id", edit("id: base\n", "id: Base_1\n"), `level "Base_1": id must be a slug`},
		{"duplicate level id", edit("id: plus\n", "id: base\n"), `level "base": duplicate level id`},
		{"non-positive sqft", edit("id: base\n    name: Base\n", "id: base\n    name: Base\n    sqft: -1500\n"), `level "base": sqft must be positive`},
		{"reserve months out of order", edit("{low: 1, expected: 1, high: 2}", "{low: 3, expected: 1, high: 2}"), `level "base": reserve_months must satisfy`},
		{"negative reserve months", edit("{low: 1, expected: 1, high: 2}", "{low: -1, expected: 1, high: 2}"), `level "base": reserve_months must not be negative`},
		{"non-slug category id", edit("{id: a, name: Alpha}", "{id: A!, name: Alpha}"), `category "A!": id must be a slug`},
		{"duplicate category id", edit("{id: b, name: Beta}", "{id: a, name: Beta}"), `category "a": duplicate category id`},
		{"duplicate item id across lists", edit("{id: m1,", "{id: a1,"), `item "a1": duplicate item id`},
		{"non-slug item id", edit("{id: a1,", "{id: A_1,"), `item "A_1": id must be a slug`},
		{"unknown item category", edit("{id: a1, category: a,", "{id: a1, category: zzz,"), `item "a1": unknown category "zzz"`},
		{"missing item category", edit("{id: a1, category: a,", "{id: a1,"), `item "a1": category is required`},
		{"monthly item with category", edit("{id: m1, level: base,", "{id: m1, category: a, level: base,"), `item "m1": monthly items must not have a category`},
		{"missing item level", edit("{id: a1, category: a, level: base,", "{id: a1, category: a,"), `item "a1": level is required`},
		{"unknown item level", edit("{id: a1, category: a, level: base,", "{id: a1, category: a, level: nope,"), `item "a1": unknown level "nope"`},
		{"missing monthly level", edit("{id: m1, level: base,", "{id: m1,"), `item "m1": level is required`},
		{"unknown monthly level", edit("{id: m1, level: base,", "{id: m1, level: nope,"), `item "m1": unknown level "nope"`},
		{"missing name", edit("name: A one, ", ""), `item "a1": name is required`},
		{"bad status", edit("status: estimate, owner_input", "status: maybe, owner_input"), `item "a2": invalid status "maybe"`},
		{"missing status", edit("high: 300, status: estimate", "high: 300"), `item "a1": invalid status`},
		{"low above expected", edit("low: 100, expected: 200", "low: 250, expected: 200"), `item "a1": low must not exceed expected`},
		{"expected above high", edit("low: 100, expected: 200, high: 300", "low: 100, expected: 400, high: 300"), `item "a1": expected must not exceed high`},
		{"bad money value", edit("low: 100, expected: 200", `low: "1,000", expected: 200`), `"1,000"`},
		{"negative money value", edit("high: 300,", "high: -300,"), `"-300"`},
		{"empty file", "", "file is empty"},
	}
	dir := t.TempDir()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeFile(t, dir, "costs.yaml", tc.body)
			_, err := Load(path)
			if err == nil {
				t.Fatal("Load succeeded; want error")
			}
			if !strings.HasPrefix(err.Error(), path+": ") {
				t.Errorf("error %q does not start with the path", err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q lacks %q", err, tc.want)
			}
		})
	}

	t.Run("missing file", func(t *testing.T) {
		path := dir + "/absent.yaml"
		if _, err := Load(path); err == nil || !strings.HasPrefix(err.Error(), path+": ") {
			t.Errorf("error = %v", err)
		}
	})
	t.Run("missing id names the index", func(t *testing.T) {
		path := writeFile(t, dir, "noid.yaml", edit("{id: a1,", "{"))
		if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "items[0]") {
			t.Errorf("error = %v", err)
		}
	})
	t.Run("monthly absent", func(t *testing.T) {
		body := exampleYAML[:strings.Index(exampleYAML, "monthly:")]
		if _, err := Load(writeFile(t, dir, "nomonthly.yaml", body)); err != nil {
			t.Errorf("error = %v", err)
		}
	})
}
