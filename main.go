package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/frommie/rawmanager/config"
	"github.com/frommie/rawmanager/processor"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		log.Fatal(err)
	}
}

// run parses the command line arguments and processes the photo directory
func run(args []string, stdout, stderr io.Writer) error {
	var (
		configPath string
		verbose    bool
	)

	flags := flag.NewFlagSet("rawmanager", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&configPath, "config", "", "Path to YAML configuration file (default: built-in configuration)")
	flags.BoolVar(&verbose, "v", false, "Verbose mode (shows detailed output)")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	// Use the directory passed as argument or the current directory
	photosDir := flags.Arg(0)
	if photosDir == "" {
		var err error
		photosDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("error determining current directory: %w", err)
		}
	}

	cfg := config.NewDefaultConfig()
	if configPath != "" {
		var err error
		cfg, err = config.LoadConfig(configPath)
		if err != nil {
			return fmt.Errorf("error loading configuration: %w", err)
		}
	}

	proc := processor.NewImageProcessor(photosDir, cfg, verbose)
	proc.Out = stdout
	proc.ErrOut = stderr
	return proc.Process()
}
