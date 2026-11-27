package costs

import (
	"math/big"
	"testing"
)

func loadFixture(t *testing.T) *Data {
	t.Helper()
	d, err := Load(writeFile(t, t.TempDir(), "costs.yaml", inputsFixture))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestResolveFixture(t *testing.T) {
	d := loadFixture(t)
	want := map[string]Range{
		"rent":       rng(t, "1458.33", "1750", "2500"),
		"rent-plus":  rng(t, "729.17", "875", "1250"),
		"deposit":    rng(t, "0", "1750", "7500"),
		"build-rent": rng(t, "0", "5250", "22500"),
		"counter":    rng(t, "1200", "1800", "6000"),
		"mural":      rng(t, "6400", "8000", "12800"),
		"seating":    rng(t, "1650", "7350", "12300"),
		"typed":      rng(t, "100", "200", "300"),
	}
	got := map[string]Item{}
	for _, list := range [][]Item{d.Items, d.Monthly} {
		for _, it := range list {
			got[it.ID] = it
		}
	}
	for id, w := range want {
		checkRange(t, id, itemRange(got[id]), w)
	}

	if got["typed"].Resolved != nil {
		t.Errorf("typed item has resolved lines: %+v", got["typed"].Resolved)
	}
	if n := len(got["seating"].Resolved); n != 2 {
		t.Fatalf("seating resolved lines = %d, want 2", n)
	}
	checkRange(t, "seats line", got["seating"].Resolved[1].Amount, rng(t, "1200", "6000", "9600"))
	if got["seating"].Resolved[1].Label != "seats" {
		t.Errorf("label = %q", got["seating"].Resolved[1].Label)
	}

	rent := got["rent"].Resolved
	if len(rent) != 1 || rent[0].Divisor != 12 {
		t.Fatalf("rent resolved = %+v", rent)
	}
	prices := [3]*big.Rat{big.NewRat(35, 2), big.NewRat(21, 1), big.NewRat(30, 1)}
	for m := 0; m < 3; m++ {
		if rent[0].Quantity[m].Cmp(big.NewRat(1000, 1)) != 0 {
			t.Errorf("quantity[%d] = %v, want 1000", m, rent[0].Quantity[m])
		}
		if rent[0].UnitPrice[m].Cmp(prices[m]) != 0 {
			t.Errorf("unit price[%d] = %v, want %v", m, rent[0].UnitPrice[m], prices[m])
		}
	}
	checkRange(t, "rent line", rent[0].Amount, want["rent"])
}

func TestResolveErrors(t *testing.T) {
	cases := []struct{ name, body, want string }{
		{"out of order", oneItem(lvBase, "", ", lines: [{quantity: {low: 5, expected: 1, high: 3}, unit_price: 1}]"),
			`item "x": derived amounts must satisfy low <= expected <= high`},
		{"negative subtraction", oneItem(lvBase, "inputs:\n  - {id: a, name: A, value: 1}\n  - {id: b, name: B, value: 2}\n",
			`, lines: [{quantity: "input.a - input.b", unit_price: 1}]`), `item "x": lines[0].quantity: low: subtraction`},
		{"overflow", oneItem(lvBase, "", ", lines: [{quantity: 999999999999999, unit_price: 999999999999999}]"),
			`item "x": lines[0]: low: amount overflows`},
		{"sum overflow", oneItem(lvBase, "", ", lines: [{quantity: 999999999999999, unit_price: 50}, {quantity: 999999999999999, unit_price: 50}]"),
			`item "x": lines[1]: amount overflows`},
		{"division by zero", oneItem(lvBase, "inputs:\n  - {id: z, name: Z, value: 0}\n",
			`, lines: [{quantity: "1 / input.z", unit_price: 1}]`), `item "x": lines[0].quantity: low: division by zero`},
		{"negative added_sqft", "levels:\n  - {id: base, name: Base, sqft: 1000}\n  - {id: plus, name: Plus, sqft: 500}\n" +
			"categories:\n  - {id: a, name: Alpha}\nitems:\n  - {id: x, category: a, level: base, name: X, status: estimate, lines: [{quantity: level.plus.added_sqft, unit_price: 1}]}\n",
			`item "x": lines[0].quantity: low: level "plus" added_sqft is negative`},
		{"input evaluation", oneItem(lvBase, "inputs:\n  - {id: a, name: A, value: \"1 / 0\"}\n", ", lines: [{quantity: input.a, unit_price: 1}]"),
			`input "a": value: low: division by zero`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { wantLoadErr(t, c.body, c.want) })
	}
}

func TestResolveTotals(t *testing.T) {
	tot, err := loadFixture(t).Totals()
	if err != nil {
		t.Fatal(err)
	}
	checkRange(t, "Alpha", tot.Categories[0].Range, rng(t, "9350", "24350", "61400"))
	checkRange(t, "Base items", tot.Levels[0].Items, rng(t, "2950", "16350", "48600"))
	checkRange(t, "Plus items", tot.Levels[1].Items, rng(t, "6400", "8000", "12800"))
	checkRange(t, "Base monthly", tot.Levels[0].Monthly, rng(t, "1458.33", "1750", "2500"))
	checkRange(t, "Plus monthly", tot.Levels[1].Monthly, rng(t, "2187.50", "2625", "3750"))
	checkRange(t, "Base reserve", tot.Levels[0].Reserve, rng(t, "1458.33", "1750", "2500"))
	checkRange(t, "Plus reserve", tot.Levels[1].Reserve, rng(t, "4375", "5250", "7500"))
	checkRange(t, "Base goal", tot.Levels[0].Goal, rng(t, "4408.33", "18100", "51100"))
	checkRange(t, "Plus goal", tot.Levels[1].Goal, rng(t, "13725", "29600", "68900"))
}
