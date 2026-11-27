package costs

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Member selects which of the three estimates an evaluation uses.
type Member int

const (
	Low Member = iota
	Expected
	High
)

const memberCount = 3

var memberKeys = [memberCount]string{"low", "expected", "high"}

// Ref is one reference inside an expression. Field is empty except for levels.
type Ref struct {
	Kind  string
	ID    string
	Field string
}

// Env resolves references for one member. Kind is input, item, or level.
type Env interface {
	Lookup(kind, id, field string, m Member) (*big.Rat, error)
}

// Value is a plain number, a low/expected/high map, or an expression.
type Value struct {
	members [memberCount]expr
}

const maxNumberDigits = 15

var numberPattern = regexp.MustCompile(`^\d+(\.\d+)?$`)

func parseNumber(s string) (*big.Rat, error) {
	if !numberPattern.MatchString(s) {
		return nil, fmt.Errorf("invalid number %q: want a non-negative decimal", s)
	}
	if len(strings.ReplaceAll(s, ".", "")) > maxNumberDigits {
		return nil, fmt.Errorf("number %q has more than %d digits", s, maxNumberDigits)
	}
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		return nil, fmt.Errorf("invalid number %q", s)
	}
	return r, nil
}

func isNumberNode(n *yaml.Node) bool {
	return n.Kind == yaml.ScalarNode && (n.Tag == "!!int" || n.Tag == "!!float")
}

func numberFromNode(n *yaml.Node) (*big.Rat, error) {
	if !isNumberNode(n) {
		return nil, fmt.Errorf("line %d: want a plain number", n.Line)
	}
	r, err := parseNumber(n.Value)
	if err != nil {
		return nil, fmt.Errorf("line %d: %w", n.Line, err)
	}
	return r, nil
}

// UnmarshalYAML accepts a number, a {low, expected, high} map, or an expression string.
func (v *Value) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.AliasNode && n.Alias != nil {
		n = n.Alias
	}
	var out Value
	switch {
	case isNumberNode(n):
		r, err := numberFromNode(n)
		if err != nil {
			return err
		}
		for i := range out.members {
			out.members[i] = numExpr{r}
		}
	case n.Kind == yaml.ScalarNode && n.Tag == "!!str":
		e, err := parseExpr(n.Value)
		if err != nil {
			return fmt.Errorf("line %d: %w", n.Line, err)
		}
		for i := range out.members {
			out.members[i] = e
		}
	case n.Kind == yaml.MappingNode:
		seen := map[string]bool{}
		for i := 0; i+1 < len(n.Content); i += 2 {
			k, val := n.Content[i], n.Content[i+1]
			idx := -1
			for j, name := range memberKeys {
				if k.Kind == yaml.ScalarNode && k.Value == name {
					idx = j
				}
			}
			if idx < 0 {
				return fmt.Errorf("line %d: unknown key %q in value: want low, expected, and high", k.Line, k.Value)
			}
			if seen[k.Value] {
				return fmt.Errorf("line %d: duplicate key %q in value", k.Line, k.Value)
			}
			seen[k.Value] = true
			r, err := numberFromNode(val)
			if err != nil {
				return fmt.Errorf("%s: %w", k.Value, err)
			}
			out.members[idx] = numExpr{r}
		}
		for _, name := range memberKeys {
			if !seen[name] {
				return fmt.Errorf("line %d: value is missing key %q: want low, expected, and high", n.Line, name)
			}
		}
	default:
		return fmt.Errorf("line %d: value must be a number, a {low, expected, high} map, or an expression string", n.Line)
	}
	*v = out
	return nil
}

// Eval computes the value for member m, resolving references through env.
func (v Value) Eval(m Member, env Env) (*big.Rat, error) {
	if m < Low || m > High {
		return nil, fmt.Errorf("invalid member %d", int(m))
	}
	e := v.members[m]
	if e == nil {
		return nil, errors.New("value is not set")
	}
	return e.eval(m, env)
}

// Refs lists every reference in the value, in order of appearance.
func (v Value) Refs() []Ref {
	var refs []Ref
	seen := map[Ref]bool{}
	for _, e := range v.members {
		if e == nil {
			continue
		}
		for _, r := range e.refs(nil) {
			if !seen[r] {
				seen[r] = true
				refs = append(refs, r)
			}
		}
	}
	return refs
}

var (
	hundred = big.NewRat(100, 1)
	half    = big.NewRat(1, 2)
)

// RoundCents rounds r dollars half up to whole cents.
func RoundCents(r *big.Rat) (Money, error) {
	if r == nil {
		return 0, errors.New("no amount to round")
	}
	x := new(big.Rat).Mul(r, hundred)
	x.Add(x, half)
	// Denominators are positive, so Euclidean division is floor division.
	q := new(big.Int).Div(x.Num(), x.Denom())
	if !q.IsInt64() {
		return 0, errOverflow
	}
	return Money(q.Int64()), nil
}

type expr interface {
	eval(m Member, env Env) (*big.Rat, error)
	refs(acc []Ref) []Ref
}

type numExpr struct{ n *big.Rat }

func (e numExpr) eval(Member, Env) (*big.Rat, error) { return new(big.Rat).Set(e.n), nil }
func (e numExpr) refs(acc []Ref) []Ref               { return acc }

type refExpr struct{ ref Ref }

func (e refExpr) eval(m Member, env Env) (*big.Rat, error) {
	if env == nil {
		return nil, fmt.Errorf("cannot resolve %s: no environment", e.ref)
	}
	r, err := env.Lookup(e.ref.Kind, e.ref.ID, e.ref.Field, m)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, fmt.Errorf("%s resolved to no value", e.ref)
	}
	return new(big.Rat).Set(r), nil
}
func (e refExpr) refs(acc []Ref) []Ref { return append(acc, e.ref) }

func (r Ref) String() string {
	s := r.Kind + "." + r.ID
	if r.Field != "" {
		s += "." + r.Field
	}
	return s
}

type binExpr struct {
	op   byte
	l, r expr
}

func (e binExpr) eval(m Member, env Env) (*big.Rat, error) {
	l, err := e.l.eval(m, env)
	if err != nil {
		return nil, err
	}
	r, err := e.r.eval(m, env)
	if err != nil {
		return nil, err
	}
	switch e.op {
	case '+':
		return l.Add(l, r), nil
	case '-':
		if l.Cmp(r) < 0 {
			return nil, fmt.Errorf("subtraction %s - %s is negative", l.FloatString(4), r.FloatString(4))
		}
		return l.Sub(l, r), nil
	case '*':
		return l.Mul(l, r), nil
	default:
		if r.Sign() == 0 {
			return nil, errors.New("division by zero")
		}
		return l.Quo(l, r), nil
	}
}

func (e binExpr) refs(acc []Ref) []Ref { return e.r.refs(e.l.refs(acc)) }

type exprParser struct {
	s   string
	pos int
}

func parseExpr(s string) (expr, error) {
	p := &exprParser{s: s}
	e, err := p.expr()
	if err != nil {
		return nil, err
	}
	p.skipSpace()
	if p.pos < len(p.s) {
		return nil, p.errf("expected an operator or the end of the expression but found %q", p.s[p.pos:p.pos+1])
	}
	return e, nil
}

func (p *exprParser) errf(format string, args ...any) error {
	return fmt.Errorf("expression %q: column %d: %s", p.s, p.pos+1, fmt.Sprintf(format, args...))
}

func (p *exprParser) skipSpace() {
	for p.pos < len(p.s) && (p.s[p.pos] == ' ' || p.s[p.pos] == '\t') {
		p.pos++
	}
}

func (p *exprParser) peek() byte {
	if p.pos < len(p.s) {
		return p.s[p.pos]
	}
	return 0
}

func (p *exprParser) expr() (expr, error) {
	left, err := p.term()
	if err != nil {
		return nil, err
	}
	for {
		p.skipSpace()
		op := p.peek()
		if op != '+' && op != '-' {
			return left, nil
		}
		if op == '-' && !(p.pos > 0 && isSpace(p.s[p.pos-1]) && p.pos+1 < len(p.s) && isSpace(p.s[p.pos+1])) {
			return nil, p.errf("subtraction needs a space on both sides of '-'")
		}
		p.pos++
		right, err := p.term()
		if err != nil {
			return nil, err
		}
		left = binExpr{op, left, right}
	}
}

func (p *exprParser) term() (expr, error) {
	left, err := p.factor()
	if err != nil {
		return nil, err
	}
	for {
		p.skipSpace()
		op := p.peek()
		if op != '*' && op != '/' {
			return left, nil
		}
		p.pos++
		right, err := p.factor()
		if err != nil {
			return nil, err
		}
		left = binExpr{op, left, right}
	}
}

func (p *exprParser) factor() (expr, error) {
	p.skipSpace()
	c := p.peek()
	switch {
	case c == '(':
		p.pos++
		e, err := p.expr()
		if err != nil {
			return nil, err
		}
		p.skipSpace()
		if p.peek() != ')' {
			return nil, p.errf("expected ')'")
		}
		p.pos++
		return e, nil
	case isDigit(c):
		return p.number()
	case isSlugChar(c):
		return p.ref()
	case c == 0:
		return nil, p.errf("expected a number, a reference, or '(' but the expression ended")
	default:
		return nil, p.errf("expected a number, a reference, or '(' but found %q", string(c))
	}
}

func (p *exprParser) number() (expr, error) {
	start := p.pos
	p.digits()
	if p.peek() == '.' {
		p.pos++
		if !isDigit(p.peek()) {
			return nil, p.errf("expected digits after '.'")
		}
		p.digits()
	}
	r, err := parseNumber(p.s[start:p.pos])
	if err != nil {
		p.pos = start
		return nil, p.errf("%v", err)
	}
	return numExpr{r}, nil
}

func (p *exprParser) digits() {
	for isDigit(p.peek()) {
		p.pos++
	}
}

func (p *exprParser) ref() (expr, error) {
	start := p.pos
	for isSlugChar(p.peek()) {
		p.pos++
	}
	kind := p.s[start:p.pos]
	if kind != "input" && kind != "item" && kind != "level" {
		p.pos = start
		return nil, p.errf("expected a number, 'input.<id>', 'item.<id>', or 'level.<id>.sqft' but found %q", kind)
	}
	if p.peek() != '.' {
		return nil, p.errf("expected '.' after %q", kind)
	}
	p.pos++
	id, err := p.id()
	if err != nil {
		return nil, err
	}
	ref := Ref{Kind: kind, ID: id}
	if kind == "level" {
		if p.peek() != '.' {
			return nil, p.errf("expected '.sqft' or '.added_sqft' after level id")
		}
		p.pos++
		fstart := p.pos
		for isSlugChar(p.peek()) || p.peek() == '_' {
			p.pos++
		}
		field := p.s[fstart:p.pos]
		if field != "sqft" && field != "added_sqft" {
			p.pos = fstart
			return nil, p.errf("expected 'sqft' or 'added_sqft' but found %q", field)
		}
		ref.Field = field
	}
	return refExpr{ref}, nil
}

// id reads a slug; a hyphen continues it only when a slug character follows.
func (p *exprParser) id() (string, error) {
	start := p.pos
	if !isSlugChar(p.peek()) {
		return "", p.errf("expected an id (lowercase letters, digits, and hyphens)")
	}
	for {
		for isSlugChar(p.peek()) {
			p.pos++
		}
		if p.peek() == '-' && p.pos+1 < len(p.s) && isSlugChar(p.s[p.pos+1]) {
			p.pos++
			continue
		}
		return p.s[start:p.pos], nil
	}
}

func isDigit(c byte) bool    { return c >= '0' && c <= '9' }
func isSpace(c byte) bool    { return c == ' ' || c == '\t' }
func isSlugChar(c byte) bool { return isDigit(c) || (c >= 'a' && c <= 'z') }
