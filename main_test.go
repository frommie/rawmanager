package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/frommie/rawmanager/testutils"
)

// createRatedPair creates a JPEG with the given rating in dir and the RAW file in dir/raw
func createRatedPair(t *testing.T, dir, name string, rating int) (jpgPath, rawPath string) {
	t.Helper()
	jpgPath = filepath.Join(dir, name+".JPG")
	rawPath = filepath.Join(dir, "raw", name+".RAF")
	if err := os.MkdirAll(filepath.Dir(rawPath), 0755); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	if err := testutils.CreateTestJPEGWithEmbeddedXMP(t, jpgPath, rating); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	if err := os.WriteFile(rawPath, []byte("RAW"), 0644); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	return jpgPath, rawPath
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestRun_ProcessesDirectoryArgument(t *testing.T) {
	dir := t.TempDir()
	jpgPath, rawPath := createRatedPair(t, dir, "img", 1)

	if err := run([]string{dir}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if fileExists(jpgPath) || fileExists(rawPath) {
		t.Error("Rating 1 files were not deleted")
	}
}

func TestRun_DefaultsToCurrentDirectory(t *testing.T) {
	dir := t.TempDir()
	jpgPath, rawPath := createRatedPair(t, dir, "img", 1)

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	t.Cleanup(func() { os.Chdir(oldWd) })

	if err := run(nil, io.Discard, io.Discard); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if fileExists(jpgPath) || fileExists(rawPath) {
		t.Error("Rating 1 files in current directory were not deleted")
	}
}

func TestRun_UsesConfigFile(t *testing.T) {
	dir := t.TempDir()
	jpgPath, rawPath := createRatedPair(t, dir, "img", 1)

	// Rating 1 keeps both files with this configuration
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("ratingActions:\n  1:\n    deleteRaw: false\n"), 0644); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	if err := run([]string{"-config", configPath, dir}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if !fileExists(jpgPath) || !fileExists(rawPath) {
		t.Error("Files were deleted although the configuration keeps them")
	}
}

func TestRun_Errors(t *testing.T) {
	invalidConfig := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(invalidConfig, []byte("xmp:\n  mode: invalid\n"), 0644); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	tests := []struct {
		name string
		args []string
	}{
		{"Missing config file", []string{"-config", filepath.Join(t.TempDir(), "missing.yaml"), t.TempDir()}},
		{"Invalid config file", []string{"-config", invalidConfig, t.TempDir()}},
		{"Unknown flag", []string{"-unknown"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := run(tt.args, io.Discard, io.Discard); err == nil {
				t.Error("run() error = nil, want error")
			}
		})
	}
}

func TestRun_HelpIsNoError(t *testing.T) {
	if err := run([]string{"-h"}, io.Discard, io.Discard); err != nil {
		t.Errorf("run() error = %v, want nil", err)
	}
}

func TestRun_DryRun(t *testing.T) {
	dir := t.TempDir()
	jpgPath, rawPath := createRatedPair(t, dir, "img", 1)

	if err := run([]string{"-dry-run", dir}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if !fileExists(jpgPath) || !fileExists(rawPath) {
		t.Error("Files were deleted in dry run")
	}
}
