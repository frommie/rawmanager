package jpeg

import (
	"bytes"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/disintegration/imaging"
	"github.com/frommie/rawmanager/config"
	"github.com/frommie/rawmanager/testutils"
	"github.com/frommie/rawmanager/xmp"
)

func TestResizeWithXMP_SmallImageIsUnchanged(t *testing.T) {
	jpgPath := filepath.Join(t.TempDir(), "test.jpg")
	if err := testutils.CreateTestJPEGWithEmbeddedXMP(t, jpgPath, 4); err != nil {
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

	rating, err := xmp.ReadRating(jpgPath, cfg.Xmp.Mode)
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
