package main

import (
    "os"
    "path/filepath"
    "testing"
)

func TestMain(t *testing.T) {
    // Setup temporary test directory
    tmpDir := t.TempDir()
    testConfigPath := filepath.Join(tmpDir, "config.yaml")

    tests := []struct {
        name    string
        args    []string
        wantErr bool
    }{
        {
            name:    "Default directory",
            args:    []string{},
            wantErr: false,
        },
        {
            name:    "Custom directory",
            args:    []string{t.TempDir()},
            wantErr: false,
        },
        {
            name:    "Custom config",
            args:    []string{"-config", testConfigPath, tmpDir},
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            oldArgs := os.Args
            defer func() { os.Args = oldArgs }()
            
            os.Args = append([]string{"rawmanager"}, tt.args...)
            
            // Create temporary config
            if err := os.WriteFile(testConfigPath, []byte(`
xmp:
  mode: "embedded"
`), 0644); err != nil {
                t.Fatal(err)
            }

            // Execute test..
        })
    }
}