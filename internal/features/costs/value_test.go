package costs

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type fakeEnv map[string][3]string

func (f fakeEnv) Lookup(kind, id, field string, m Member) (*big.Rat, error) {
	key := kind + "." + id
	if field != "" {
		key += "." + field
	}
	v, ok := f[key]
	if !ok {
		return nil, fmt.Errorf("unknown reference %s", key)
	}
	r, _ := new(big.Rat).SetString(v[m])
	return r, nil
}

func decodeValue(t *testing.T, src string) (Value, error) {
	t.Helper()
	var doc struct {
		V Value `yaml:"v"`
	}
	err := yaml.Unmarshal([]byte(src), &doc)
	return doc.V, err
}

func mustValue(t *testing.T, src string) Value {
	t.Helper()
	v, err := decodeValue(t, src)
	if err != nil {
		t.Fatalf("decode %q: %v", src, err)
	}
	return v
}

func evalAll(t *testing.T, v Value, env Env) [3]string {
	t.Helper()
	var out [3]string
	for m := Low; m <= High; m++ {
		r, err := v.Eval(m, env)
		if err != nil {
			t.Fatalf("Eval(%d): %v", m, err)
		}
		out[m] = r.RatString()
	}
	return out
}

func TestValueYAMLNumber(t *testing.T) {
	got := evalAll(t, mustValue(t, "v: 17.5"), nil)
	if want := [3]string{"35/2", "35/2", "35/2"}; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := evalAll(t, mustValue(t, "v: 0"), nil); got != [3]string{"0", "0", "0"} {
		t.Errorf("zero: got %v", got)
	}
}

func TestValueYAMLMap(t *testing.T) {
	got := evalAll(t, mustValue(t, "v: {low: 25, expected: 30, high: 35.5}"), nil)
	if want := [3]string{"25", "30", "71/2"}; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestValueYAMLExpression(t *testing.T) {
	if got := evalAll(t, mustValue(t, `v: "40 * 8"`), nil); got != [3]string{"320", "320", "320"} {
		t.Errorf("got %v", got)
	}
	if got := evalAll(t, mustValue(t, "v: 40 * 8"), nil); got != [3]string{"320", "320", "320"} {
		t.Errorf("unquoted: got %v", got)
	}
}

func TestValueYAMLErrors(t *testing.T) {
	tests := []struct{ name, src, want string }{
		{"missing key", "\nv: {low: 1, expected: 2}", "line 2: value is missing key \"high\""},
		{"unknown key", "v: {low: 1, expected: 2, high: 3, mid: 2}", "unknown key \"mid\""},
		{"duplicate key", "v: {low: 1, low: 2, expected: 2, high: 3}", "duplicate key \"low\""},
		{"negative number", "\nv: -5", "line 2: invalid number \"-5\""},
		{"negative in map", "v: {low: -1, expected: 2, high: 3}", "line 1: invalid number \"-1\""},
		{"expression in map", `v: {low: "1 + 1", expected: 2, high: 3}`, "want a plain number"},
		{"exponent", "v: 1e3", "invalid number"},
		{"too many digits", "v: 1234567890.123456", "more than 15 digits"},
		{"bool", "v: true", "line 1: value must be"},
		{"sequence", "v: [1, 2]", "line 1: value must be"},
		{"bad expression", "\nv: \"1 +\"", "line 2:"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decodeValue(t, tt.src)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v; want containing %q", err, tt.want)
			}
		})
	}
	if _, err := decodeValue(t, "v: 123456789012345"); err != nil {
		t.Errorf("15 digits should be accepted: %v", err)
	}
}

func TestExprPrecedence(t *testing.T) {
	tests := map[string]string{
		"2 + 3 * 4":       "14",
		"(2 + 3) * 4":     "20",
		"10 / 4":          "5/2",
		"10 - 4 - 3":      "3",
		"8 / 2 / 2":       "2",
		"2 * (3 + 4) * 2": "28",
		"1.5 + 0.25":      "7/4",
		"  7  ":           "7",
		"((3))":           "3",
	}
	for src, want := range tests {
		got := evalAll(t, mustValue(t, "v: "+fmt.Sprintf("%q", src)), nil)
		if got != [3]string{want, want, want} {
			t.Errorf("%q = %v, want %s", src, got, want)
		}
	}
}

func TestExprRefs(t *testing.T) {
	v := mustValue(t, `v: "input.rent-free-months"`)
	if want := []Ref{{Kind: "input", ID: "rent-free-months"}}; !reflect.DeepEqual(v.Refs(), want) {
		t.Errorf("Refs = %v, want %v", v.Refs(), want)
	}

	v = mustValue(t, `v: "input.a - input.b"`)
	if want := []Ref{{Kind: "input", ID: "a"}, {Kind: "input", ID: "b"}}; !reflect.DeepEqual(v.Refs(), want) {
		t.Errorf("Refs = %v, want %v", v.Refs(), want)
	}
	env := fakeEnv{"input.a": {"10", "20", "30"}, "input.b": {"1", "2", "3"}}
	if got := evalAll(t, v, env); got != [3]string{"9", "18", "27"} {
		t.Errorf("got %v", got)
	}

	v = mustValue(t, `v: "level.base.sqft * input.x + item.rent-plus / (level.my-lvl.added_sqft - 1) + input.x"`)
	want := []Ref{
		{"level", "base", "sqft"},
		{"input", "x", ""},
		{"item", "rent-plus", ""},
		{"level", "my-lvl", "added_sqft"},
	}
	if !reflect.DeepEqual(v.Refs(), want) {
		t.Errorf("Refs = %v, want %v", v.Refs(), want)
	}
	if got := mustValue(t, "v: 5").Refs(); len(got) != 0 {
		t.Errorf("number Refs = %v", got)
	}
}

func TestExprParseErrors(t *testing.T) {
	tests := []struct{ src, want string }{
		{"input.a -input.b", "subtraction needs a space on both sides"},
		{"input.a- input.b", "subtraction needs a space on both sides"},
		{"2-3", "subtraction needs a space on both sides"},
		{"input.a-input.b", "column 14: expected an operator or the end of the expression but found \".\""},
		{"- 3", "column 1: expected a number, a reference, or '(' but found \"-\""},
		{"2 * - 3", "column 5: expected a number"},
		{"", "expected a number, a reference, or '(' but the expression ended"},
		{"1 +", "but the expression ended"},
		{"(1 + 2", "expected ')'"},
		{"1 + 2)", "found \")\""},
		{"1 2", "found \"2\""},
		{"5.", "expected digits after '.'"},
		{"foo.bar", "found \"foo\""},
		{"input", "expected '.' after \"input\""},
		{"input.", "expected an id"},
		{"input.Bad", "expected an id"},
		{"level.base", "expected '.sqft' or '.added_sqft'"},
		{"level.base.area", "expected 'sqft' or 'added_sqft' but found \"area\""},
		{"max(1, 2)", "found \"max\""},
		{"1234567890.123456", "more than 15 digits"},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			_, err := parseExpr(tt.src)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v; want containing %q", err, tt.want)
			}
		})
	}
}

func TestExprEvalErrors(t *testing.T) {
	env := fakeEnv{"input.z": {"0", "0", "0"}, "input.a": {"1", "5", "9"}}
	tests := []struct{ src, want string }{
		{"1 - 2", "negative"},
		{"input.a - 5", "negative"},
		{"1 / 0", "division by zero"},
		{"4 / input.z", "division by zero"},
		{"input.missing + 1", "unknown reference input.missing"},
	}
	for _, tt := range tests {
		v := mustValue(t, "v: "+fmt.Sprintf("%q", tt.src))
		_, err := v.Eval(Low, env)
		if tt.src == "input.a - 5" {
			_, err = v.Eval(Expected, env)
			if err != nil {
				t.Errorf("%q expected: unexpected error %v", tt.src, err)
			}
			_, err = v.Eval(Low, env)
		}
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%q: err = %v; want containing %q", tt.src, err, tt.want)
		}
	}
	if _, err := mustValue(t, `v: "input.a"`).Eval(Low, nil); err == nil {
		t.Error("nil env with a ref should error")
	}
	var zero Value
	if _, err := zero.Eval(Low, nil); err == nil {
		t.Error("zero Value should error")
	}
	if _, err := mustValue(t, "v: 1").Eval(Member(7), nil); err == nil {
		t.Error("invalid member should error")
	}
}

func TestValueEvalPerMember(t *testing.T) {
	env := fakeEnv{
		"input.rent-year": {"35/2", "21", "30"},
		"level.base.sqft": {"1000", "1000", "1000"},
	}
	v := mustValue(t, `v: "level.base.sqft * input.rent-year"`)
	if got := evalAll(t, v, env); got != [3]string{"17500", "21000", "30000"} {
		t.Errorf("got %v", got)
	}
}

func TestRoundCents(t *testing.T) {
	rat := func(s string) *big.Rat {
		r, ok := new(big.Rat).SetString(s)
		if !ok {
			t.Fatalf("bad rat %q", s)
		}
		return r
	}
	tests := []struct {
		in   string
		want Money
	}{
		{"1458.3333", 145833},
		{"729.1666", 72917},
		{"0.005", 1},
		{"0.0049", 0},
		{"0", 0},
		{"1000 * 17.5 / 12", 145833},
		{"2.5", 250},
	}
	for _, tt := range tests {
		in := tt.in
		var r *big.Rat
		if in == "1000 * 17.5 / 12" {
			r = new(big.Rat).Quo(rat("17500"), rat("12"))
		} else {
			r = rat(in)
		}
		got, err := RoundCents(r)
		if err != nil || got != tt.want {
			t.Errorf("RoundCents(%s) = %d, %v; want %d", tt.in, got, err, tt.want)
		}
	}
	exact, err := RoundCents(rat("1/3"))
	if err != nil || exact != 33 {
		t.Errorf("RoundCents(1/3) = %d, %v; want 33", exact, err)
	}
	max := new(big.Rat).SetInt64(math.MaxInt64)
	if _, err := RoundCents(max); err == nil {
		t.Error("overflow: want error")
	}
	big1 := new(big.Rat).SetInt(new(big.Int).Lsh(big.NewInt(1), 80))
	if _, err := RoundCents(big1); err == nil {
		t.Error("large overflow: want error")
	}
	if _, err := RoundCents(nil); err == nil {
		t.Error("nil: want error")
	}
}
