package costs

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"gopkg.in/yaml.v3"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Load reads, strictly decodes, and validates the data file at path.
func Load(path string) (*Data, error) {
	return load(path, path)
}

// LoadIn reads dir/name but reports name in errors.
func LoadIn(dir, name string) (*Data, error) {
	return load(filepath.Join(dir, name), name)
}

func load(readPath, reportPath string) (*Data, error) {
	raw, err := os.ReadFile(readPath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", reportPath, errors.Unwrap(err))
	}
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	var d Data
	if err := dec.Decode(&d); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("%s: file is empty", reportPath)
		}
		return nil, fmt.Errorf("%s: %w", reportPath, err)
	}
	d.written = readWritten(raw)
	if err := d.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", reportPath, err)
	}
	return &d, nil
}

// readWritten reports which items spelled out any of low, expected, or high.
func readWritten(raw []byte) [2][]bool {
	type amounts struct {
		Low      yaml.Node `yaml:"low"`
		Expected yaml.Node `yaml:"expected"`
		High     yaml.Node `yaml:"high"`
	}
	var f struct {
		Items   []amounts `yaml:"items"`
		Monthly []amounts `yaml:"monthly"`
	}
	_ = yaml.Unmarshal(raw, &f)
	var w [2][]bool
	for i, list := range [][]amounts{f.Items, f.Monthly} {
		for _, a := range list {
			w[i] = append(w[i], a.Low.Kind != 0 || a.Expected.Kind != 0 || a.High.Kind != 0)
		}
	}
	return w
}

func label(kind, list string, i int, id string) string {
	if id == "" {
		return fmt.Sprintf("%s[%d]", list, i)
	}
	return fmt.Sprintf("%s %q", kind, id)
}

func (d *Data) validate() error {
	if len(d.Levels) == 0 {
		return errors.New("at least one level is required")
	}
	if len(d.Categories) == 0 {
		return errors.New("at least one category is required")
	}
	if len(d.Items) == 0 {
		return errors.New("at least one item is required")
	}

	levels := map[string]bool{}
	levelIdx := map[string]int{}
	for i, l := range d.Levels {
		who := label("level", "levels", i, l.ID)
		if !slugPattern.MatchString(l.ID) {
			return fmt.Errorf("%s: id must be a slug (lowercase letters, digits, hyphens)", who)
		}
		if levels[l.ID] {
			return fmt.Errorf("%s: duplicate level id", who)
		}
		levels[l.ID] = true
		levelIdx[l.ID] = i
		if l.Name == "" {
			return fmt.Errorf("%s: name is required", who)
		}
		if l.Sqft < 0 {
			return fmt.Errorf("%s: sqft must be positive", who)
		}
		r := l.ReserveMonths
		if r.Low < 0 || r.Expected < 0 || r.High < 0 {
			return fmt.Errorf("%s: reserve_months must not be negative", who)
		}
		if r.Low > r.Expected || r.Expected > r.High {
			return fmt.Errorf("%s: reserve_months must satisfy low <= expected <= high", who)
		}
	}

	cats := map[string]bool{}
	for i, c := range d.Categories {
		who := label("category", "categories", i, c.ID)
		if !slugPattern.MatchString(c.ID) {
			return fmt.Errorf("%s: id must be a slug (lowercase letters, digits, hyphens)", who)
		}
		if cats[c.ID] {
			return fmt.Errorf("%s: duplicate category id", who)
		}
		cats[c.ID] = true
		if c.Name == "" {
			return fmt.Errorf("%s: name is required", who)
		}
	}

	inputIDs := map[string]bool{}
	for i, in := range d.Inputs {
		who := label("input", "inputs", i, in.ID)
		if !slugPattern.MatchString(in.ID) {
			return fmt.Errorf("%s: id must be a slug (lowercase letters, digits, hyphens)", who)
		}
		if inputIDs[in.ID] {
			return fmt.Errorf("%s: duplicate input id", who)
		}
		inputIDs[in.ID] = true
		if in.Name == "" {
			return fmt.Errorf("%s: name is required", who)
		}
		if !in.Value.isSet() {
			return fmt.Errorf("%s: value is required", who)
		}
	}

	ids := map[string]bool{}
	check := func(list string, kind int, items []Item, monthly bool) error {
		for i, it := range items {
			who := label("item", list, i, it.ID)
			if !slugPattern.MatchString(it.ID) {
				return fmt.Errorf("%s: id must be a slug (lowercase letters, digits, hyphens)", who)
			}
			if ids[it.ID] {
				return fmt.Errorf("%s: duplicate item id", who)
			}
			ids[it.ID] = true
			if it.Name == "" {
				return fmt.Errorf("%s: name is required", who)
			}
			switch {
			case monthly && it.Category != "":
				return fmt.Errorf("%s: monthly items must not have a category", who)
			case !monthly && it.Category == "":
				return fmt.Errorf("%s: category is required", who)
			case !monthly && !cats[it.Category]:
				return fmt.Errorf("%s: unknown category %q", who, it.Category)
			}
			if it.Lines != nil && kind < len(d.written) && i < len(d.written[kind]) && d.written[kind][i] {
				return fmt.Errorf("%s: lines and typed amounts cannot be combined", who)
			}
			if it.Lines != nil && len(it.Lines) == 0 {
				return fmt.Errorf("%s: lines must not be empty", who)
			}
			for j, ln := range it.Lines {
				if !ln.Quantity.isSet() {
					return fmt.Errorf("%s: lines[%d].quantity: is required", who, j)
				}
				if !ln.UnitPrice.isSet() {
					return fmt.Errorf("%s: lines[%d].unit_price: is required", who, j)
				}
				if ln.Divisor < 1 {
					return fmt.Errorf("%s: lines[%d].divisor: must be at least 1", who, j)
				}
			}
			if it.Level == "" {
				return fmt.Errorf("%s: level is required", who)
			}
			if !levels[it.Level] {
				return fmt.Errorf("%s: unknown level %q", who, it.Level)
			}
			switch it.Status {
			case StatusEstimate, StatusQuote, StatusPaid:
			default:
				return fmt.Errorf("%s: invalid status %q (want estimate, quote, or paid)", who, it.Status)
			}
			if it.Lines == nil {
				if it.Low > it.Expected {
					return fmt.Errorf("%s: low must not exceed expected", who)
				}
				if it.Expected > it.High {
					return fmt.Errorf("%s: expected must not exceed high", who)
				}
			}
			if err := validateItemReserve(who, it, monthly, levelIdx); err != nil {
				return err
			}
		}
		return nil
	}
	if err := check("items", 0, d.Items, false); err != nil {
		return err
	}
	if err := check("monthly", 1, d.Monthly, true); err != nil {
		return err
	}
	if err := d.validateRefs(inputIDs, ids, levelIdx); err != nil {
		return err
	}
	return d.validateCycles()
}

func validateItemReserve(who string, it Item, monthly bool, levelIdx map[string]int) error {
	if it.ReserveMonths == nil {
		return nil
	}
	if !monthly {
		return fmt.Errorf("%s: reserve_months is only for monthly items", who)
	}
	if len(it.ReserveMonths) == 0 {
		return fmt.Errorf("%s: reserve_months must not be empty", who)
	}
	keys := make([]string, 0, len(it.ReserveMonths))
	for k := range it.ReserveMonths {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		i, ok := levelIdx[k]
		if !ok {
			return fmt.Errorf("%s: reserve_months has unknown level %q", who, k)
		}
		if i < levelIdx[it.Level] {
			return fmt.Errorf("%s: reserve_months level %q comes before the item's level %q", who, k, it.Level)
		}
		r := it.ReserveMonths[k]
		if r.Low < 0 || r.Expected < 0 || r.High < 0 {
			return fmt.Errorf("%s: reserve_months level %q must not be negative", who, k)
		}
		if r.Low > r.Expected || r.Expected > r.High {
			return fmt.Errorf("%s: reserve_months level %q must satisfy low <= expected <= high", who, k)
		}
	}
	return nil
}
