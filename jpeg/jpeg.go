package jpeg

import (
    "bytes"
    "fmt"
    "os"
    "github.com/dsoprea/go-jpeg-image-structure/v2"
    "github.com/disintegration/imaging"
    "math"
    "strings"
    "github.com/frommie/rawmanager/config"
    "github.com/frommie/rawmanager/xmp"
    "path/filepath"
)

const (
    // XmpNamespace defines the namespace for Adobe XMP metadata
    xmpNamespace = "http://ns.adobe.com/xap/1.0/"
    
    // app1MarkerId is the marker for APP1 segments in JPEG files (0xE1)
    app1MarkerId = 0xE1
)

// GetRatingFromFile reads the rating from a JPEG file
func GetRatingFromFile(jpgPath string, cfg *config.Config) (int, error) {
    switch cfg.Xmp.Mode {
    case config.XmpModeEmbedded:
        // Read embedded XMP data from JPEG
        file, err := os.Open(jpgPath)
        if err != nil {
            return 0, err
        }
        defer file.Close()

        xmpData, err := xmp.ExtractXmpData(file)
        if err != nil {
            return 0, err
        }

        return xmp.GetRating(xmpData)

    case config.XmpModeSeparate:
        // Read separate .xmp file
        xmpPath := jpgPath[:len(jpgPath)-len(filepath.Ext(jpgPath))] + ".xmp"
        return xmp.GetRatingFromFile(xmpPath)

    case config.XmpModeSeparateExt:
        // Read .jpg.xmp file
        xmpPath := jpgPath + ".xmp"
        return xmp.GetRatingFromFile(xmpPath)

    default:
        return 0, fmt.Errorf("Invalid XMP mode: %s", cfg.Xmp.Mode)
    }
}

func ResizeWithXMP(jpgPath string, config *config.Config, verbose bool) error {
    // Read original file
    data, err := os.ReadFile(jpgPath)
    if err != nil {
        return fmt.Errorf("Fehler beim Lesen der Datei: %v", err)
    }

    // Extract EXIF and XMP
    jmp := jpegstructure.NewJpegMediaParser()
    intfc, err := jmp.ParseBytes(data)
    if err != nil {
        return fmt.Errorf("Fehler beim Parsen der JPEG-Datei: %v", err)
    }

    sl := intfc.(*jpegstructure.SegmentList)
    
    // Save EXIF segment
    var exifSegment *jpegstructure.Segment
    var xmpSegment *jpegstructure.Segment
    
    for _, segment := range sl.Segments() {
        // Find XMP
        if segment.MarkerId == app1MarkerId && bytes.HasPrefix(segment.Data, []byte(xmpNamespace)) {
            xmpSegment = segment
        }
        // Find EXIF (0xE1 is the marker for both EXIF and XMP)
        if segment.MarkerId == app1MarkerId && !bytes.HasPrefix(segment.Data, []byte(xmpNamespace)) {
            exifSegment = segment
        }
    }

    // Resize image
    img, err := imaging.Open(jpgPath)
    if err != nil {
        return fmt.Errorf("Fehler beim Öffnen des Bildes: %v", err)
    }

    bounds := img.Bounds()
    currentWidth := bounds.Max.X
    currentHeight := bounds.Max.Y
    currentMP := float64(currentWidth * currentHeight) / 1000000.0
    
    if currentMP <= config.Process.TargetMegapixels {
        if verbose {
            fmt.Printf("Image %s is already small enough (%.1f MP)\n", 
                jpgPath, currentMP)
        }
        return nil
    }

    ratio := math.Sqrt(config.Process.TargetMegapixels / currentMP)
    newWidth := int(float64(currentWidth) * ratio)
    newHeight := int(float64(currentHeight) * ratio)

    resized := imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)

    // Temporarily save resized image
    tempPath := strings.TrimSuffix(jpgPath, config.Files.JpegExtension) + "_temp.jpg"
    if err := imaging.Save(resized, tempPath, imaging.JPEGQuality(config.Process.JpegQuality)); err != nil {
        return fmt.Errorf("Fehler beim Speichern des temporären Bildes: %v", err)
    }
    defer os.Remove(tempPath)

    // Read temporary file
    newData, err := os.ReadFile(tempPath)
    if err != nil {
        return fmt.Errorf("Fehler beim Lesen des temporären Bildes: %v", err)
    }

    newIntfc, err := jmp.ParseBytes(newData)
    if err != nil {
        return fmt.Errorf("Fehler beim Parsen des temporären Bildes: %v", err)
    }

    newSl := newIntfc.(*jpegstructure.SegmentList)
    segments := newSl.Segments()

    // Insert EXIF and XMP at the correct position
    insertPos := 1
    for i, seg := range segments {
        if seg.MarkerId == 0xE0 { // APP0
            insertPos = i + 1
            break
        }
    }

    // Create new segment list
    var newSegments []*jpegstructure.Segment
    newSegments = append(newSegments, segments[:insertPos]...)
    
    // Add EXIF first
    if exifSegment != nil {
        newSegments = append(newSegments, exifSegment)
    }
    
    // Then XMP
    if xmpSegment != nil {
        newSegments = append(newSegments, xmpSegment)
    }
    
    // Append remaining segments
    newSegments = append(newSegments, segments[insertPos:]...)
    
    newJpeg := jpegstructure.NewSegmentList(newSegments)

    // Write final file
    var buffer bytes.Buffer
    if err := newJpeg.Write(&buffer); err != nil {
        return fmt.Errorf("Fehler beim Serialisieren der JPEG-Daten: %v", err)
    }

    if err := os.WriteFile(jpgPath, buffer.Bytes(), 0644); err != nil {
        return fmt.Errorf("Fehler beim Speichern der finalen JPEG: %v", err)
    }

    if verbose {
        fmt.Printf("Image %s resized to %dx%d pixels (EXIF and XMP data preserved)\n", 
            jpgPath, newWidth, newHeight)
    }
    return nil
}