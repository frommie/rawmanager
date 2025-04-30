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
        if os.IsNotExist(err) {
            fmt.Printf("Info: JPEG nicht gefunden: %s\n", jpgPath)
            return nil
        }
        fmt.Printf("Warnung: Konnte Bewertung für %s nicht lesen: %v\n", jpgPath, err)
        return nil
    }

    // Prüfe ob RAW existiert bevor wir weitermachen
    if _, err := os.Stat(rawPath); err != nil {
        if os.IsNotExist(err) {
            fmt.Printf("Info: RAW nicht gefunden: %s\n", rawPath)
            return nil
        }
        fmt.Printf("Warnung: Fehler beim Zugriff auf RAW %s: %v\n", rawPath, err)
        return nil
    }

    switch rating {
    case 1:
        if err := p.handleRating1(jpgPath, rawPath); err != nil {
            fmt.Printf("Warnung: Fehler bei Rating 1 Verarbeitung von %s: %v\n", jpgPath, err)
            return nil
        }
    case 2:
        if err := p.handleRating2(jpgPath, rawPath); err != nil {
            fmt.Printf("Warnung: Fehler bei Rating 2 Verarbeitung von %s: %v\n", jpgPath, err)
            return nil
        }
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
    // Prüfe ob raw-Verzeichnis existiert
    if _, err := os.Stat(rawDir); err != nil {
        if os.IsNotExist(err) {
            fmt.Printf("Info: Überspringe nicht existierendes Verzeichnis: %s\n", rawDir)
            return nil
        }
        return fmt.Errorf("Fehler beim Zugriff auf Verzeichnis %s: %v", rawDir, err)
    }

    // Zuerst: Prüfe alle JPEGs auf Bewertungen
    jpegFiles, err := os.ReadDir(parentDir)
    if err != nil {
        if os.IsNotExist(err) {
            fmt.Printf("Info: Überspringe nicht existierendes Verzeichnis: %s\n", parentDir)
            return nil
        }
        return fmt.Errorf("Fehler beim Lesen des JPEG-Verzeichnisses %s: %v", parentDir, err)
    }

    // Verarbeite JPEGs basierend auf Bewertungen
    for _, file := range jpegFiles {
        if !file.IsDir() && strings.HasSuffix(strings.ToUpper(file.Name()), constants.JpegExtension) {
            jpgPath := filepath.Join(parentDir, file.Name())
            rawName := file.Name()[:len(file.Name())-len(constants.JpegExtension)] + constants.RawExtension
            rawPath := filepath.Join(rawDir, rawName)

            if _, err := os.Stat(rawPath); err != nil && os.IsNotExist(err) {
                fmt.Printf("Info: Keine RAW-Datei gefunden für: %s\n", jpgPath)
                continue
            }

            if err := p.ProcessJPEG(jpgPath, rawPath); err != nil {
                fmt.Printf("Warnung: Fehler bei der Verarbeitung von %s: %v\n", jpgPath, err)
                continue
            }
        }
    }

    // Danach: Prüfe alle RAWs ohne zugehöriges JPEG
    rawFiles, err := os.ReadDir(rawDir)
    if err != nil {
        if os.IsNotExist(err) {
            fmt.Printf("Info: Überspringe nicht existierendes Verzeichnis: %s\n", rawDir)
            return nil
        }
        return fmt.Errorf("Fehler beim Lesen des RAW-Verzeichnisses %s: %v", rawDir, err)
    }

    for _, file := range rawFiles {
        if !file.IsDir() && strings.HasSuffix(strings.ToUpper(file.Name()), constants.RawExtension) {
            rawPath := filepath.Join(rawDir, file.Name())
            jpgName := file.Name()[:len(file.Name())-len(constants.RawExtension)] + constants.JpegExtension
            jpgPath := filepath.Join(parentDir, jpgName)

            // Prüfe ob JPEG existiert
            if _, err := os.Stat(jpgPath); err != nil {
                if os.IsNotExist(err) {
                    if err := p.deleteFile(rawPath); err != nil {
                        fmt.Printf("Warnung: Fehler beim Löschen von %s: %v\n", rawPath, err)
                        continue
                    }
                    fmt.Printf("Info: RAW-Datei gelöscht (keine JPG gefunden): %s\n", rawPath)
                } else {
                    fmt.Printf("Warnung: Fehler beim Prüfen von %s: %v\n", jpgPath, err)
                    continue
                }
            }
        }
    }

    return nil
}

func (p *ImageProcessor) Walk() error {
    return filepath.Walk(p.RootDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            if os.IsNotExist(err) {
                fmt.Printf("Info: Überspringe nicht existierenden Pfad: %s\n", path)
                return nil
            }
            fmt.Printf("Warnung: Fehler beim Zugriff auf %s: %v\n", path, err)
            return nil
        }
        if info.IsDir() && info.Name() == "raw" {
            parentDir := filepath.Dir(path)
            if err := p.ProcessDirectory(path, parentDir); err != nil {
                fmt.Printf("Warnung: Fehler bei der Verarbeitung von %s: %v\n", path, err)
                return nil // Weitermachen statt abbrechen
            }
        }
        return nil
    })
}