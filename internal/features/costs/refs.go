package costs

import "fmt"

// validateRefs checks that every reference names something that exists and
// that level sizes are available where they are used.
func (d *Data) validateRefs(inputIDs, itemIDs map[string]bool, levelIdx map[string]int) error {
	check := func(who, where string, v Value) error {
		for _, r := range v.Refs() {
			switch r.Kind {
			case "input":
				if !inputIDs[r.ID] {
					return fmt.Errorf("%s: %s: unknown input %q", who, where, r.ID)
				}
			case "item":
				if !itemIDs[r.ID] {
					return fmt.Errorf("%s: %s: unknown item %q", who, where, r.ID)
				}
			case "level":
				i, ok := levelIdx[r.ID]
				if !ok {
					return fmt.Errorf("%s: %s: unknown level %q", who, where, r.ID)
				}
				if d.Levels[i].Sqft <= 0 {
					return fmt.Errorf("%s: %s: level %q has no sqft for %s", who, where, r.ID, r)
				}
				if r.Field == "added_sqft" && i > 0 && d.Levels[i-1].Sqft <= 0 {
					return fmt.Errorf("%s: %s: level %q has no sqft for %s", who, where, d.Levels[i-1].ID, r)
				}
			}
		}
		return nil
	}
	for i, in := range d.Inputs {
		if err := check(label("input", "inputs", i, in.ID), "value", in.Value); err != nil {
			return err
		}
	}
	for _, list := range []struct {
		name  string
		items []Item
	}{{"items", d.Items}, {"monthly", d.Monthly}} {
		for i, it := range list.items {
			who := label("item", list.name, i, it.ID)
			for j, ln := range it.Lines {
				if err := check(who, fmt.Sprintf("lines[%d].quantity", j), ln.Quantity); err != nil {
					return err
				}
				if err := check(who, fmt.Sprintf("lines[%d].unit_price", j), ln.UnitPrice); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// validateCycles rejects references that lead back to where they started,
// among inputs and items.
func (d *Data) validateCycles() error {
	type node struct {
		name string
		deps []string
	}
	nodes := map[string]*node{}
	var order []string
	add := func(kind, id string, vals ...Value) {
		n := &node{name: fmt.Sprintf("%s %q", kind, id)}
		for _, v := range vals {
			for _, r := range v.Refs() {
				if r.Kind == "input" || r.Kind == "item" {
					n.deps = append(n.deps, r.Kind+"."+r.ID)
				}
			}
		}
		key := kind + "." + id
		nodes[key] = n
		order = append(order, key)
	}
	for _, in := range d.Inputs {
		add("input", in.ID, in.Value)
	}
	for _, list := range [][]Item{d.Items, d.Monthly} {
		for _, it := range list {
			var vals []Value
			for _, ln := range it.Lines {
				vals = append(vals, ln.Quantity, ln.UnitPrice)
			}
			add("item", it.ID, vals...)
		}
	}

	const (
		visiting = 1
		done     = 2
	)
	state := map[string]int{}
	var stack []string
	var visit func(key string) error
	visit = func(key string) error {
		state[key] = visiting
		stack = append(stack, key)
		for _, dep := range nodes[key].deps {
			switch state[dep] {
			case visiting:
				start := 0
				for stack[start] != dep {
					start++
				}
				path := ""
				for _, k := range stack[start:] {
					path += nodes[k].name + " -> "
				}
				path += nodes[dep].name
				return fmt.Errorf("%s: reference cycle %s", nodes[dep].name, path)
			case 0:
				if err := visit(dep); err != nil {
					return err
				}
			}
		}
		stack = stack[:len(stack)-1]
		state[key] = done
		return nil
	}
	for _, key := range order {
		if state[key] == 0 {
			if err := visit(key); err != nil {
				return err
			}
		}
	}
	return nil
}
