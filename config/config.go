package config

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type XmpMode string

const (
	// XmpModeEmbedded reads XMP data from JPEG's APP1 segment
	XmpModeEmbedded XmpMode = "embedded"

	// XmpModeSeparate reads XMP data from .xmp sidecar file
	XmpModeSeparate XmpMode = "separate"

	// XmpModeSeparateExt reads XMP data from .jpg.xmp sidecar file
	XmpModeSeparateExt XmpMode = "separate_ext"
)

type XmpConfig struct {
	Mode XmpMode `yaml:"mode"` // XMP mode: embedded, separate, or separate_ext
}

type Action struct {
	DeleteRaw    bool `yaml:"deleteRaw"`
	DeleteJpeg   bool `yaml:"deleteJpeg"`
	CompressJpeg bool `yaml:"compressJpeg"`
}

type FileConfig struct {
	RawExtension  string `yaml:"rawExtension"`  // e.g. ".RAF"
	JpegExtension string `yaml:"jpegExtension"` // e.g. ".JPG"
	RawFolder     string `yaml:"rawFolder"`     // e.g. "raw" or "."
	SameDir       bool   `yaml:"sameDir"`       // true if RAWs are in same directory
}

// IsJpeg reports whether the file name has the configured JPEG extension (case-insensitive)
func (f FileConfig) IsJpeg(name string) bool {
	return hasExtension(name, f.JpegExtension)
}

// IsRaw reports whether the file name has the configured RAW extension (case-insensitive)
func (f FileConfig) IsRaw(name string) bool {
	return hasExtension(name, f.RawExtension)
}

func hasExtension(name, ext string) bool {
	return len(name) >= len(ext) && strings.EqualFold(name[len(name)-len(ext):], ext)
}

type ProcessConfig struct {
	TargetMegapixels float64 `yaml:"targetMegapixels"` // Target size for JPEG compression
	JpegQuality      int     `yaml:"jpegQuality"`      // JPEG quality (0-100)
}

type Config struct {
	RatingActions map[int]Action `yaml:"ratingActions"`
	NoJpegAction  Action         `yaml:"noJpegAction"`
	Xmp           XmpConfig      `yaml:"xmp"`
	Files         FileConfig     `yaml:"files"`
	Process       ProcessConfig  `yaml:"process"`
}

func (c *Config) Validate() error {
	// Validate XMP-Mode
	validModes := map[XmpMode]bool{
		XmpModeEmbedded:    true,
		XmpModeSeparate:    true,
		XmpModeSeparateExt: true,
	}
	if !validModes[c.Xmp.Mode] {
		return fmt.Errorf("invalid XMP mode: %s", c.Xmp.Mode)
	}

	// An empty extension would match every file and could delete unrelated files
	if c.Files.RawExtension == "" {
		return fmt.Errorf("RAW extension must not be empty")
	}
	if c.Files.JpegExtension == "" {
		return fmt.Errorf("JPEG extension must not be empty")
	}

	// Deleting the JPEG implies deleting the RAW; otherwise the remaining RAW would be handled by the NoJpegAction
	ratings := make([]int, 0, len(c.RatingActions))
	for rating := range c.RatingActions {
		ratings = append(ratings, rating)
	}
	sort.Ints(ratings)
	for _, rating := range ratings {
		if action := c.RatingActions[rating]; action.DeleteJpeg && !action.DeleteRaw {
			return fmt.Errorf("invalid action for rating %d: deleteJpeg requires deleteRaw", rating)
		}
	}
	return nil
}

// LoadConfig loads config from yaml file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Start from defaults so that omitted values keep sensible settings
	cfg := *NewDefaultConfig()
	// Configured rating actions replace the defaults instead of being merged with them
	defaultRatingActions := cfg.RatingActions
	cfg.RatingActions = nil
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error parsing YAML config: %w", err)
	}
	if cfg.RatingActions == nil {
		cfg.RatingActions = defaultRatingActions
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// NewDefaultConfig creates a config with default values
func NewDefaultConfig() *Config {
	return &Config{
		RatingActions: map[int]Action{
			1: {DeleteRaw: true, DeleteJpeg: true, CompressJpeg: false},
			2: {DeleteRaw: true, DeleteJpeg: false, CompressJpeg: true},
			3: {DeleteRaw: false, DeleteJpeg: false, CompressJpeg: false},
			4: {DeleteRaw: false, DeleteJpeg: false, CompressJpeg: false},
			5: {DeleteRaw: false, DeleteJpeg: false, CompressJpeg: false},
		},
		NoJpegAction: Action{DeleteRaw: true, DeleteJpeg: false, CompressJpeg: false},
		Xmp:          XmpConfig{Mode: XmpModeEmbedded},
		Files: FileConfig{
			RawExtension:  ".RAF",
			JpegExtension: ".JPG",
			RawFolder:     "raw",
			SameDir:       false,
		},
		Process: ProcessConfig{
			TargetMegapixels: 10.0,
			JpegQuality:      95,
		},
	}
}
