// Package costs loads and validates the startup cost data file and holds
// the money arithmetic used to total it.
package costs

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Money is an amount in cents.
type Money int64

var moneyPattern = regexp.MustCompile(`^\d+(\.\d{1,2})?$`)

var errOverflow = errors.New("amount overflows")

// ParseMoney reads plain digits with at most two decimals, such as 1500 or 1500.50.
func ParseMoney(s string) (Money, error) {
	if !moneyPattern.MatchString(s) {
		return 0, fmt.Errorf("invalid money value %q: want plain digits with at most two decimals", s)
	}
	whole, frac, _ := strings.Cut(s, ".")
	dollars, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("money value %q is too large", s)
	}
	for len(frac) < 2 {
		frac += "0"
	}
	cents, _ := strconv.ParseInt(frac, 10, 64)
	if dollars > (math.MaxInt64-cents)/100 {
		return 0, fmt.Errorf("money value %q is too large", s)
	}
	return Money(dollars*100 + cents), nil
}

// String formats as $1,234 or $1,234.50.
func (m Money) String() string {
	sign := ""
	u := uint64(m)
	if m < 0 {
		sign = "-"
		u = -u
	}
	dollars := strconv.FormatUint(u/100, 10)
	var b strings.Builder
	for i, r := range dollars {
		if i > 0 && (len(dollars)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if cents := u % 100; cents != 0 {
		return fmt.Sprintf("%s$%s.%02d", sign, b.String(), cents)
	}
	return sign + "$" + b.String()
}

// Add returns m+o or an error on overflow.
func (m Money) Add(o Money) (Money, error) {
	if (o > 0 && m > math.MaxInt64-o) || (o < 0 && m < math.MinInt64-o) {
		return 0, errOverflow
	}
	return m + o, nil
}

// Sub returns m-o or an error on overflow.
func (m Money) Sub(o Money) (Money, error) {
	if (o < 0 && m > math.MaxInt64+o) || (o > 0 && m < math.MinInt64+o) {
		return 0, errOverflow
	}
	return m - o, nil
}

// MulInt returns m*n or an error on overflow.
func (m Money) MulInt(n int) (Money, error) {
	if m == 0 || n == 0 {
		return 0, nil
	}
	k := int64(n)
	r := int64(m) * k
	if (m == math.MinInt64 && k == -1) || (k == math.MinInt64 && m == -1) || r/k != int64(m) {
		return 0, errOverflow
	}
	return Money(r), nil
}

// UnmarshalYAML reads the scalar's text so 1500, 1500.5, and "1500.50" all work.
func (m *Money) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.ScalarNode {
		return fmt.Errorf("line %d: money value must be a plain number", n.Line)
	}
	v, err := ParseMoney(n.Value)
	if err != nil {
		return fmt.Errorf("line %d: %w", n.Line, err)
	}
	*m = v
	return nil
}
