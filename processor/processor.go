package processor

import (
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "github.com/frommie/rawmanager/jpeg"
    "github.com/frommie/rawmanager/constants"
)

type ImageProcessor struct {
    RootDir string
}

func (p *ImageProcessor) ProcessJPEG(jpgPath, rawPath string) error {
    rating, err := jpeg.GetRating(jpgPath)
    if err != nil {
        fmt.Printf("Warnung: Konnte Bewertung für %s nicht lesen: %v\n", jpgPath, err)
        return nil
    }

    switch rating {
    case 1:
        return p.handleRating1(jpgPath, rawPath)
    case 2:
        return p.handleRating2(jpgPath, rawPath)
    }

    return nil
}

func (p *ImageProcessor) handleRating1(jpgPath, rawPath string) error {
    // Logik für Rating 1
    return p.deleteFiles(jpgPath, rawPath)
}

func (p *ImageProcessor) handleRating2(jpgPath, rawPath string) error {
    if err := jpeg.ResizeWithXMP(jpgPath); err != nil {
        return fmt.Errorf("Fehler beim Verkleinern von %s: %v", jpgPath, err)
    }
    return p.deleteFile(rawPath)
}

func (p *ImageProcessor) deleteFile(path string) error {
    if err := os.Remove(path); err != nil {
        if !errors.Is(err, os.ErrNotExist) {
            return fmt.Errorf("Fehler beim Löschen von %s: %v", path, err)
        }
        fmt.Printf("Warnung: %s wurde bereits gelöscht\n", path)
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

func (p *ImageProcessor) ProcessDirectory(rawDir string, parentDir string) error {
    // Zuerst: Prüfe alle JPEGs auf Bewertungen
    jpegFiles, err := os.ReadDir(parentDir)
    if err != nil {
        return fmt.Errorf("Fehler beim Lesen des JPEG-Verzeichnisses %s: %v", parentDir, err)
    }

    // Verarbeite JPEGs basierend auf Bewertungen
    for _, file := range jpegFiles {
        if !file.IsDir() && strings.HasSuffix(strings.ToUpper(file.Name()), constants.JpegExtension) {
            jpgPath := filepath.Join(parentDir, file.Name())
            rawName := file.Name()[:len(file.Name())-len(constants.JpegExtension)] + constants.RawExtension
            rawPath := filepath.Join(rawDir, rawName)

            if err := p.ProcessJPEG(jpgPath, rawPath); err != nil {
                return fmt.Errorf("Fehler bei der Verarbeitung von %s: %v", jpgPath, err)
            }
        }
    }

    // Danach: Prüfe alle RAWs ohne zugehöriges JPEG
    rawFiles, err := os.ReadDir(rawDir)
    if err != nil {
        return fmt.Errorf("Fehler beim Lesen des RAW-Verzeichnisses %s: %v", rawDir, err)
    }

    for _, file := range rawFiles {
        if !file.IsDir() && strings.HasSuffix(strings.ToUpper(file.Name()), constants.RawExtension) {
            rawPath := filepath.Join(rawDir, file.Name())
            jpgName := file.Name()[:len(file.Name())-len(constants.RawExtension)] + constants.JpegExtension
            jpgPath := filepath.Join(parentDir, jpgName)

            // Prüfe ob JPEG existiert
            if _, err := os.Stat(jpgPath); err != nil {
                if errors.Is(err, os.ErrNotExist) {
                    fmt.Printf("Lösche %s (keine zugehörige JPG-Datei gefunden)\n", rawPath)
                    if err := p.deleteFile(rawPath); err != nil {
                        return fmt.Errorf("Fehler beim Löschen von %s: %v", rawPath, err)
                    }
                } else {
                    return fmt.Errorf("Fehler beim Prüfen von %s: %v", jpgPath, err)
                }
            }
        }
    }

    return nil
}

func (p *ImageProcessor) Walk() error {
    return filepath.Walk(p.RootDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if info.IsDir() && info.Name() == "raw" {
            parentDir := filepath.Dir(path)
            if err := p.ProcessDirectory(path, parentDir); err != nil {
                return fmt.Errorf("Fehler bei der Verarbeitung von %s: %v", path, err)
            }
        }
        return nil
    })
}