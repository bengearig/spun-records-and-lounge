package costs

import (
	"strings"
	"testing"
)

const m3Line = "  - {id: m3, level: base, name: M three, low: 100, expected: 100, high: 100, status: estimate, reserve_months: {base: {low: 1, expected: 1, high: 1}, plus: {low: 1, expected: 2, high: 2}}}\n"

func loadWithM3(t *testing.T) *Data {
	t.Helper()
	d, err := Load(writeFile(t, t.TempDir(), "costs.yaml", exampleYAML+m3Line))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestTotalsItemReserveMonths(t *testing.T) {
	tot, err := loadWithM3(t).Totals()
	if err != nil {
		t.Fatal(err)
	}
	base, plus := tot.Levels[0], tot.Levels[1]
	checkRange(t, "Base Monthly", base.Monthly, rng(t, "110", "120", "130"))
	checkRange(t, "Base Reserve", base.Reserve, rng(t, "110", "120", "160"))
	checkRange(t, "Base Goal", base.Goal, rng(t, "1210.50", "1820", "2460"))
	checkRange(t, "Plus Monthly", plus.Monthly, rng(t, "115", "125", "135"))
	checkRange(t, "Plus Reserve", plus.Reserve, rng(t, "215", "350", "405"))
	checkRange(t, "Plus Goal", plus.Goal, rng(t, "1365.50", "2110", "2775"))
	checkRange(t, "Plus Adds", plus.Adds, rng(t, "155", "290", "315"))
	checkRange(t, "Reserve", tot.Reserve, plus.Reserve)
	checkRange(t, "Total", tot.Total, plus.Goal)
}

func TestItemReserveMonths(t *testing.T) {
	d := loadWithM3(t)
	for id, want := range map[string]ReserveMonths{
		"m3": {2, 3, 3},
		"m1": {1, 2, 3},
		"m2": {1, 2, 3},
	} {
		it, ok := d.Item(id)
		if !ok {
			t.Fatalf("item %q missing", id)
		}
		if got := d.ItemReserveMonths(it); got != want {
			t.Errorf("%s months = %+v, want %+v", id, got, want)
		}
	}
}

func TestLoadItemReserveMonthsErrors(t *testing.T) {
	good := "reserve_months: {base: {low: 1, expected: 1, high: 1}}"
	cases := []struct {
		name, body, want string
	}{
		{"startup item", strings.Replace(exampleYAML, "name: A one,", "name: A one, "+good+",", 1), `item "a1": reserve_months is only for monthly items`},
		{"empty map", exampleYAML + "  - {id: m3, level: base, name: M, low: 1, expected: 1, high: 1, status: estimate, reserve_months: {}}\n", `item "m3": reserve_months must not be empty`},
		{"unknown level", exampleYAML + "  - {id: m3, level: base, name: M, low: 1, expected: 1, high: 1, status: estimate, reserve_months: {nope: {low: 1, expected: 1, high: 1}}}\n", `item "m3": reserve_months has unknown level "nope"`},
		{"level before item", exampleYAML + "  - {id: m3, level: plus, name: M, low: 1, expected: 1, high: 1, status: estimate, reserve_months: {base: {low: 1, expected: 1, high: 1}}}\n", `item "m3": reserve_months level "base" comes before`},
		{"negative months", exampleYAML + "  - {id: m3, level: base, name: M, low: 1, expected: 1, high: 1, status: estimate, reserve_months: {base: {low: -1, expected: 1, high: 1}}}\n", `item "m3": reserve_months level "base" must not be negative`},
		{"unordered months", exampleYAML + "  - {id: m3, level: base, name: M, low: 1, expected: 1, high: 1, status: estimate, reserve_months: {base: {low: 2, expected: 1, high: 1}}}\n", `item "m3": reserve_months level "base" must satisfy`},
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
}
