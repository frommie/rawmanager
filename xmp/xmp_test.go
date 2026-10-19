package xmp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/frommie/rawmanager/config"
	"github.com/frommie/rawmanager/testutils"
)

func TestGetRatingFromSidecar(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(string) (string, error)
		want      int
		wantErr   bool
	}{
		{
			name: "Rating 4 from XMP",
			setupFunc: func(dir string) (string, error) {
				xmpPath := filepath.Join(dir, "test.xmp")
				err := testutils.CreateTestXMP(t, xmpPath, 4)
				return xmpPath, err
			},
			want:    4,
			wantErr: false,
		},
		{
			name: "Invalid XMP file",
			setupFunc: func(dir string) (string, error) {
				xmpPath := filepath.Join(dir, "invalid.xmp")
				err := os.WriteFile(xmpPath, []byte("invalid"), 0644)
				return xmpPath, err
			},
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			tmpDir := t.TempDir()
			xmpPath, err := tt.setupFunc(tmpDir)
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			// Execute tests
			got, err := GetRatingFromSidecar(xmpPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRatingFromSidecar() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && got != tt.want {
				t.Errorf("GetRatingFromSidecar() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRating(t *testing.T) {
	const header = `<x:xmpmeta xmlns:x="adobe:ns:meta/"><rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">`
	const footer = `</rdf:RDF></x:xmpmeta>`
	msRating := func(r int) string {
		return fmt.Sprintf(`<rdf:Description xmlns:MicrosoftPhoto="http://ns.microsoft.com/photo/1.0/"><MicrosoftPhoto:Rating>%d</MicrosoftPhoto:Rating></rdf:Description>`, r)
	}

	tests := []struct {
		name string
		body string
		want int
	}{
		{"Adobe rating as element", `<rdf:Description xmlns:xmp="http://ns.adobe.com/xap/1.0/"><xmp:Rating>3</xmp:Rating></rdf:Description>`, 3},
		{"Adobe rating as attribute", `<rdf:Description xmlns:xmp="http://ns.adobe.com/xap/1.0/" xmp:Rating="3"/>`, 3},
		{"Microsoft rating 1", msRating(1), 1},
		{"Microsoft rating 25", msRating(25), 2},
		{"Microsoft rating 50", msRating(50), 3},
		{"Microsoft rating 75", msRating(75), 4},
		{"Microsoft rating 99", msRating(99), 5},
		{"Microsoft rating as attribute", `<rdf:Description xmlns:MicrosoftPhoto="http://ns.microsoft.com/photo/1.0/" MicrosoftPhoto:Rating="99"/>`, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetRating([]byte(header + tt.body + footer))
			if err != nil {
				t.Fatalf("GetRating() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("GetRating() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadRating(t *testing.T) {
	tests := []struct {
		name  string
		mode  config.XmpMode
		setup func(t *testing.T, dir string) error
	}{
		{
			name: "Embedded XMP",
			mode: config.XmpModeEmbedded,
			setup: func(t *testing.T, dir string) error {
				return testutils.CreateTestJPEGWithEmbeddedXMP(t, filepath.Join(dir, "img.JPG"), 4)
			},
		},
		{
			name: "Separate .xmp file",
			mode: config.XmpModeSeparate,
			setup: func(t *testing.T, dir string) error {
				return testutils.CreateTestXMP(t, filepath.Join(dir, "img.xmp"), 4)
			},
		},
		{
			name: "Separate .JPG.xmp file",
			mode: config.XmpModeSeparateExt,
			setup: func(t *testing.T, dir string) error {
				return testutils.CreateTestXMP(t, filepath.Join(dir, "img.JPG.xmp"), 4)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := tt.setup(t, dir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			got, err := ReadRating(filepath.Join(dir, "img.JPG"), tt.mode)
			if err != nil {
				t.Fatalf("ReadRating() error = %v", err)
			}
			if got != 4 {
				t.Errorf("ReadRating() = %v, want 4", got)
			}
		})
	}
}

func TestReadRating_InvalidMode(t *testing.T) {
	if _, err := ReadRating("img.JPG", "invalid"); err == nil {
		t.Error("ReadRating() error = nil, want error")
	}
}

func TestExtractXmpData_NoXmp(t *testing.T) {
	jpgPath := filepath.Join(t.TempDir(), "img.jpg")
	if err := testutils.CreateEmptyJPEG(t, jpgPath); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	data, err := os.ReadFile(jpgPath)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	if _, err := ExtractXmpData(data); !errors.Is(err, ErrNoRating) {
		t.Errorf("ExtractXmpData() error = %v, want ErrNoRating", err)
	}
}
