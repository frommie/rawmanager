package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
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
		return fmt.Errorf("ungültiger XMP-Mode: %s", c.Xmp.Mode)
	}
	return nil
}

// LoadConfig loads config from yaml file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Fehler beim Lesen der Konfigurationsdatei: %v", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("Fehler beim Parsen der YAML-Konfiguration: %v", err)
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
