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

func TestResizeWithXMP_PreservesMetadataSegments(t *testing.T) {
	jpgPath := filepath.Join(t.TempDir(), "large.jpg")
	if err := imaging.Save(imaging.New(2000, 1500, color.White), jpgPath); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Minimal TIFF structure: little endian header and an empty IFD
	exif := append([]byte("Exif\x00\x00"), "II*\x00\x08\x00\x00\x00\x00\x00\x00\x00\x00\x00"...)
	icc := append([]byte("ICC_PROFILE\x00\x01\x01"), bytes.Repeat([]byte{0x42}, 128)...)
	extendedXmp := append([]byte("http://ns.adobe.com/xmp/extension/\x00"), bytes.Repeat([]byte{0x43}, 64)...)
	mpf := append([]byte("MPF\x00"), bytes.Repeat([]byte{0x44}, 16)...)
	comment := []byte("test comment")

	// Inserted in reverse order, as each segment is inserted directly after SOI
	segments := []struct {
		marker  byte
		payload []byte
	}{
		{0xFE, comment},
		{0xE2, mpf},
		{0xE1, extendedXmp},
		{0xE2, icc},
		{0xE1, exif},
	}
	for _, seg := range segments {
		if err := testutils.InsertSegment(t, jpgPath, seg.marker, seg.payload); err != nil {
			t.Fatalf("Setup failed: %v", err)
		}
	}
	if err := testutils.EmbedXMP(t, jpgPath, 3); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// All metadata segments of the original except MPF, whose offsets would be invalid after re-encoding
	want := metadataSegmentData(t, jpgPath)
	if len(want) != 6 {
		t.Fatalf("Setup created %d metadata segments, want 6", len(want))
	}
	want = append(want[:4], want[5])

	cfg := config.NewDefaultConfig().Process
	cfg.TargetMegapixels = 1.0
	if result, err := ResizeWithXMP(jpgPath, cfg); err != nil || !result.Resized {
		t.Fatalf("ResizeWithXMP() = %+v, %v, want resized", result, err)
	}

	got := metadataSegmentData(t, jpgPath)
	if len(got) != len(want) {
		t.Fatalf("Got %d metadata segments after resize, want %d (XMP, EXIF, ICC, extended XMP, comment)", len(got), len(want))
	}
	for i := range want {
		if !bytes.Equal(got[i], want[i]) {
			t.Errorf("Metadata segment %d differs after resize: got prefix %q, want %q", i, prefix(got[i]), prefix(want[i]))
		}
	}
}

// metadataSegmentData returns the data of all APPn and COM segments of a JPEG file in file order
func metadataSegmentData(t *testing.T, path string) [][]byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read image: %v", err)
	}
	segments, err := parseSegments(data)
	if err != nil {
		t.Fatalf("Failed to parse image: %v", err)
	}

	var result [][]byte
	for _, seg := range segments {
		if (seg.MarkerId >= 0xE0 && seg.MarkerId <= 0xEF) || seg.MarkerId == 0xFE {
			result = append(result, seg.Data)
		}
	}
	return result
}

func prefix(data []byte) []byte {
	if len(data) > 20 {
		return data[:20]
	}
	return data
}
