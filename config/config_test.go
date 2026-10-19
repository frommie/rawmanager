package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantErr     bool
	}{
		{
			name: "Invalid XMP configuration",
			yamlContent: `
xmp:
  mode: "invalid"
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			if err := os.WriteFile(configPath, []byte(tt.yamlContent), 0644); err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			_, err := LoadConfig(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	return path
}

func TestLoadConfig_PartialConfigKeepsDefaults(t *testing.T) {
	cfg, err := LoadConfig(writeConfig(t, `
xmp:
  mode: separate
`))
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	defaults := NewDefaultConfig()
	if cfg.Xmp.Mode != XmpModeSeparate {
		t.Errorf("Xmp.Mode = %q, want %q", cfg.Xmp.Mode, XmpModeSeparate)
	}
	if cfg.Files != defaults.Files {
		t.Errorf("Files = %+v, want defaults %+v", cfg.Files, defaults.Files)
	}
	if cfg.Process != defaults.Process {
		t.Errorf("Process = %+v, want defaults %+v", cfg.Process, defaults.Process)
	}
	if len(cfg.RatingActions) != len(defaults.RatingActions) {
		t.Errorf("RatingActions = %v, want defaults", cfg.RatingActions)
	}
}

func TestLoadConfig_RatingActionsReplaceDefaults(t *testing.T) {
	cfg, err := LoadConfig(writeConfig(t, `
ratingActions:
  1:
    deleteRaw: true
`))
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if _, exists := cfg.RatingActions[2]; exists {
		t.Errorf("RatingActions contains default for rating 2, want only configured ratings: %v", cfg.RatingActions)
	}
	if !cfg.RatingActions[1].DeleteRaw {
		t.Errorf("RatingActions[1].DeleteRaw = false, want true")
	}
}

func TestLoadConfig_RejectsEmptyExtensions(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{"Empty RAW extension", "files:\n  rawExtension: \"\"\n"},
		{"Empty JPEG extension", "files:\n  jpegExtension: \"\"\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := LoadConfig(writeConfig(t, tt.yaml)); err == nil {
				t.Error("LoadConfig() error = nil, want error")
			}
		})
	}
}
