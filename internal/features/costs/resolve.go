package costs

import (
	"fmt"
	"math/big"
)

// ResolvedLine is a line with its quantity and unit price evaluated for
// low, expected, and high, and the rounded amount that results.
type ResolvedLine struct {
	Label     string
	Quantity  [memberCount]*big.Rat
	UnitPrice [memberCount]*big.Rat
	Divisor   int
	Amount    Range
}

type resolver struct {
	d       *Data
	inputs  map[string]*Input
	items   map[string]*Item
	levels  map[string]int
	inVals  map[string][memberCount]*big.Rat
	itemAmt map[string][memberCount]*big.Rat
}

// resolve derives the amounts of every item that has lines. It runs after
// validation, so references are known to exist and to be free of cycles.
func (d *Data) resolve() error {
	r := &resolver{
		d:       d,
		inputs:  map[string]*Input{},
		items:   map[string]*Item{},
		levels:  map[string]int{},
		inVals:  map[string][memberCount]*big.Rat{},
		itemAmt: map[string][memberCount]*big.Rat{},
	}
	for i := range d.Inputs {
		r.inputs[d.Inputs[i].ID] = &d.Inputs[i]
	}
	for _, list := range [][]Item{d.Items, d.Monthly} {
		for i := range list {
			r.items[list[i].ID] = &list[i]
		}
	}
	for i, l := range d.Levels {
		r.levels[l.ID] = i
	}
	for i := range d.Inputs {
		if err := r.input(d.Inputs[i].ID); err != nil {
			return err
		}
	}
	for _, list := range [][]Item{d.Items, d.Monthly} {
		for i := range list {
			if err := r.item(list[i].ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *resolver) deps(v Value) error {
	for _, ref := range v.Refs() {
		var err error
		switch ref.Kind {
		case "input":
			err = r.input(ref.ID)
		case "item":
			err = r.item(ref.ID)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *resolver) input(id string) error {
	if _, ok := r.inVals[id]; ok {
		return nil
	}
	in := r.inputs[id]
	who := fmt.Sprintf("input %q", id)
	if err := r.deps(in.Value); err != nil {
		return err
	}
	var vals [memberCount]*big.Rat
	for m := Low; m <= High; m++ {
		v, err := in.Value.Eval(m, r)
		if err != nil {
			return fmt.Errorf("%s: value: %s: %w", who, memberKeys[m], err)
		}
		vals[m] = v
	}
	r.inVals[id] = vals
	return nil
}

func (r *resolver) item(id string) error {
	if _, ok := r.itemAmt[id]; ok {
		return nil
	}
	it := r.items[id]
	if len(it.Lines) == 0 {
		var amt [memberCount]*big.Rat
		for m, c := range [memberCount]Money{it.Low, it.Expected, it.High} {
			amt[m] = centsToRat(c)
		}
		r.itemAmt[id] = amt
		return nil
	}
	who := fmt.Sprintf("item %q", id)
	var sum Range
	resolved := make([]ResolvedLine, 0, len(it.Lines))
	for j, ln := range it.Lines {
		for _, v := range []Value{ln.Quantity, ln.UnitPrice} {
			if err := r.deps(v); err != nil {
				return err
			}
		}
		rl := ResolvedLine{Label: ln.Label, Divisor: ln.Divisor}
		var amounts [memberCount]Money
		for m := Low; m <= High; m++ {
			q, err := ln.Quantity.Eval(m, r)
			if err != nil {
				return fmt.Errorf("%s: lines[%d].quantity: %s: %w", who, j, memberKeys[m], err)
			}
			p, err := ln.UnitPrice.Eval(m, r)
			if err != nil {
				return fmt.Errorf("%s: lines[%d].unit_price: %s: %w", who, j, memberKeys[m], err)
			}
			total := new(big.Rat).Mul(q, p)
			total.Quo(total, new(big.Rat).SetInt64(int64(ln.Divisor)))
			amounts[m], err = RoundCents(total)
			if err != nil {
				return fmt.Errorf("%s: lines[%d]: %s: %w", who, j, memberKeys[m], err)
			}
			rl.Quantity[m], rl.UnitPrice[m] = q, p
		}
		rl.Amount = Range{amounts[Low], amounts[Expected], amounts[High]}
		var err error
		if sum, err = sum.Add(rl.Amount); err != nil {
			return fmt.Errorf("%s: lines[%d]: %w", who, j, err)
		}
		resolved = append(resolved, rl)
	}
	if sum.Low > sum.Expected || sum.Expected > sum.High {
		return fmt.Errorf("%s: derived amounts must satisfy low <= expected <= high", who)
	}
	it.Low, it.Expected, it.High = sum.Low, sum.Expected, sum.High
	it.Resolved = resolved
	r.itemAmt[id] = [memberCount]*big.Rat{centsToRat(sum.Low), centsToRat(sum.Expected), centsToRat(sum.High)}
	return nil
}

func centsToRat(c Money) *big.Rat { return big.NewRat(int64(c), 100) }

// Lookup serves references whose targets were resolved beforehand.
func (r *resolver) Lookup(kind, id, field string, m Member) (*big.Rat, error) {
	switch kind {
	case "input":
		if v, ok := r.inVals[id]; ok {
			return v[m], nil
		}
	case "item":
		if v, ok := r.itemAmt[id]; ok {
			return v[m], nil
		}
	case "level":
		i, ok := r.levels[id]
		if !ok {
			break
		}
		sqft := r.d.Levels[i].Sqft
		if sqft <= 0 {
			return nil, fmt.Errorf("level %q has no sqft", id)
		}
		if field == "sqft" {
			return big.NewRat(int64(sqft), 1), nil
		}
		prev := 0
		if i > 0 {
			if prev = r.d.Levels[i-1].Sqft; prev <= 0 {
				return nil, fmt.Errorf("level %q has no sqft", r.d.Levels[i-1].ID)
			}
		}
		if sqft < prev {
			return nil, fmt.Errorf("level %q added_sqft is negative", id)
		}
		return big.NewRat(int64(sqft-prev), 1), nil
	}
	return nil, fmt.Errorf("cannot resolve %s", Ref{Kind: kind, ID: id, Field: field})
}
