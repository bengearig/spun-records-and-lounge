package feature

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/yuin/goldmark"
)

type basic struct{ name string }

func (b basic) Name() string { return b.name }

type full struct{ basic }

func (full) Extenders() []goldmark.Extender { return nil }
func (full) StageName() string              { return "s" }
func (full) Phase() Phase                   { return PostRender }
func (full) Process(d string) (string, error) {
	return d, nil
}
func (full) Commands() []*cobra.Command { return []*cobra.Command{{Use: "x"}} }
func (full) HeadHTML() string           { return "<meta>" }
func (full) Styles() string             { return "a{}" }
func (full) Scripts() string            { return "1" }

func TestRegisterLookup(t *testing.T) {
	r := NewRegistry()
	r.Register(basic{"alpha"})
	f, err := r.Lookup("alpha")
	if err != nil || f.Name() != "alpha" {
		t.Fatalf("Lookup = %v, %v", f, err)
	}
}

func TestDuplicateRegisterPanics(t *testing.T) {
	r := NewRegistry()
	r.Register(basic{"alpha"})
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	r.Register(basic{"alpha"})
}

func TestLookupUnknownListsKnown(t *testing.T) {
	r := NewRegistry()
	r.Register(basic{"beta"})
	r.Register(basic{"alpha"})
	_, err := r.Lookup("nope")
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{`"nope"`, "alpha, beta"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q missing %q", err, want)
		}
	}
	if _, err := NewRegistry().Lookup("x"); err == nil || !strings.Contains(err.Error(), "none") {
		t.Fatalf("empty registry error = %v", err)
	}
}

func TestEnabled(t *testing.T) {
	r := NewRegistry()
	for _, n := range []string{"c", "a", "b"} {
		r.Register(basic{n})
	}
	got, err := r.Enabled(map[string]bool{"c": true, "a": true, "b": false})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name() != "a" || got[1].Name() != "c" {
		t.Fatalf("Enabled = %v", got)
	}
	if _, err := r.Enabled(map[string]bool{"zzz": false}); err == nil {
		t.Fatal("expected error for unknown feature")
	}
	if got, err := r.Enabled(nil); err != nil || len(got) != 0 {
		t.Fatalf("Enabled(nil) = %v, %v", got, err)
	}
}

func TestAllSorted(t *testing.T) {
	r := NewRegistry()
	r.Register(basic{"b"})
	r.Register(basic{"a"})
	all := r.All()
	if len(all) != 2 || all[0].Name() != "a" {
		t.Fatalf("All = %v", all)
	}
}

func TestCapabilityDetection(t *testing.T) {
	var plain Feature = basic{"p"}
	if _, ok := plain.(MarkdownExtender); ok {
		t.Error("basic should not be MarkdownExtender")
	}
	if _, ok := plain.(Stage); ok {
		t.Error("basic should not be Stage")
	}
	if _, ok := plain.(CommandProvider); ok {
		t.Error("basic should not be CommandProvider")
	}
	if _, ok := plain.(Assets); ok {
		t.Error("basic should not be Assets")
	}

	var f Feature = full{basic{"f"}}
	if _, ok := f.(MarkdownExtender); !ok {
		t.Error("want MarkdownExtender")
	}
	if s, ok := f.(Stage); !ok || s.Phase() != PostRender {
		t.Error("want PostRender Stage")
	}
	if c, ok := f.(CommandProvider); !ok || len(c.Commands()) != 1 {
		t.Error("want CommandProvider")
	}
	if a, ok := f.(Assets); !ok || a.HeadHTML() == "" {
		t.Error("want Assets")
	}
}

func TestPhaseOrder(t *testing.T) {
	if !(PreRender < PostRender) {
		t.Fatal("PreRender must precede PostRender")
	}
}
