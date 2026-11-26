package costs

import (
	"math"
	"testing"
)

func mustMoney(t *testing.T, s string) Money {
	t.Helper()
	m, err := ParseMoney(s)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func rng(t *testing.T, low, exp, high string) Range {
	t.Helper()
	return Range{mustMoney(t, low), mustMoney(t, exp), mustMoney(t, high)}
}

func loadExample(t *testing.T) *Data {
	t.Helper()
	d, err := Load(writeFile(t, t.TempDir(), "costs.yaml", exampleYAML))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func checkRange(t *testing.T, name string, got, want Range) {
	t.Helper()
	if got.Low != want.Low {
		t.Errorf("%s low = %v, want %v", name, got.Low, want.Low)
	}
	if got.Expected != want.Expected {
		t.Errorf("%s expected = %v, want %v", name, got.Expected, want.Expected)
	}
	if got.High != want.High {
		t.Errorf("%s high = %v, want %v", name, got.High, want.High)
	}
}

func TestTotals(t *testing.T) {
	tot, err := loadExample(t).Totals()
	if err != nil {
		t.Fatal(err)
	}
	if len(tot.Categories) != 2 || len(tot.Levels) != 2 {
		t.Fatalf("got %d categories and %d levels, want 2 and 2", len(tot.Categories), len(tot.Levels))
	}
	checkRange(t, "Alpha", tot.Categories[0].Range, rng(t, "150", "260", "370"))
	checkRange(t, "Beta", tot.Categories[1].Range, rng(t, "1000.50", "1500", "2000"))

	base, plus := tot.Levels[0], tot.Levels[1]
	checkRange(t, "Base Items", base.Items, rng(t, "1100.50", "1700", "2300"))
	checkRange(t, "Base Monthly", base.Monthly, rng(t, "10", "20", "30"))
	if want := (ReserveMonths{1, 1, 2}); base.ReserveMonths != want {
		t.Errorf("Base ReserveMonths = %+v, want %+v", base.ReserveMonths, want)
	}
	checkRange(t, "Base Reserve", base.Reserve, rng(t, "10", "20", "60"))
	checkRange(t, "Base Goal", base.Goal, rng(t, "1110.50", "1720", "2360"))
	checkRange(t, "Base Adds", base.Adds, rng(t, "1110.50", "1720", "2360"))

	checkRange(t, "Plus Items", plus.Items, rng(t, "50", "60", "70"))
	checkRange(t, "Plus Monthly", plus.Monthly, rng(t, "15", "25", "35"))
	if want := (ReserveMonths{1, 2, 3}); plus.ReserveMonths != want {
		t.Errorf("Plus ReserveMonths = %+v, want %+v", plus.ReserveMonths, want)
	}
	checkRange(t, "Plus Reserve", plus.Reserve, rng(t, "15", "50", "105"))
	checkRange(t, "Plus Goal", plus.Goal, rng(t, "1165.50", "1810", "2475"))
	checkRange(t, "Plus Adds", plus.Adds, rng(t, "55", "90", "115"))

	checkRange(t, "Startup", tot.Startup, rng(t, "1150.50", "1760", "2370"))
	checkRange(t, "Monthly", tot.Monthly, rng(t, "15", "25", "35"))
	checkRange(t, "Reserve", tot.Reserve, rng(t, "15", "50", "105"))
	checkRange(t, "Total", tot.Total, rng(t, "1165.50", "1810", "2475"))
}

func TestTotalsAddsEqualItemsPlusAddedReserve(t *testing.T) {
	tot, err := loadExample(t).Totals()
	if err != nil {
		t.Fatal(err)
	}
	var prevReserve Range
	for _, l := range tot.Levels {
		added, err := l.Reserve.Sub(prevReserve)
		if err != nil {
			t.Fatal(err)
		}
		want, err := l.Items.Add(added)
		if err != nil {
			t.Fatal(err)
		}
		checkRange(t, l.ID+" Adds", l.Adds, want)
		prevReserve = l.Reserve
	}
}

func TestTotalsEmptyCategory(t *testing.T) {
	d := loadExample(t)
	d.Categories = []Category{{ID: "z", Name: "Zeta"}, d.Categories[0], {ID: "y", Name: "Yankee"}, d.Categories[1]}
	tot, err := d.Totals()
	if err != nil {
		t.Fatal(err)
	}
	wantIDs := []string{"z", "a", "y", "b"}
	if len(tot.Categories) != len(wantIDs) {
		t.Fatalf("got %d categories, want %d", len(tot.Categories), len(wantIDs))
	}
	for i, id := range wantIDs {
		if tot.Categories[i].ID != id {
			t.Errorf("category %d = %q, want %q", i, tot.Categories[i].ID, id)
		}
	}
	checkRange(t, "Zeta", tot.Categories[0].Range, Range{})
	checkRange(t, "Yankee", tot.Categories[2].Range, Range{})
	checkRange(t, "Alpha", tot.Categories[1].Range, rng(t, "150", "260", "370"))
}

func TestTotalsNoMonthly(t *testing.T) {
	d := loadExample(t)
	d.Monthly = nil
	tot, err := d.Totals()
	if err != nil {
		t.Fatal(err)
	}
	checkRange(t, "Monthly", tot.Monthly, Range{})
	checkRange(t, "Reserve", tot.Reserve, Range{})
	checkRange(t, "Base Goal", tot.Levels[0].Goal, tot.Levels[0].Items)
	checkRange(t, "Plus Goal", tot.Levels[1].Goal, rng(t, "1150.50", "1760", "2370"))
	checkRange(t, "Total", tot.Total, tot.Startup)
}

func TestTotalsOverflow(t *testing.T) {
	big := Money(math.MaxInt64 - 5)
	tests := map[string]func(d *Data){
		"items": func(d *Data) {
			d.Items = []Item{
				{ID: "x", Category: "a", Level: "base", Low: big, Expected: 1, High: 1},
				{ID: "y", Category: "a", Level: "base", Low: big, Expected: 1, High: 1},
			}
		},
		"monthly-reserve": func(d *Data) {
			d.Monthly = []Item{{ID: "m", Level: "base", Low: big, Expected: 1, High: 1}}
		},
		"goal": func(d *Data) {
			d.Items = []Item{{ID: "x", Category: "a", Level: "base", Low: big, Expected: 1, High: 1}}
			d.Monthly = []Item{{ID: "m", Level: "base", Low: 10, Expected: 1, High: 1}}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			d := loadExample(t)
			mutate(d)
			if _, err := d.Totals(); err == nil {
				t.Fatal("expected an overflow error")
			}
		})
	}
}

func TestLookups(t *testing.T) {
	d := loadExample(t)

	if c, ok := d.Category("b"); !ok || c.Name != "Beta" {
		t.Errorf("Category(b) = %+v, %v", c, ok)
	}
	if _, ok := d.Category("nope"); ok {
		t.Error("Category(nope) found")
	}

	if it, ok := d.Item("a2"); !ok || it.Name != "A two" {
		t.Errorf("Item(a2) = %+v, %v", it, ok)
	}
	if it, ok := d.Item("m2"); !ok || it.Name != "M two" {
		t.Errorf("Item(m2) = %+v, %v", it, ok)
	}
	if _, ok := d.Item("nope"); ok {
		t.Error("Item(nope) found")
	}

	if l, i, ok := d.Level("plus"); !ok || i != 1 || l.Name != "Plus" {
		t.Errorf("Level(plus) = %+v, %d, %v", l, i, ok)
	}
	if _, i, ok := d.Level("nope"); ok || i != -1 {
		t.Errorf("Level(nope) = %d, %v", i, ok)
	}
}
