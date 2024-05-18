package main

import (
	"fmt"
	"os"
	"log"
	"errors"
)

func main() {
	files, err := os.ReadDir("./raw/")
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		rawlen := len(file.Name())
		if file.Name()[rawlen-3:] != "RAF" {
			continue
		}

		jpgfile := file.Name()[:rawlen-3] + "JPG"
		// check if corresponding JPG is present in root dir
		if _, err := os.Stat("./"+jpgfile); err == nil {
			// JPG file is present - skip
			continue
		} else if errors.Is(err, os.ErrNotExist) {
			// path/to/whatever does *not* exist - delete raw file
			fmt.Println("deleted " + file.Name())
			e := os.Remove("./raw/" + file.Name()) 
			if e != nil { 
				log.Fatal(e) 
			}
		} else {
			// Schrodinger: file may or may not exist. See err for details.
		
			// Therefore, do *NOT* use !os.IsNotExist(err) to test for file existence
		
		
		}
	}
}