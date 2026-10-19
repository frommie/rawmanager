package processor

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/frommie/rawmanager/config"
	"github.com/frommie/rawmanager/testutils"
	"github.com/schollz/progressbar/v3"
)

func createTestConfig() *config.Config {
	return config.NewDefaultConfig()
}

// newTestProcessor creates a processor that writes its output into buffers instead of the terminal
func newTestProcessor(rootDir string, cfg *config.Config, verbose bool) (*ImageProcessor, *bytes.Buffer, *bytes.Buffer) {
	var out, errOut bytes.Buffer
	proc := NewImageProcessor(rootDir, cfg, verbose)
	proc.Out = &out
	proc.ErrOut = &errOut
	return proc, &out, &errOut
}

// createFile creates an empty file including its parent directories
func createFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	if err := os.WriteFile(path, []byte("RAW"), 0644); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
}

// Helper function to check file existence
func checkFileExists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func createTestFiles(t *testing.T, jpgPath, rawPath string, rating int) error {
	t.Helper()

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(jpgPath), 0755); err != nil {
		return fmt.Errorf("Error creating JPEG directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(rawPath), 0755); err != nil {
		return fmt.Errorf("Error creating RAW directory: %v", err)
	}

	// Create JPEG with rating in temporary directory
	if err := testutils.CreateTestJPEGWithEmbeddedXMP(t, jpgPath, rating); err != nil {
		return fmt.Errorf("Error creating test JPEG: %v", err)
	}

	// Create empty RAW file in temporary directory
	if err := os.WriteFile(rawPath, []byte("RAW"), 0644); err != nil {
		return fmt.Errorf("Error creating test RAW: %v", err)
	}

	return nil
}

func TestProcessJPEG(t *testing.T) {
	// Setup: Create temporary test directory
	tmpDir := t.TempDir()

	tests := []struct {
		name         string
		rating       int
		wantRawDel   bool
		wantJpgDel   bool
		wantCompress bool
	}{
		{
			name:         "Rating 1 deletes both files",
			rating:       1,
			wantRawDel:   true,
			wantJpgDel:   true,
			wantCompress: false,
		},
		{
			name:         "Rating 2 deletes RAW and compresses JPEG",
			rating:       2,
			wantRawDel:   true,
			wantJpgDel:   false,
			wantCompress: true,
		},
		// More test cases...
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test files in temporary directory
			jpgPath := filepath.Join(tmpDir, "test.JPG")
			rawDir := filepath.Join(tmpDir, "raw")
			rawPath := filepath.Join(rawDir, "test.RAF")

			// Ensure RAW directory exists
			if err := os.MkdirAll(filepath.Dir(rawPath), 0755); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			// Create test files
			if err := createTestFiles(t, jpgPath, rawPath, tt.rating); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			proc, _, _ := newTestProcessor(tmpDir, createTestConfig(), true)

			if err := proc.processJPEG(jpgPath, rawPath); err != nil {
				t.Errorf("processJPEG() error = %v", err)
			}

			// Check results with relative paths
			if exists := checkFileExists(t, rawPath); exists != !tt.wantRawDel {
				t.Errorf("RAW file exists = %v, want %v", exists, !tt.wantRawDel)
			}
			if exists := checkFileExists(t, jpgPath); exists != !tt.wantJpgDel {
				t.Errorf("JPEG file exists = %v, want %v", exists, !tt.wantJpgDel)
			}
		})
	}
}

func TestImageProcessor_Process(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(dir string) error
		wantErr   bool
	}{
		{
			name: "Process with ratings",
			setupFunc: func(dir string) error {
				// Create test files in the transferred temporary directory
				files := map[string]struct {
					name   string
					rating int
				}{
					"test1": {"test1", 1},
					"test2": {"test2", 2},
					"test3": {"test3", 3},
				}

				for _, f := range files {
					jpgPath := filepath.Join(dir, f.name+".JPG")
					rawPath := filepath.Join(dir, "raw", f.name+".RAF")

					if err := os.MkdirAll(filepath.Dir(rawPath), 0755); err != nil {
						return err
					}

					if err := createTestFiles(t, jpgPath, rawPath, f.rating); err != nil {
						return err
					}
				}
				return nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			if err := tt.setupFunc(tmpDir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			proc, _, _ := newTestProcessor(tmpDir, config.NewDefaultConfig(), true)

			if err := proc.Process(); err != nil != tt.wantErr {
				t.Errorf("Process() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProcess_NoJpegAction(t *testing.T) {
	tests := []struct {
		name       string
		deleteRaw  bool
		wantRawDel bool
	}{
		{"Orphaned RAW is deleted when configured", true, true},
		{"Orphaned RAW is kept when configured", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			rawPath := filepath.Join(tmpDir, "raw", "orphan.RAF")
			if err := os.MkdirAll(filepath.Dir(rawPath), 0755); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
			if err := os.WriteFile(rawPath, []byte("RAW"), 0644); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			cfg := config.NewDefaultConfig()
			cfg.NoJpegAction.DeleteRaw = tt.deleteRaw

			proc, _, _ := newTestProcessor(tmpDir, cfg, false)
			if err := proc.Process(); err != nil {
				t.Fatalf("Process() error = %v", err)
			}

			if exists := checkFileExists(t, rawPath); exists != !tt.wantRawDel {
				t.Errorf("RAW file exists = %v, want %v", exists, !tt.wantRawDel)
			}
		})
	}
}

func TestProcess_IgnoresExtensionCase(t *testing.T) {
	tmpDir := t.TempDir()
	jpgPath := filepath.Join(tmpDir, "test.JPG")
	rawPath := filepath.Join(tmpDir, "raw", "test.RAF")
	if err := createTestFiles(t, jpgPath, rawPath, 1); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	cfg := config.NewDefaultConfig()
	cfg.Files.JpegExtension = ".jpg"
	cfg.Files.RawExtension = ".raf"

	proc, _, _ := newTestProcessor(tmpDir, cfg, false)
	if err := proc.Process(); err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Rating 1 deletes both files
	if checkFileExists(t, jpgPath) {
		t.Errorf("JPEG file still exists, want deleted")
	}
	if checkFileExists(t, rawPath) {
		t.Errorf("RAW file still exists, want deleted")
	}
}

func TestProgressBars(t *testing.T) {
	tmpDir := t.TempDir()
	for name, rating := range map[string]int{"img1": 1, "img2": 2, "img3": 3} {
		jpgPath := filepath.Join(tmpDir, name+".JPG")
		rawPath := filepath.Join(tmpDir, "raw", name+".RAF")
		if err := createTestFiles(t, jpgPath, rawPath, rating); err != nil {
			t.Fatalf("Failed to create test files: %v", err)
		}
	}

	proc, _, _ := newTestProcessor(tmpDir, config.NewDefaultConfig(), false)
	if err := proc.Process(); err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if proc.jpegCount != 3 {
		t.Errorf("Expected 3 JPEGs, got %d", proc.jpegCount)
	}
	if proc.rawCount != 3 {
		t.Errorf("Expected 3 RAWs, got %d", proc.rawCount)
	}
}

func TestProgressBars_CountOnlyProcessedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	if err := createTestFiles(t, filepath.Join(tmpDir, "img1.JPG"), filepath.Join(tmpDir, "raw", "img1.RAF"), 3); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	// Directory without RAW folder is not processed
	createFile(t, filepath.Join(tmpDir, "export", "img2.JPG"))
	createFile(t, filepath.Join(tmpDir, "export", "img2.RAF"))

	proc, _, _ := newTestProcessor(tmpDir, config.NewDefaultConfig(), false)
	if err := proc.Process(); err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if proc.jpegCount != 1 || proc.rawCount != 1 {
		t.Errorf("Counted %d JPEGs and %d RAWs, want 1 and 1", proc.jpegCount, proc.rawCount)
	}
}

func TestProcess_SkipsHiddenDirectories(t *testing.T) {
	tests := []struct {
		name    string
		sameDir bool
		rawPath string
	}{
		{"RAW folder below hidden directory", false, filepath.Join(".hidden", "shoot", "raw", "orphan.RAF")},
		{"Same directory below hidden directory", true, filepath.Join(".hidden", "shoot", "orphan.RAF")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			rawPath := filepath.Join(tmpDir, tt.rawPath)
			createFile(t, rawPath)

			cfg := config.NewDefaultConfig()
			cfg.Files.SameDir = tt.sameDir

			proc, _, _ := newTestProcessor(tmpDir, cfg, false)
			if err := proc.Process(); err != nil {
				t.Fatalf("Process() error = %v", err)
			}

			if !checkFileExists(t, rawPath) {
				t.Error("RAW file in hidden directory was deleted")
			}
		})
	}
}

func TestProcess_SameDirWithDotAsRoot(t *testing.T) {
	tmpDir := t.TempDir()
	rawPath := filepath.Join(tmpDir, "orphan.RAF")
	createFile(t, rawPath)
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	t.Cleanup(func() { os.Chdir(oldWd) })

	cfg := config.NewDefaultConfig()
	cfg.Files.SameDir = true

	proc, _, _ := newTestProcessor(".", cfg, false)
	if err := proc.Process(); err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if checkFileExists(t, rawPath) {
		t.Error("Orphaned RAW in root directory was not processed")
	}
}

func TestLogging_InfoOnlyInVerboseMode(t *testing.T) {
	for _, verbose := range []bool{true, false} {
		proc, out, _ := newTestProcessor(t.TempDir(), config.NewDefaultConfig(), verbose)
		proc.infof("Test message %d\n", 42)

		if got := strings.Contains(out.String(), "Test message 42"); got != verbose {
			t.Errorf("verbose=%v: message shown = %v, want %v", verbose, got, verbose)
		}
	}
}

func TestLogging_WarningsAlwaysShown(t *testing.T) {
	tmpDir := t.TempDir()
	jpgPath := filepath.Join(tmpDir, "corrupt.JPG")
	if err := os.WriteFile(jpgPath, []byte("not a jpeg"), 0644); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	createFile(t, filepath.Join(tmpDir, "raw", "corrupt.RAF"))

	proc, _, errOut := newTestProcessor(tmpDir, config.NewDefaultConfig(), false)
	if err := proc.Process(); err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if !strings.Contains(errOut.String(), jpgPath) {
		t.Errorf("Warning for unreadable JPEG not shown, got %q", errOut.String())
	}
}

func TestLogging_UnratedJpegIsNoWarning(t *testing.T) {
	tmpDir := t.TempDir()
	jpgPath := filepath.Join(tmpDir, "unrated.JPG")
	rawPath := filepath.Join(tmpDir, "raw", "unrated.RAF")
	if err := testutils.CreateEmptyJPEG(t, jpgPath); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	createFile(t, rawPath)

	proc, out, errOut := newTestProcessor(tmpDir, config.NewDefaultConfig(), true)
	if err := proc.Process(); err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if errOut.Len() != 0 {
		t.Errorf("Unexpected warning for unrated JPEG: %q", errOut.String())
	}
	if !strings.Contains(out.String(), "no rating") {
		t.Errorf("Info for unrated JPEG not shown, got %q", out.String())
	}
	if !checkFileExists(t, jpgPath) || !checkFileExists(t, rawPath) {
		t.Error("Unrated files were deleted")
	}
}

func TestProgressBars_CompleteWhenRawsDeletedByRating(t *testing.T) {
	tmpDir := t.TempDir()
	// Rating 1 deletes the RAW while processing the JPEGs
	if err := createTestFiles(t, filepath.Join(tmpDir, "img1.JPG"), filepath.Join(tmpDir, "raw", "img1.RAF"), 1); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	if err := createTestFiles(t, filepath.Join(tmpDir, "img3.JPG"), filepath.Join(tmpDir, "raw", "img3.RAF"), 3); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	proc, _, _ := newTestProcessor(tmpDir, config.NewDefaultConfig(), false)
	if err := proc.Process(); err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	for name, bar := range map[string]interface{ State() progressbar.State }{"JPEG": proc.jpegBar, "RAW": proc.rawBar} {
		if state := bar.State(); state.CurrentNum != state.Max {
			t.Errorf("%s progress bar at %d/%d, want complete", name, state.CurrentNum, state.Max)
		}
	}
}
