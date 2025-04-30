package jpeg

import (
    "bytes"
    "encoding/xml"
    "fmt"
    "os"
    "github.com/dsoprea/go-jpeg-image-structure/v2"
    "github.com/disintegration/imaging"
    "math"
    "github.com/frommie/rawmanager/types"
    "github.com/frommie/rawmanager/constants"
    "strings"
)

func GetRating(jpgPath string) (int, error) {
    data, err := os.ReadFile(jpgPath)
    if err != nil {
        return 0, fmt.Errorf("Fehler beim Lesen der Datei: %v", err)
    }

    jmp := jpegstructure.NewJpegMediaParser()
    intfc, err := jmp.ParseBytes(data)
    if err != nil {
        return 0, fmt.Errorf("Fehler beim Parsen der JPEG-Datei: %v", err)
    }

    sl := intfc.(*jpegstructure.SegmentList)
    var xmpData []byte
    for _, segment := range sl.Segments() {
        if segment.MarkerId == constants.App1MarkerId {
            payload := segment.Data
            if bytes.HasPrefix(payload, []byte(constants.XmpNamespace)) {
                cleanData := bytes.Map(func(r rune) rune {
                    if r == 0 {
                        return -1
                    }
                    return r
                }, payload[len(constants.XmpNamespace):])
                
                xmpData = bytes.TrimSpace(cleanData)
                break
            }
        }
    }

    if xmpData == nil {
        return 0, fmt.Errorf("Keine XMP-Daten gefunden")
    }

    var xmp types.XmpMeta
    if err := xml.Unmarshal(xmpData, &xmp); err != nil {
        return 0, fmt.Errorf("Fehler beim Parsen der XMP-Daten: %v", err)
    }

    rating := 0
    if xmp.RDF.Description.Rating != "" {
        fmt.Sscanf(xmp.RDF.Description.Rating, "%d", &rating)
    } else if xmp.RDF.Description.MSRating != "" {
        var msRating int
        fmt.Sscanf(xmp.RDF.Description.MSRating, "%d", &msRating)
        rating = (msRating + 24) / 25
    }
    
    return rating, nil
}

func ResizeWithXMP(jpgPath string) error {
    // Lese zunächst die XMP-Daten aus dem Original
    data, err := os.ReadFile(jpgPath)
    if err != nil {
        return fmt.Errorf("Fehler beim Lesen der Datei: %v", err)
    }

    // Extrahiere XMP-Segment
    jmp := jpegstructure.NewJpegMediaParser()
    intfc, err := jmp.ParseBytes(data)
    if err != nil {
        return fmt.Errorf("Fehler beim Parsen der JPEG-Datei: %v", err)
    }

    sl := intfc.(*jpegstructure.SegmentList)
    var xmpSegment *jpegstructure.Segment
    for _, segment := range sl.Segments() {
        if segment.MarkerId == constants.App1MarkerId && bytes.HasPrefix(segment.Data, []byte(constants.XmpNamespace)) {
            xmpSegment = segment
            break
        }
    }

    // Verkleinere das Bild
    img, err := imaging.Open(jpgPath)
    if err != nil {
        return fmt.Errorf("Fehler beim Öffnen des Bildes: %v", err)
    }

    bounds := img.Bounds()
    currentWidth := bounds.Max.X
    currentHeight := bounds.Max.Y
    currentMP := float64(currentWidth * currentHeight) / 1000000.0
    
    if currentMP <= constants.TargetMegapixels {
        return nil // Bild ist bereits klein genug
    }

    ratio := math.Sqrt(constants.TargetMegapixels / currentMP)
    newWidth := int(float64(currentWidth) * ratio)
    newHeight := int(float64(currentHeight) * ratio)

    resized := imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)

    // Speichere temporär das verkleinerte Bild
    tempPath := strings.TrimSuffix(jpgPath, constants.JpegExtension) + "_temp.jpg"
    if err := imaging.Save(resized, tempPath, imaging.JPEGQuality(constants.JpegQuality)); err != nil {
        return fmt.Errorf("Fehler beim Speichern des temporären Bildes: %v", err)
    }
    defer os.Remove(tempPath)

    // Füge XMP-Segment wieder ein
    if xmpSegment != nil {
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
        
        insertPos := 1
        for i, seg := range segments {
            if seg.MarkerId == 0xE0 { // APP0
                insertPos = i + 1
                break
            }
        }
        
        var newSegments []*jpegstructure.Segment
        newSegments = append(newSegments, segments[:insertPos]...)
        newSegments = append(newSegments, xmpSegment)
        newSegments = append(newSegments, segments[insertPos:]...)
        
        newJpeg := jpegstructure.NewSegmentList(newSegments)

        var buffer bytes.Buffer
        if err := newJpeg.Write(&buffer); err != nil {
            return fmt.Errorf("Fehler beim Serialisieren der JPEG-Daten: %v", err)
        }

        if err := os.WriteFile(jpgPath, buffer.Bytes(), 0644); err != nil {
            return fmt.Errorf("Fehler beim Speichern der finalen JPEG: %v", err)
        }
    }

    fmt.Printf("Bild %s auf %dx%d Pixel verkleinert (XMP-Daten erhalten)\n", 
        jpgPath, newWidth, newHeight)
    return nil
}