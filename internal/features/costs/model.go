package costs

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
}
