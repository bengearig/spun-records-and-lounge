package bundle

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/bengearig/spun-records-and-lounge/internal/feature"
	"gopkg.in/yaml.v3"
)

type Bundle struct {
	Dir    string
	Config Config
	// DocumentPaths are absolute, in Config.Documents order.
	DocumentPaths []string
}

func Load(dir string) (*Bundle, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ConfigFile, err)
	}
	f, err := os.Open(filepath.Join(absDir, ConfigFile))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%s: not found in bundle directory %q", ConfigFile, absDir)
		}
		return nil, fmt.Errorf("%s: %w", ConfigFile, err)
	}
	defer f.Close()

	cfg := defaultConfig()
	file := struct {
		*Config  `yaml:",inline"`
		Features map[string]yaml.Node `yaml:"features"`
	}{Config: &cfg}
	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)
	if err := dec.Decode(&file); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%s: %w", ConfigFile, err)
	}
	if len(file.Features) > 0 {
		cfg.Features = make(map[string]FeatureConfig, len(file.Features))
	}
	for name, node := range file.Features {
		fc, err := decodeFeature(name, &node)
		if err != nil {
			return nil, err
		}
		cfg.Features[name] = fc
	}
	if cfg.Output == "" {
		cfg.Output = filepath.Base(absDir) + ".pdf"
	}

	paths, err := validate(absDir, &cfg)
	if err != nil {
		return nil, err
	}
	return &Bundle{Dir: absDir, Config: cfg, DocumentPaths: paths}, nil
}

func fieldErr(field, format string, args ...any) error {
	return fmt.Errorf("%s: %s: %s", ConfigFile, field, fmt.Sprintf(format, args...))
}

func validate(absDir string, cfg *Config) ([]string, error) {
	if strings.TrimSpace(cfg.Title) == "" {
		return nil, fieldErr("title", "is required")
	}
	paths, err := validateDocuments(absDir, cfg.Documents)
	if err != nil {
		return nil, err
	}
	if cfg.Page.Size != "letter" && cfg.Page.Size != "a4" {
		return nil, fieldErr("page.size", "must be letter or a4, got %q", cfg.Page.Size)
	}
	if cfg.Page.Margins == "" {
		return nil, fieldErr("page.margins", "must not be empty")
	}
	if cfg.Numbering.Depth < 1 || cfg.Numbering.Depth > 6 {
		return nil, fieldErr("numbering.depth", "must be between 1 and 6, got %d", cfg.Numbering.Depth)
	}
	names := make([]string, 0, len(cfg.Features))
	for name := range cfg.Features {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if _, err := feature.Default.Lookup(name); err != nil {
			return nil, fieldErr("features."+name, "%v", err)
		}
	}
	return paths, nil
}

func validateDocuments(absDir string, docs []string) ([]string, error) {
	if len(docs) == 0 {
		return nil, fieldErr("documents", "must list at least one document")
	}
	realDir, err := filepath.EvalSymlinks(absDir)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ConfigFile, err)
	}
	paths := make([]string, 0, len(docs))
	seen := make([]string, 0, len(docs))
	for i, doc := range docs {
		field := fmt.Sprintf("documents[%d]", i)
		if doc == "" {
			return nil, fieldErr(field, "path must not be empty")
		}
		if filepath.IsAbs(doc) {
			return nil, fieldErr(field, "%q must be relative to the bundle", doc)
		}
		joined := filepath.Join(absDir, doc)
		if !within(absDir, joined) {
			return nil, fieldErr(field, "%q escapes the bundle", doc)
		}
		real, err := filepath.EvalSymlinks(joined)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, fieldErr(field, "%q does not exist", doc)
			}
			return nil, fieldErr(field, "%q: %v", doc, err)
		}
		if !within(realDir, real) {
			return nil, fieldErr(field, "%q escapes the bundle", doc)
		}
		info, err := os.Stat(real)
		if err != nil {
			return nil, fieldErr(field, "%q: %v", doc, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fieldErr(field, "%q is not a regular file", doc)
		}
		if slices.Contains(seen, real) {
			return nil, fieldErr(field, "%q is listed more than once", doc)
		}
		seen = append(seen, real)
		paths = append(paths, joined)
	}
	return paths, nil
}

func within(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
