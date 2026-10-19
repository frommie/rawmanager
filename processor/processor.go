package processor

import (
	"errors"
	"fmt"
	"github.com/frommie/rawmanager/config"
	"github.com/frommie/rawmanager/counter"
	"github.com/frommie/rawmanager/jpeg"
	"github.com/frommie/rawmanager/xmp"
	"github.com/schollz/progressbar/v3"
	"os"
	"path/filepath"
	"strings"
)

type ImageProcessor struct {
	RootDir string
	Config  *config.Config
	Verbose bool
	counter *counter.FileCounter
	jpegBar *progressbar.ProgressBar
	rawBar  *progressbar.ProgressBar
}

func NewImageProcessor(rootDir string, cfg *config.Config, verbose bool) *ImageProcessor {
	return &ImageProcessor{
		RootDir: rootDir,
		Config:  cfg,
		Verbose: verbose,
	}
}

// Helper method for output
func (p *ImageProcessor) logf(format string, args ...interface{}) error {
	if p.Verbose {
		// Save position of both status bars
		p.jpegBar.Clear()
		p.rawBar.Clear()
		// Print message
		fmt.Printf(format, args...)
		// Restore status bars
		p.rawBar.RenderBlank()
		p.jpegBar.RenderBlank()
	}
	// Return error with message
	return fmt.Errorf(format, args...)
}

func (p *ImageProcessor) Process() error {
	// Count files
	p.counter = &counter.FileCounter{}
	if err := p.counter.CountFiles(p.RootDir, p.Config); err != nil {
		return err
	}

	// Initialize JPEG progress bar
	p.jpegBar = progressbar.NewOptions(p.counter.JpegCount,
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSetDescription("[cyan][1/2]Processing JPEGs..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	// Initialize RAW progress bar
	p.rawBar = progressbar.NewOptions(p.counter.RawCount,
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSetDescription("[cyan][2/2]Processing RAWs... "),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[yellow]=[reset]",
			SaucerHead:    "[yellow]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	// Start processing
	if err := p.Walk(); err != nil {
		return err
	}

	return nil
}

func (p *ImageProcessor) ProcessJPEG(jpgPath, rawPath string) error {
	// Check if JPEG exists
	if _, err := os.Stat(jpgPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Apply NoJpegAction
			if p.Config.NoJpegAction.DeleteRaw {
				p.logf("Deleting %s (no corresponding JPG file found)\n", rawPath)
				return os.Remove(rawPath)
			}
			return nil
		}
		return err
	}

	// Get rating from JPEG or XMP file
	rating, err := xmp.ReadRating(jpgPath, p.Config.Xmp.Mode)
	if err != nil {
		return fmt.Errorf("Error reading rating: %v", err)
	}

	// Get configured actions for this rating
	action, exists := p.Config.RatingActions[rating]
	if !exists {
		return p.logf("No action configured for rating %d", rating)
	}

	// Execute configured actions
	if action.DeleteRaw {
		p.logf("Deleting RAW %s (Rating %d)\n", rawPath, rating)
		if err := p.deleteFile(rawPath); err != nil {
			return err
		}
	}

	if action.DeleteJpeg {
		p.logf("Deleting JPEG %s (Rating %d)\n", jpgPath, rating)
		if err := p.deleteFile(jpgPath); err != nil {
			return err
		}
	}

	if action.CompressJpeg {
		p.logf("Compressing JPEG %s (Rating %d)\n", jpgPath, rating)
		result, err := jpeg.ResizeWithXMP(jpgPath, p.Config.Process)
		if err != nil {
			return err
		}
		if result.Resized {
			p.logf("Image %s resized to %dx%d pixels (EXIF and XMP data preserved)\n", jpgPath, result.Width, result.Height)
		} else {
			p.logf("Image %s is already small enough (%dx%d pixels)\n", jpgPath, result.Width, result.Height)
		}
	}

	return nil
}

func (p *ImageProcessor) deleteFile(path string) error {
	if err := os.Remove(path); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("Error deleting %s: %v", path, err)
		}
		p.logf("Warning: %s has already been deleted\n", path)
	}
	return nil
}

func (p *ImageProcessor) deleteFiles(paths ...string) error {
	for _, path := range paths {
		if err := p.deleteFile(path); err != nil {
			return err
		}
	}
	return nil
}

// ProcessDirectory only coordinates the process
func (p *ImageProcessor) ProcessDirectory(rawDir string, parentDir string) error {
	if err := p.validateDirectories(rawDir, parentDir); err != nil {
		return err
	}

	if err := p.processJpegFiles(rawDir, parentDir); err != nil {
		return err
	}

	if err := p.processRawFiles(rawDir, parentDir); err != nil {
		return err
	}

	return nil
}

// Help function for validating the directories
func (p *ImageProcessor) validateDirectories(rawDir string, parentDir string) error {
	if _, err := os.Stat(rawDir); err != nil {
		if os.IsNotExist(err) {
			p.logf("Info: Skipping non-existent directory: %s\n", rawDir)
			return nil
		}
		return fmt.Errorf("Error accessing directory %s: %v", rawDir, err)
	}
	return nil
}

// Processing JPEG files
func (p *ImageProcessor) processJpegFiles(rawDir string, parentDir string) error {
	jpegFiles, err := os.ReadDir(parentDir)
	if err != nil {
		if os.IsNotExist(err) {
			p.logf("Info: Skipping non-existent directory: %s\n", parentDir)
			return nil
		}
		return fmt.Errorf("Error reading JPEG directory %s: %v", parentDir, err)
	}

	for _, file := range jpegFiles {
		if err := p.processJpegFile(file, rawDir, parentDir); err != nil {
			p.logf("Warning: %v\n", err)
			continue
		}
	}
	return nil
}

// Processing of single JPEG file
func (p *ImageProcessor) processJpegFile(file os.DirEntry, rawDir string, parentDir string) error {
	if !file.IsDir() && p.Config.Files.IsJpeg(file.Name()) {
		p.jpegBar.Add(1)
		jpgPath := filepath.Join(parentDir, file.Name())
		rawPath := counterpartPath(rawDir, file.Name(), p.Config.Files.JpegExtension, p.Config.Files.RawExtension)

		if _, err := os.Stat(rawPath); err != nil && os.IsNotExist(err) {
			return fmt.Errorf("No RAW file found for: %s", jpgPath)
		}

		if err := p.ProcessJPEG(jpgPath, rawPath); err != nil {
			return fmt.Errorf("Error when processing %s: %v", jpgPath, err)
		}
	}
	return nil
}

// Processing RAW files
func (p *ImageProcessor) processRawFiles(rawDir string, parentDir string) error {
	rawFiles, err := os.ReadDir(rawDir)
	if err != nil {
		if os.IsNotExist(err) {
			p.logf("Info: Skipping non-existent directory: %s\n", rawDir)
			return nil
		}
		return fmt.Errorf("Error reading RAW directory %s: %v", rawDir, err)
	}

	for _, file := range rawFiles {
		if err := p.processRawFile(file, rawDir, parentDir); err != nil {
			p.logf("Warning: %v\n", err)
			continue
		}
	}
	return nil
}

// Processing of single RAW file
func (p *ImageProcessor) processRawFile(file os.DirEntry, rawDir string, parentDir string) error {
	if !file.IsDir() && p.Config.Files.IsRaw(file.Name()) {
		p.rawBar.Add(1)
		rawPath := filepath.Join(rawDir, file.Name())
		jpgPath := counterpartPath(parentDir, file.Name(), p.Config.Files.RawExtension, p.Config.Files.JpegExtension)

		if _, err := os.Stat(jpgPath); err != nil {
			if os.IsNotExist(err) {
				if !p.Config.NoJpegAction.DeleteRaw {
					return nil
				}
				if err := p.deleteFile(rawPath); err != nil {
					return fmt.Errorf("Error when deleting %s: %v", rawPath, err)
				}
				p.logf("Info: RAW file deleted (no JPG found): %s\n", rawPath)
			} else {
				return fmt.Errorf("Error when checking %s: %v", jpgPath, err)
			}
		}
	}
	return nil
}

// counterpartPath returns the path of the file in dir that belongs to name, with ext replaced by
// otherExt. As file systems can be case-sensitive, the configured, upper and lower case variants
// of otherExt are tried. If none exists, the path with the configured extension is returned.
func counterpartPath(dir, name, ext, otherExt string) string {
	base := name[:len(name)-len(ext)]
	for _, candidateExt := range []string{otherExt, strings.ToUpper(otherExt), strings.ToLower(otherExt)} {
		candidate := filepath.Join(dir, base+candidateExt)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return filepath.Join(dir, base+otherExt)
}

func (p *ImageProcessor) Walk() error {
	return filepath.Walk(p.RootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				p.logf("Info: Skip non-existing path: %s\n", path)
				return nil
			}
			p.logf("Warning: Error accessing %s: %v\n", path, err)
			return nil
		}

		// If RAWs are in same directory, process each directory
		if p.Config.Files.SameDir {
			if info.IsDir() && !strings.HasPrefix(info.Name(), ".") {
				if err := p.ProcessDirectory(path, path); err != nil {
					p.logf("Warning: Error processing %s: %v\n", path, err)
				}
			}
			return nil
		}

		// Otherwise only process RAW subdirectories
		if info.IsDir() && info.Name() == p.Config.Files.RawFolder {
			parentDir := filepath.Dir(path)
			if err := p.ProcessDirectory(path, parentDir); err != nil {
				p.logf("Warning: Error processing %s: %v\n", path, err)
			}
		}
		return nil
	})
}
