package render

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/bengearig/spun-records-and-lounge/internal/bundle"
	"github.com/bengearig/spun-records-and-lounge/internal/feature"
)

type plainFeature struct{ name string }

func (p plainFeature) Name() string { return p.name }

type fakeConfigurable struct {
	got *feature.BundleInfo
	err error
}

func (fakeConfigurable) Name() string { return "fake" }

func (f fakeConfigurable) Configure(info feature.BundleInfo) (feature.Feature, error) {
	if f.err != nil {
		return nil, f.err
	}
	*f.got = info
	return plainFeature{name: "fake"}, nil
}

func fakeBundle() *bundle.Bundle {
	return &bundle.Bundle{
		Dir: "/the/dir",
		Config: bundle.Config{Features: map[string]bundle.FeatureConfig{
			"fake": {Enabled: true, Options: map[string]any{"file": "x.yaml"}},
		}},
	}
}

func TestConfigureFeatures(t *testing.T) {
	t.Run("substitutes in order and passes info", func(t *testing.T) {
		var info feature.BundleInfo
		a, c := plainFeature{"a"}, plainFeature{"c"}
		in := []feature.Feature{a, fakeConfigurable{got: &info}, c}
		out, err := configureFeatures(in, fakeBundle())
		if err != nil {
			t.Fatal(err)
		}
		want := []feature.Feature{a, plainFeature{"fake"}, c}
		if !reflect.DeepEqual(out, want) {
			t.Fatalf("out = %#v, want %#v", out, want)
		}
		if info.Dir != "/the/dir" || !reflect.DeepEqual(info.Options, map[string]any{"file": "x.yaml"}) {
			t.Fatalf("info = %#v", info)
		}
	})
	t.Run("only non-configurable", func(t *testing.T) {
		in := []feature.Feature{plainFeature{"a"}, plainFeature{"b"}}
		out, err := configureFeatures(in, fakeBundle())
		if err != nil || !reflect.DeepEqual(out, in) {
			t.Fatalf("out = %#v, err = %v", out, err)
		}
	})
	t.Run("error is wrapped", func(t *testing.T) {
		in := []feature.Feature{fakeConfigurable{err: errors.New("boom")}}
		_, err := configureFeatures(in, fakeBundle())
		if err == nil || !strings.HasPrefix(err.Error(), "features.fake: ") || !strings.Contains(err.Error(), "boom") {
			t.Fatalf("err = %v", err)
		}
	})
}
