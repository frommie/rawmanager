package jpeg

import (
	"bytes"
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

func TestResizeWithXMP_SmallImageIsUnchanged(t *testing.T) {
	jpgPath := filepath.Join(t.TempDir(), "test.jpg")
	if err := createTestJPEGWithRating(t, jpgPath, 4); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	original, err := os.ReadFile(jpgPath)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	result, err := ResizeWithXMP(jpgPath, config.NewDefaultConfig().Process)
	if err != nil {
		t.Fatalf("ResizeWithXMP() error = %v", err)
	}
	if result.Resized {
		t.Errorf("ResizeWithXMP() resized a %dx%d image, want unchanged", result.Width, result.Height)
	}

	after, err := os.ReadFile(jpgPath)
	if err != nil {
		t.Fatalf("Failed to read image: %v", err)
	}
	if !bytes.Equal(original, after) {
		t.Error("Small image was rewritten, want unchanged file")
	}
}

func TestResizeWithXMP_NonExistingFile(t *testing.T) {
	if _, err := ResizeWithXMP(filepath.Join(t.TempDir(), "nonexistent.jpg"), config.NewDefaultConfig().Process); err == nil {
		t.Error("ResizeWithXMP() error = nil, want error")
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
	if err := os.Chmod(jpgPath, 0640); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	cfg := config.NewDefaultConfig()
	cfg.Process.TargetMegapixels = 1.0

	result, err := ResizeWithXMP(jpgPath, cfg.Process)
	if err != nil {
		t.Fatalf("ResizeWithXMP() error = %v", err)
	}
	if !result.Resized || result.Width*result.Height > 1000000 {
		t.Errorf("ResizeWithXMP() = %+v, want resized to at most 1 MP", result)
	}

	resized, err := imaging.Open(jpgPath)
	if err != nil {
		t.Fatalf("Failed to open resized image: %v", err)
	}
	if w, h := resized.Bounds().Dx(), resized.Bounds().Dy(); w != result.Width || h != result.Height {
		t.Errorf("Image on disk is %dx%d, want %dx%d", w, h, result.Width, result.Height)
	}

	rating, err := GetRatingFromFile(jpgPath, cfg)
	if err != nil {
		t.Fatalf("XMP not preserved: %v", err)
	}
	if rating != 2 {
		t.Errorf("Rating after resize = %d, want 2", rating)
	}

	info, err := os.Stat(jpgPath)
	if err != nil {
		t.Fatalf("Failed to stat image: %v", err)
	}
	if info.Mode().Perm() != 0640 {
		t.Errorf("File mode after resize = %v, want 0640", info.Mode().Perm())
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
