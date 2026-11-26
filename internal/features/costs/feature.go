// Package costs adds startup cost data, totals, and document directives.
package costs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bengearig/spun-records-and-lounge/internal/feature"
	"github.com/yuin/goldmark"
)

// Name is the feature name used in spun.yaml.
const Name = "costs"

func init() {
	feature.Default.Register(Feature{})
}

// Feature is the unconfigured costs feature; Configure binds it to a bundle.
type Feature struct{}

func (Feature) Name() string { return Name }

// DataFile validates the file option and returns it as a bundle-relative name.
func DataFile(info feature.BundleInfo) (string, error) {
	v, ok := info.Options["file"]
	if !ok || v == nil {
		return "", errors.New("file is required")
	}
	name, ok := v.(string)
	if !ok {
		return "", errors.New("file must be a string")
	}
	if name == "" {
		return "", errors.New("file is required")
	}
	if filepath.IsAbs(name) {
		return "", errors.New("file must be relative to the bundle")
	}
	path := filepath.Join(info.Dir, name)
	rel, err := filepath.Rel(info.Dir, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("file escapes the bundle")
	}
	st, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("file %q does not exist", name)
	}
	if err != nil {
		return "", fmt.Errorf("file %q: %w", name, err)
	}
	if !st.Mode().IsRegular() {
		return "", fmt.Errorf("file %q is not a regular file", name)
	}
	return name, nil
}

// Configure loads and checks the bundle's data file.
func (Feature) Configure(info feature.BundleInfo) (feature.Feature, error) {
	name, err := DataFile(info)
	if err != nil {
		return nil, err
	}
	d, err := LoadIn(info.Dir, name)
	if err != nil {
		return nil, err
	}
	if _, err := d.Totals(); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return configured{data: d}, nil
}

type configured struct {
	data *Data
}

func (configured) Name() string { return Name }

func (c configured) Extenders() []goldmark.Extender {
	return []goldmark.Extender{extension{data: c.data}}
}

type extension struct {
	data *Data
}

func (extension) Extend(goldmark.Markdown) {}
