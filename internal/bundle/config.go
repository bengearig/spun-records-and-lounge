package bundle

import (
	"gopkg.in/yaml.v3"
)

const ConfigFile = "spun.yaml"

type Config struct {
	Title     string                   `yaml:"title"`
	Subtitle  string                   `yaml:"subtitle"`
	Author    string                   `yaml:"author"`
	Date      string                   `yaml:"date"`
	Documents []string                 `yaml:"documents"`
	Template  string                   `yaml:"template"`
	Output    string                   `yaml:"output"`
	Page      PageConfig               `yaml:"page"`
	Numbering NumberingConfig          `yaml:"numbering"`
	Features  map[string]FeatureConfig `yaml:"-"`
}

type PageConfig struct {
	Size    string `yaml:"size"`
	Margins string `yaml:"margins"`
	Numbers bool   `yaml:"numbers"`
}

type NumberingConfig struct {
	Enabled bool `yaml:"enabled"`
	Depth   int  `yaml:"depth"`
}

// FeatureConfig is a feature's bundle setting: a bool, or a mapping with an
// optional `enabled` (default true) plus feature-specific options.
type FeatureConfig struct {
	Enabled bool
	Options map[string]any
}

func decodeFeature(name string, node *yaml.Node) (FeatureConfig, error) {
	field := "features." + name
	switch node.Kind {
	case yaml.ScalarNode:
		var b bool
		if node.Tag != "!!bool" || node.Decode(&b) != nil {
			return FeatureConfig{}, fieldErr(field, "line %d: must be a bool or a mapping", node.Line)
		}
		return FeatureConfig{Enabled: b}, nil
	case yaml.MappingNode:
		var opts map[string]any
		if err := node.Decode(&opts); err != nil {
			return FeatureConfig{}, fieldErr(field, "%v", err)
		}
		enabled := true
		if v, ok := opts["enabled"]; ok {
			b, isBool := v.(bool)
			if !isBool {
				return FeatureConfig{}, fieldErr(field+".enabled", "must be a bool")
			}
			enabled = b
			delete(opts, "enabled")
		}
		return FeatureConfig{Enabled: enabled, Options: opts}, nil
	}
	return FeatureConfig{}, fieldErr(field, "line %d: must be a bool or a mapping", node.Line)
}

func defaultConfig() Config {
	return Config{
		Template:  "default",
		Page:      PageConfig{Size: "letter", Margins: "1in", Numbers: true},
		Numbering: NumberingConfig{Enabled: true, Depth: 3},
	}
}
