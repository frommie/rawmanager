package jpeg

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/disintegration/imaging"
	"github.com/frommie/rawmanager/config"
	"github.com/frommie/rawmanager/testutils"
)

func TestGetRatingFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	jpgPath := filepath.Join(tmpDir, "test.jpg")

	// Create valid JPEG with XMP
	if err := testutils.CreateTestJPEGWithEmbeddedXMP(t, jpgPath, 4); err != nil {
		t.Fatalf("Failed to create test JPEG: %v", err)
	}

	// Create test configuration
	cfg := &config.Config{
		Xmp: config.XmpConfig{
			Mode: config.XmpModeEmbedded,
		},
	}

	got, err := GetRatingFromFile(jpgPath, cfg)
	if err != nil {
		t.Errorf("GetRatingFromFile() error = %v", err)
	}
	if got != 4 {
		t.Errorf("GetRatingFromFile() = %v, want 4", got)
	}
}

func TestResizeWithXMP(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T, dir string) (string, error)
		wantErr   bool
	}{
		{
			name: "Successful downsizing",
			setupFunc: func(t *testing.T, dir string) (string, error) {
				path := filepath.Join(dir, "test.jpg")
				if err := createTestJPEGWithRating(t, path, 4); err != nil {
					return "", err
				}
				return path, nil
			},
			wantErr: false,
		},
		{
			name: "Non-existing file",
			setupFunc: func(t *testing.T, dir string) (string, error) {
				return filepath.Join(dir, "nonexistent.jpg"), nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			jpgPath, err := tt.setupFunc(t, tmpDir)
			if err != nil {
				t.Fatalf("Error during test setup: %v", err)
			}

			// Create test configuration
			cfg := &config.Config{
				Process: config.ProcessConfig{
					TargetMegapixels: 10.0,
					JpegQuality:      95,
				},
				Files: config.FileConfig{
					JpegExtension: ".JPG",
					RawExtension:  ".RAF",
				},
			}

			// Call ResizeWithXMP with Verbose parameter
			err = ResizeWithXMP(jpgPath, cfg, false)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResizeWithXMP() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResizeWithXMP_DownsizesLargeImage(t *testing.T) {
	tmpDir := t.TempDir()
	jpgPath := filepath.Join(tmpDir, "large.jpg")

	// 2000x1500 = 3 MP, target is 1 MP
	img := imaging.New(2000, 1500, color.White)
	if err := imaging.Save(img, jpgPath); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	if err := testutils.EmbedXMP(t, jpgPath, 2); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	cfg := config.NewDefaultConfig()
	cfg.Process.TargetMegapixels = 1.0

	if err := ResizeWithXMP(jpgPath, cfg, false); err != nil {
		t.Fatalf("ResizeWithXMP() error = %v", err)
	}

	resized, err := imaging.Open(jpgPath)
	if err != nil {
		t.Fatalf("Failed to open resized image: %v", err)
	}
	if w, h := resized.Bounds().Dx(), resized.Bounds().Dy(); w*h > 1000000 {
		t.Errorf("Image not downsized: %dx%d", w, h)
	}

	rating, err := GetRatingFromFile(jpgPath, cfg)
	if err != nil {
		t.Fatalf("XMP not preserved: %v", err)
	}
	if rating != 2 {
		t.Errorf("Rating after resize = %d, want 2", rating)
	}

	entries, _ := os.ReadDir(tmpDir)
	if len(entries) != 1 {
		t.Errorf("Expected only the resized image in directory, got %d entries", len(entries))
	}
}

// Help functions for tests
func createEmptyJPEG(path string) error {
	img := imaging.New(100, 100, color.White)
	return imaging.Save(img, path)
}

func createTestJPEGWithRating(t *testing.T, path string, rating int) error {
	t.Helper()

	// Create a JPEG with embedded rating
	if err := createEmptyJPEG(path); err != nil {
		return err
	}

	// Use testutils to embed XMP data
	if err := testutils.CreateTestJPEGWithEmbeddedXMP(t, path, rating); err != nil {
		return fmt.Errorf("Error embedding XMP data: %v", err)
	}

	return nil
}

func createTestXMP(path string, rating int) error {
	// Create a test XMP file
	xmpContent := fmt.Sprintf(`<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
    <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
        <rdf:Description rdf:about="" xmlns:xmp="http://ns.adobe.com/xap/1.0/">
            <xmp:Rating>%d</xmp:Rating>
        </rdf:Description>
    </rdf:RDF>
</x:xmpmeta>`, rating)
	return os.WriteFile(path, []byte(xmpContent), 0644)
}
