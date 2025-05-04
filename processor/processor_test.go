package processor

import (
    "bytes"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"
    "testing"

    "github.com/frommie/rawmanager/config"
    "github.com/frommie/rawmanager/counter"
    "github.com/frommie/rawmanager/testutils"
    "github.com/schollz/progressbar/v3"
)

func createTestConfig() *config.Config {
    return config.NewDefaultConfig()
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
            wantRawDel:  true,
            wantJpgDel:  true,
            wantCompress: false,
        },
        {
            name:        "Rating 2 deletes RAW and compresses JPEG",
            rating:      2,
            wantRawDel:  true,
            wantJpgDel:  false,
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

            proc := &ImageProcessor{
                RootDir: tmpDir,
                Config:  createTestConfig(),
                Verbose: true,
            }

            // Initialize progress bars
            proc.jpegBar = progressbar.NewOptions(1)
            proc.rawBar = progressbar.NewOptions(1)

            if err := proc.ProcessJPEG(jpgPath, rawPath); err != nil {
                t.Errorf("ProcessJPEG() error = %v", err)
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

            proc := &ImageProcessor{
                RootDir:  tmpDir,
                Config:   config.NewDefaultConfig(),
                Verbose: true,
            }

            if err := proc.Process(); err != nil != tt.wantErr {
                t.Errorf("Process() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}

func TestProgressBars(t *testing.T) {
    // Setup: Create temporary test directory
    tmpDir := t.TempDir()
    
    // Setup test files with relative names
    files := map[string]int{
        "img1": 1,
        "img2": 2,
        "img3": 3,
    }

    // Create test files
    for name, rating := range files {
        jpgPath := filepath.Join(tmpDir, name+".JPG")
        rawDir := filepath.Join(tmpDir, "raw")
        rawPath := filepath.Join(rawDir, name+".RAF")

        if err := createTestFiles(t, jpgPath, rawPath, rating); err != nil {
            t.Fatalf("Failed to create test files: %v", err)
        }
    }
    
    // Initialize Processor with temporary directory
    proc := &ImageProcessor{
        RootDir: tmpDir,
        Config:  config.NewDefaultConfig(),
        Verbose: false,
    }

    // Process files
    if err := proc.Process(); err != nil {
        t.Fatalf("Process() error = %v", err)
    }

    // Check counter
    if proc.counter.JpegCount != 3 {
        t.Errorf("Expected 3 JPEGs, got %d", proc.counter.JpegCount)
    }
    if proc.counter.RawCount != 3 {
        t.Errorf("Expected 3 RAWs, got %d", proc.counter.RawCount)
    }
}

func TestVerboseLogging(t *testing.T) {
    proc := &ImageProcessor{
        RootDir: t.TempDir(),
        Config:  config.NewDefaultConfig(),
        Verbose: true,
    }

    // Initialize progress bars before testing logging
    proc.counter = &counter.FileCounter{}
    proc.jpegBar = progressbar.NewOptions(0)
    proc.rawBar = progressbar.NewOptions(0)

    old := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    proc.logf("Test message %d", 42)

    w.Close()
    os.Stdout = old

    var buf bytes.Buffer
    io.Copy(&buf, r)

    if !strings.Contains(buf.String(), "Test message 42") {
        t.Error("Expected log message not found in output")
    }
}

func TestEnglishMessages(t *testing.T) {
    proc := &ImageProcessor{
        RootDir: t.TempDir(),
        Config:  config.NewDefaultConfig(),
        Verbose: true,
    }

    // Initialize progress bars
    proc.jpegBar = progressbar.NewOptions(1)
    proc.rawBar = progressbar.NewOptions(1)

    // Capture stdout
    old := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    proc.logf("Test message %d", 42)

    w.Close()
    os.Stdout = old

    var buf bytes.Buffer
    io.Copy(&buf, r)

    if !strings.Contains(buf.String(), "Test message 42") {
        t.Error("Expected log message not found in output")
    }
}