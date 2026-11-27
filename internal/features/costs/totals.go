package costs

import "fmt"

// Range is an amount at the low, expected, and high estimates.
type Range struct {
	Low, Expected, High Money
}

func (r Range) combine(o Range, op func(a, b Money) (Money, error)) (Range, error) {
	var out Range
	var err error
	if out.Low, err = op(r.Low, o.Low); err != nil {
		return Range{}, err
	}
	if out.Expected, err = op(r.Expected, o.Expected); err != nil {
		return Range{}, err
	}
	if out.High, err = op(r.High, o.High); err != nil {
		return Range{}, err
	}
	return out, nil
}

// Add returns r+o or an error on overflow.
func (r Range) Add(o Range) (Range, error) {
	return r.combine(o, Money.Add)
}

// Sub returns r-o or an error on overflow.
func (r Range) Sub(o Range) (Range, error) {
	return r.combine(o, Money.Sub)
}

// MulMonths multiplies low, expected, and high by the matching month counts.
func (r Range) MulMonths(m ReserveMonths) (Range, error) {
	var out Range
	var err error
	if out.Low, err = r.Low.MulInt(m.Low); err != nil {
		return Range{}, err
	}
	if out.Expected, err = r.Expected.MulInt(m.Expected); err != nil {
		return Range{}, err
	}
	if out.High, err = r.High.MulInt(m.High); err != nil {
		return Range{}, err
	}
	return out, nil
}

func itemRange(it Item) Range {
	return Range{Low: it.Low, Expected: it.Expected, High: it.High}
}

// CategoryTotal is the startup total of one category.
type CategoryTotal struct {
	ID, Name string
	Range    Range
}

// LevelTotal is the totals of one level. Monthly and ReserveMonths run
// through the level; Adds is the change in Goal from the previous level.
type LevelTotal struct {
	ID, Name                            string
	Items, Monthly, Reserve, Goal, Adds Range
	ReserveMonths                       ReserveMonths
}

// Totals is every total derived from the data.
type Totals struct {
	Categories                       []CategoryTotal
	Levels                           []LevelTotal
	Startup, Monthly, Reserve, Total Range
}

// Totals computes the totals on demand.
func (d *Data) Totals() (Totals, error) {
	var t Totals
	catIdx := make(map[string]int, len(d.Categories))
	for i, c := range d.Categories {
		catIdx[c.ID] = i
		t.Categories = append(t.Categories, CategoryTotal{ID: c.ID, Name: c.Name})
	}
	levelIdx := make(map[string]int, len(d.Levels))
	t.Levels = make([]LevelTotal, len(d.Levels))
	for i, l := range d.Levels {
		levelIdx[l.ID] = i
		t.Levels[i] = LevelTotal{ID: l.ID, Name: l.Name}
	}

	for _, it := range d.Items {
		var err error
		if i, ok := catIdx[it.Category]; ok {
			if t.Categories[i].Range, err = t.Categories[i].Range.Add(itemRange(it)); err != nil {
				return Totals{}, fmt.Errorf("totaling category %q: %w", it.Category, err)
			}
		}
		if i, ok := levelIdx[it.Level]; ok {
			if t.Levels[i].Items, err = t.Levels[i].Items.Add(itemRange(it)); err != nil {
				return Totals{}, fmt.Errorf("totaling level %q items: %w", it.Level, err)
			}
		}
		if t.Startup, err = t.Startup.Add(itemRange(it)); err != nil {
			return Totals{}, fmt.Errorf("totaling startup: %w", err)
		}
	}

	ownMonthly := make([]Range, len(d.Levels))
	plainMonthly := make([]Range, len(d.Levels))
	for _, it := range d.Monthly {
		i, ok := levelIdx[it.Level]
		if !ok {
			continue
		}
		var err error
		if ownMonthly[i], err = ownMonthly[i].Add(itemRange(it)); err != nil {
			return Totals{}, fmt.Errorf("totaling level %q monthly: %w", it.Level, err)
		}
		if it.ReserveMonths == nil {
			if plainMonthly[i], err = plainMonthly[i].Add(itemRange(it)); err != nil {
				return Totals{}, fmt.Errorf("totaling level %q monthly: %w", it.Level, err)
			}
		}
	}

	var runItems, runMonthly, runPlain, prevGoal Range
	var runMonths ReserveMonths
	for i, l := range d.Levels {
		lt := &t.Levels[i]
		var err error
		if runItems, err = runItems.Add(lt.Items); err != nil {
			return Totals{}, fmt.Errorf("totaling level %q: %w", l.ID, err)
		}
		if runMonthly, err = runMonthly.Add(ownMonthly[i]); err != nil {
			return Totals{}, fmt.Errorf("totaling level %q: %w", l.ID, err)
		}
		if runPlain, err = runPlain.Add(plainMonthly[i]); err != nil {
			return Totals{}, fmt.Errorf("totaling level %q: %w", l.ID, err)
		}
		runMonths.Low += l.ReserveMonths.Low
		runMonths.Expected += l.ReserveMonths.Expected
		runMonths.High += l.ReserveMonths.High

		lt.Monthly = runMonthly
		lt.ReserveMonths = runMonths
		if lt.Reserve, err = runPlain.MulMonths(runMonths); err != nil {
			return Totals{}, fmt.Errorf("totaling level %q reserve: %w", l.ID, err)
		}
		for _, it := range d.Monthly {
			if it.ReserveMonths == nil || levelIdx[it.Level] > i {
				continue
			}
			var own Range
			if own, err = itemRange(it).MulMonths(d.itemMonthsThrough(it, i)); err != nil {
				return Totals{}, fmt.Errorf("totaling level %q reserve: %w", l.ID, err)
			}
			if lt.Reserve, err = lt.Reserve.Add(own); err != nil {
				return Totals{}, fmt.Errorf("totaling level %q reserve: %w", l.ID, err)
			}
		}
		if lt.Goal, err = runItems.Add(lt.Reserve); err != nil {
			return Totals{}, fmt.Errorf("totaling level %q goal: %w", l.ID, err)
		}
		if lt.Adds, err = lt.Goal.Sub(prevGoal); err != nil {
			return Totals{}, fmt.Errorf("totaling level %q adds: %w", l.ID, err)
		}
		prevGoal = lt.Goal
	}

	if n := len(t.Levels); n > 0 {
		last := t.Levels[n-1]
		t.Monthly, t.Reserve, t.Total = last.Monthly, last.Reserve, last.Goal
	}
	return t, nil
}

// itemMonthsThrough is an item's running reserve months through level index n.
func (d *Data) itemMonthsThrough(it Item, n int) ReserveMonths {
	var run ReserveMonths
	for i := 0; i <= n && i < len(d.Levels); i++ {
		m := d.Levels[i].ReserveMonths
		if it.ReserveMonths != nil {
			m = it.ReserveMonths[d.Levels[i].ID]
		}
		run.Low += m.Low
		run.Expected += m.Expected
		run.High += m.High
	}
	return run
}

// ItemReserveMonths returns an item's running reserve months through the last level.
func (d *Data) ItemReserveMonths(it Item) ReserveMonths {
	return d.itemMonthsThrough(it, len(d.Levels)-1)
}

// Category returns the category with the given id.
func (d *Data) Category(id string) (Category, bool) {
	for _, c := range d.Categories {
		if c.ID == id {
			return c, true
		}
	}
	return Category{}, false
}

// Item returns the startup or monthly item with the given id.
func (d *Data) Item(id string) (Item, bool) {
	for _, list := range [][]Item{d.Items, d.Monthly} {
		for _, it := range list {
			if it.ID == id {
				return it, true
			}
		}
	}
	return Item{}, false
}

// Level returns the level with the given id and its index.
func (d *Data) Level(id string) (Level, int, bool) {
	for i, l := range d.Levels {
		if l.ID == id {
			return l, i, true
		}
	}
	return Level{}, -1, false
}
