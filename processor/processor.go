package processor

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/frommie/rawmanager/config"
	"github.com/frommie/rawmanager/jpeg"
	"github.com/frommie/rawmanager/xmp"
	"github.com/schollz/progressbar/v3"
)

type ImageProcessor struct {
	RootDir string
	Config  *config.Config
	Verbose bool
	DryRun  bool      // only report the actions without changing any files
	Out     io.Writer // progress bars and info messages
	ErrOut  io.Writer // warnings

	jpegCount int
	rawCount  int
	jpegBar   *progressbar.ProgressBar
	rawBar    *progressbar.ProgressBar
}

// directoryPair is a directory with JPEG files and the directory containing the corresponding RAW files
type directoryPair struct {
	jpegDir string
	rawDir  string
}

func NewImageProcessor(rootDir string, cfg *config.Config, verbose bool) *ImageProcessor {
	return &ImageProcessor{
		RootDir: rootDir,
		Config:  cfg,
		Verbose: verbose,
		Out:     os.Stdout,
		ErrOut:  os.Stderr,
	}
}

// infof prints a message in verbose mode
func (p *ImageProcessor) infof(format string, args ...interface{}) {
	if p.Verbose {
		p.printf(p.Out, format, args...)
	}
}

// warnf always prints a message
func (p *ImageProcessor) warnf(format string, args ...interface{}) {
	p.printf(p.ErrOut, "Warning: "+format, args...)
}

// perform executes an action, or only reports it in dry-run mode.
// The description names the action, e.g. "delete RAW <path>".
func (p *ImageProcessor) perform(description string, action func() error) error {
	if p.DryRun {
		p.printf(p.Out, "[dry-run] Would %s\n", description)
		return nil
	}
	p.infof("%s%s\n", strings.ToUpper(description[:1]), description[1:])
	return action()
}

// printf prints a message without breaking the progress bars
func (p *ImageProcessor) printf(w io.Writer, format string, args ...interface{}) {
	if p.jpegBar != nil {
		p.jpegBar.Clear()
		p.rawBar.Clear()
	}
	fmt.Fprintf(w, format, args...)
	if p.jpegBar != nil {
		p.rawBar.RenderBlank()
		p.jpegBar.RenderBlank()
	}
}

func (p *ImageProcessor) Process() error {
	dirs, err := p.findDirectories()
	if err != nil {
		return err
	}

	p.jpegCount, p.rawCount = p.countFiles(dirs)
	p.jpegBar = p.newProgressBar(p.jpegCount, "[cyan][1/2]Processing JPEGs...", "green")
	p.rawBar = p.newProgressBar(p.rawCount, "[cyan][2/2]Processing RAWs... ", "yellow")

	for _, dir := range dirs {
		p.processDirectory(dir)
	}
	return nil
}

func (p *ImageProcessor) newProgressBar(count int, description, color string) *progressbar.ProgressBar {
	return progressbar.NewOptions(count,
		progressbar.OptionSetWriter(p.Out),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[" + color + "]=[reset]",
			SaucerHead:    "[" + color + "]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
}

// findDirectories returns all directories to process. Hidden directories are skipped.
func (p *ImageProcessor) findDirectories() ([]directoryPair, error) {
	var dirs []directoryPair
	err := filepath.WalkDir(p.RootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			p.warnf("Error accessing %s: %v\n", path, err)
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if path != p.RootDir && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}

		if p.Config.Files.SameDir {
			// RAWs are in the same directory as the JPEGs
			dirs = append(dirs, directoryPair{jpegDir: path, rawDir: path})
		} else if path != p.RootDir && d.Name() == p.Config.Files.RawFolder {
			// RAWs are in a subdirectory of the JPEG directory
			dirs = append(dirs, directoryPair{jpegDir: filepath.Dir(path), rawDir: path})
		}
		return nil
	})
	return dirs, err
}

func (p *ImageProcessor) countFiles(dirs []directoryPair) (jpegCount, rawCount int) {
	for _, dir := range dirs {
		jpegCount += len(p.listFiles(dir.jpegDir, p.Config.Files.IsJpeg))
		rawCount += len(p.listFiles(dir.rawDir, p.Config.Files.IsRaw))
	}
	return jpegCount, rawCount
}

// listFiles returns the names of all files in dir that match
func (p *ImageProcessor) listFiles(dir string, match func(name string) bool) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		p.warnf("Error reading directory %s: %v\n", dir, err)
		return nil
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && match(entry.Name()) {
			names = append(names, entry.Name())
		}
	}
	return names
}

// processDirectory processes the JPEGs first and then applies the NoJpegAction to RAWs without JPEG
func (p *ImageProcessor) processDirectory(dir directoryPair) {
	// List RAWs before processing the JPEGs, which may delete some of them,
	// so that the progress bar matches the counted files
	rawNames := p.listFiles(dir.rawDir, p.Config.Files.IsRaw)

	for _, name := range p.listFiles(dir.jpegDir, p.Config.Files.IsJpeg) {
		p.jpegBar.Add(1)
		if err := p.processJpegFile(dir, name); err != nil {
			p.warnf("%v\n", err)
		}
	}

	for _, name := range rawNames {
		p.rawBar.Add(1)
		if err := p.processRawFile(dir, name); err != nil {
			p.warnf("%v\n", err)
		}
	}
}

func (p *ImageProcessor) processJpegFile(dir directoryPair, name string) error {
	jpgPath := filepath.Join(dir.jpegDir, name)
	rawPath := counterpartPath(dir.rawDir, name, p.Config.Files.JpegExtension, p.Config.Files.RawExtension)

	if _, err := os.Stat(rawPath); errors.Is(err, os.ErrNotExist) {
		p.infof("Skipping %s (no RAW file found)\n", jpgPath)
		return nil
	}

	err := p.processJPEG(jpgPath, rawPath)
	if errors.Is(err, xmp.ErrNoRating) {
		p.infof("Skipping %s (no rating)\n", jpgPath)
		return nil
	}
	if err != nil {
		return fmt.Errorf("error processing %s: %w", jpgPath, err)
	}
	return nil
}

// processJPEG executes the actions configured for the rating of the JPEG
func (p *ImageProcessor) processJPEG(jpgPath, rawPath string) error {
	rating, err := xmp.ReadRating(jpgPath, p.Config.Xmp.Mode)
	if err != nil {
		return fmt.Errorf("error reading rating: %w", err)
	}

	action, exists := p.Config.RatingActions[rating]
	if !exists {
		p.infof("Skipping %s (no action configured for rating %d)\n", jpgPath, rating)
		return nil
	}

	if action.DeleteRaw {
		description := fmt.Sprintf("delete RAW %s (Rating %d)", rawPath, rating)
		if err := p.perform(description, func() error { return p.deleteFile(rawPath) }); err != nil {
			return err
		}
	}

	if action.DeleteJpeg {
		description := fmt.Sprintf("delete JPEG %s (Rating %d)", jpgPath, rating)
		if err := p.perform(description, func() error { return p.deleteFile(jpgPath) }); err != nil {
			return err
		}
	}

	if action.CompressJpeg {
		description := fmt.Sprintf("compress JPEG %s (Rating %d)", jpgPath, rating)
		if err := p.perform(description, func() error { return p.compressJpeg(jpgPath) }); err != nil {
			return err
		}
	}

	return nil
}

func (p *ImageProcessor) compressJpeg(jpgPath string) error {
	result, err := jpeg.ResizeWithXMP(jpgPath, p.Config.Process)
	if err != nil {
		return err
	}
	if result.Resized {
		p.infof("Image %s resized to %dx%d pixels (EXIF and XMP data preserved)\n", jpgPath, result.Width, result.Height)
	} else {
		p.infof("Image %s is already small enough (%dx%d pixels)\n", jpgPath, result.Width, result.Height)
	}
	return nil
}

// processRawFile applies the NoJpegAction to a RAW file without JPEG
func (p *ImageProcessor) processRawFile(dir directoryPair, name string) error {
	rawPath := filepath.Join(dir.rawDir, name)
	if _, err := os.Stat(rawPath); errors.Is(err, os.ErrNotExist) {
		// Already deleted because of the rating of its JPEG
		return nil
	}
	jpgPath := counterpartPath(dir.jpegDir, name, p.Config.Files.RawExtension, p.Config.Files.JpegExtension)

	_, err := os.Stat(jpgPath)
	if err == nil {
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("error checking %s: %w", jpgPath, err)
	}

	if p.Config.NoJpegAction.DeleteRaw {
		description := fmt.Sprintf("delete RAW %s (no JPEG found)", rawPath)
		return p.perform(description, func() error { return p.deleteFile(rawPath) })
	}
	return nil
}

func (p *ImageProcessor) deleteFile(path string) error {
	if err := os.Remove(path); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("error deleting %s: %w", path, err)
		}
		p.warnf("%s has already been deleted\n", path)
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
