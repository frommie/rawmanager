package counter

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/frommie/rawmanager/config"
)

func TestFileCounter_CountFiles(t *testing.T) {
    tests := []struct {
        name          string
        setupFiles    []string
        wantJpegCount int
        wantRawCount  int
        config        *config.Config
    }{
        {
            name: "Empty Directory",
            setupFiles: []string{},
            wantJpegCount: 0,
            wantRawCount:  0,
        },
        {
            name: "Mixed files",
            setupFiles: []string{
                "foto1.JPG",
                "foto1.RAF",
                "foto2.JPG",
                "foto3.RAF",
                "andere.txt",
            },
            wantJpegCount: 2,
            wantRawCount:  2,
        },
        {
            name: "Nested Directories",
            setupFiles: []string{
                "ordner1/foto1.JPG",
                "ordner1/raw/foto1.RAF",
                "ordner2/foto2.JPG",
                "ordner2/raw/foto2.RAF",
            },
            wantJpegCount: 2,
            wantRawCount:  2,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test-Setup
            tmpDir := t.TempDir()
            for _, file := range tt.setupFiles {
                path := filepath.Join(tmpDir, file)
                if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
                    t.Fatalf("Error while creating the test directory: %v", err)
                }
                if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
                    t.Fatalf("Error creating the test file: %v", err)
                }
            }

            // Test durchführen
            counter := &FileCounter{}
            cfg := tt.config
            if cfg == nil {
                cfg = config.NewDefaultConfig()
            }

            if err := counter.CountFiles(tmpDir, cfg); err != nil {
                t.Errorf("CountFiles() error = %v", err)
                return
            }

            // Ergebnisse überprüfen
            if counter.JpegCount != tt.wantJpegCount {
                t.Errorf("JpegCount = %v, want %v", counter.JpegCount, tt.wantJpegCount)
            }
            if counter.RawCount != tt.wantRawCount {
                t.Errorf("RawCount = %v, want %v", counter.RawCount, tt.wantRawCount)
            }
        })
    }
}