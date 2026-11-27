package costs

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Item statuses.
const (
	StatusEstimate = "estimate"
	StatusQuote    = "quote"
	StatusPaid     = "paid"
)

// Data is the decoded cost data file.
type Data struct {
	Levels     []Level    `yaml:"levels"`
	Categories []Category `yaml:"categories"`
	Items      []Item     `yaml:"items"`
	Monthly    []Item     `yaml:"monthly"`
	Inputs     []Input    `yaml:"inputs"`

	// written records, per list, which items spelled out low, expected, or high.
	written [2][]bool
}

// Input is a named value shared by the lines and inputs that refer to it.
type Input struct {
	ID     string `yaml:"id"`
	Name   string `yaml:"name"`
	Value  Value  `yaml:"value"`
	Source string `yaml:"source"`
	Notes  string `yaml:"notes"`
}

// Line is one quantity times unit price, divided by Divisor, of an item.
type Line struct {
	Label     string
	Quantity  Value
	UnitPrice Value
	Divisor   int
}

var lineKeys = map[string]bool{"label": true, "quantity": true, "unit_price": true, "divisor": true}

// UnmarshalYAML rejects unknown keys and defaults an absent divisor to 1.
func (l *Line) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.AliasNode && n.Alias != nil {
		n = n.Alias
	}
	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: a line must be a mapping", n.Line)
	}
	hasDivisor := false
	for i := 0; i+1 < len(n.Content); i += 2 {
		k := n.Content[i]
		if !lineKeys[k.Value] {
			return fmt.Errorf("line %d: field %s not found in type costs.Line", k.Line, k.Value)
		}
		if k.Value == "divisor" {
			hasDivisor = true
		}
	}
	type plain struct {
		Label     string `yaml:"label"`
		Quantity  Value  `yaml:"quantity"`
		UnitPrice Value  `yaml:"unit_price"`
		Divisor   int    `yaml:"divisor"`
	}
	var p plain
	if err := n.Decode(&p); err != nil {
		return err
	}
	if !hasDivisor {
		p.Divisor = 1
	}
	*l = Line(p)
	return nil
}

// Level is a funding level; the first is the base.
type Level struct {
	ID            string        `yaml:"id"`
	Name          string        `yaml:"name"`
	Description   string        `yaml:"description"`
	Sqft          int           `yaml:"sqft"`
	ReserveMonths ReserveMonths `yaml:"reserve_months"`
}

// ReserveMonths is the months of reserve a level adds.
type ReserveMonths struct {
	Low      int `yaml:"low"`
	Expected int `yaml:"expected"`
	High     int `yaml:"high"`
}

// Category groups startup items.
type Category struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

// Item is a startup item, or a monthly item when it has no Category.
type Item struct {
	ID         string `yaml:"id"`
	Name       string `yaml:"name"`
	Category   string `yaml:"category"`
	Level      string `yaml:"level"`
	Low        Money  `yaml:"low"`
	Expected   Money  `yaml:"expected"`
	High       Money  `yaml:"high"`
	Status     string `yaml:"status"`
	OwnerInput bool   `yaml:"owner_input"`
	Source     string `yaml:"source"`
	Notes      string `yaml:"notes"`
	Lines      []Line `yaml:"lines"`

	// Resolved holds the evaluated lines of an item that has Lines.
	Resolved []ResolvedLine `yaml:"-"`

	ReserveMonths map[string]ReserveMonths `yaml:"reserve_months"`
}
