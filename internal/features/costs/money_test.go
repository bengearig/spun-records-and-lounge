package costs

import (
	"math"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseMoney(t *testing.T) {
	valid := map[string]Money{
		"0":       0,
		"1500":    150000,
		"1500.5":  150050,
		"1500.50": 150050,
		"0.05":    5,
	}
	for in, want := range valid {
		got, err := ParseMoney(in)
		if err != nil || got != want {
			t.Errorf("ParseMoney(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	invalid := []string{"", "-1", "$5", "1,500", "1.234", "1e3", "abc", ".5", "5.", "92233720368547759", "99999999999999999999"}
	for _, in := range invalid {
		if got, err := ParseMoney(in); err == nil {
			t.Errorf("ParseMoney(%q) = %d; want error", in, got)
		}
	}
	if got, err := ParseMoney("92233720368547758.07"); err != nil || got != math.MaxInt64 {
		t.Errorf("ParseMoney(max) = %d, %v", got, err)
	}
	if _, err := ParseMoney("92233720368547758.08"); err == nil {
		t.Error("ParseMoney(max+1) succeeded")
	}
}

func TestMoneyString(t *testing.T) {
	cases := map[Money]string{
		0:         "$0",
		150000:    "$1,500",
		150050:    "$1,500.50",
		100000000: "$1,000,000",
		5:         "$0.05",
		99900:     "$999",
		-150050:   "-$1,500.50",
	}
	for in, want := range cases {
		if got := in.String(); got != want {
			t.Errorf("Money(%d).String() = %q; want %q", int64(in), got, want)
		}
	}
}

func TestMoneyArithmetic(t *testing.T) {
	max := Money(math.MaxInt64)
	min := Money(math.MinInt64)

	if got, err := Money(100).Add(250); err != nil || got != 350 {
		t.Errorf("Add = %d, %v", got, err)
	}
	if got, err := Money(100).Sub(250); err != nil || got != -150 {
		t.Errorf("Sub = %d, %v", got, err)
	}
	if got, err := Money(250).MulInt(3); err != nil || got != 750 {
		t.Errorf("MulInt = %d, %v", got, err)
	}
	if got, err := Money(250).MulInt(0); err != nil || got != 0 {
		t.Errorf("MulInt(0) = %d, %v", got, err)
	}
	if got, err := max.Add(0); err != nil || got != max {
		t.Errorf("max.Add(0) = %d, %v", got, err)
	}

	overflows := map[string]func() (Money, error){
		"Add max+1":     func() (Money, error) { return max.Add(1) },
		"Add min-1":     func() (Money, error) { return min.Add(-1) },
		"Sub min-1":     func() (Money, error) { return min.Sub(1) },
		"Sub max-(-1)":  func() (Money, error) { return max.Sub(-1) },
		"MulInt big":    func() (Money, error) { return max.MulInt(2) },
		"MulInt neg":    func() (Money, error) { return min.MulInt(-1) },
		"MulInt neg 2":  func() (Money, error) { return Money(-1).MulInt(math.MinInt) },
		"MulInt medium": func() (Money, error) { return Money(math.MaxInt64 / 2).MulInt(3) },
	}
	for name, f := range overflows {
		if got, err := f(); err == nil {
			t.Errorf("%s = %d; want overflow error", name, got)
		}
	}
}

func TestMoneyUnmarshalYAML(t *testing.T) {
	var ok struct {
		A Money `yaml:"A"`
		B Money `yaml:"B"`
		C Money `yaml:"C"`
	}
	if err := yaml.Unmarshal([]byte("A: 1500\nB: 1500.5\nC: \"1500.50\"\n"), &ok); err != nil {
		t.Fatal(err)
	}
	if ok.A != 150000 || ok.B != 150050 || ok.C != 150050 {
		t.Errorf("decoded %+v", ok)
	}
	for _, in := range []string{"A: 1,500", "A: -5", "A: [1]"} {
		var v struct {
			A Money `yaml:"A"`
		}
		if err := yaml.Unmarshal([]byte(in), &v); err == nil {
			t.Errorf("%q decoded without error", in)
		}
	}
	var v struct {
		A Money `yaml:"A"`
	}
	err := yaml.Unmarshal([]byte("A: 1,500"), &v)
	if err == nil || !contains(err.Error(), `"1,500"`) {
		t.Errorf("error %v does not name the value", err)
	}
}
