package costs

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

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
	if err := d.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", reportPath, err)
	}
	return &d, nil
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
	for i, l := range d.Levels {
		who := label("level", "levels", i, l.ID)
		if !slugPattern.MatchString(l.ID) {
			return fmt.Errorf("%s: id must be a slug (lowercase letters, digits, hyphens)", who)
		}
		if levels[l.ID] {
			return fmt.Errorf("%s: duplicate level id", who)
		}
		levels[l.ID] = true
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

	ids := map[string]bool{}
	check := func(list string, items []Item, monthly bool) error {
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
			if it.Low > it.Expected {
				return fmt.Errorf("%s: low must not exceed expected", who)
			}
			if it.Expected > it.High {
				return fmt.Errorf("%s: expected must not exceed high", who)
			}
		}
		return nil
	}
	if err := check("items", d.Items, false); err != nil {
		return err
	}
	return check("monthly", d.Monthly, true)
}
