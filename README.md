# rawmanager

![Version](https://img.shields.io/github/v/release/frommie/rawmanager)
[![Go Report Card](https://goreportcard.com/badge/github.com/frommie/rawmanager)](https://goreportcard.com/report/github.com/frommie/rawmanager)
[![License](https://img.shields.io/github/license/frommie/rawmanager)](LICENSE)

A tool to manage RAW image files based on their corresponding JPEG ratings.

## Features

- Processes RAW+JPEG pairs in your photo library
- Deletes or resize files based on JPEG ratings (1-5 stars)
- Resizing preserves EXIF data
- Supports embedded and separate XMP metadata
- Configurable actions for each rating level
- Progress bars and detailed logging (optional)

## Installation

```bash
# Clone the repository
git clone https://github.com/frommie/rawmanager.git

# Change to project directory
cd rawmanager

# Install dependencies
go mod download

# Install the tool
go install
```

## Usage

```bash
rawmanager [-config path/to/config.yaml] [-v] [directory]
```

Options:
- `-config`: Path to configuration file (default: config.yaml)
- `-v`: Verbose output
- `directory`: Directory to process (default: current directory)

## Configuration

Create a `config.yaml` file to customize the behavior. The default config is as follows:

```yaml
# Rating Actions (0-5)
ratingActions:
  1:
    deleteRaw: true    # Delete RAW file
    deleteJpeg: true   # Delete JPEG file
    compressJpeg: false # Compress JPEG file
  2:
    deleteRaw: true
    deleteJpeg: false
    compressJpeg: true
  # ... configure other ratings as needed

# Action when no JPEG is found
noJpegAction:
  deleteRaw: true
  deleteJpeg: false
  compressJpeg: false

# XMP Configuration
xmp:
  mode: "embedded"  # embedded, separate (.xmp), or separate_ext (.jpg.xmp)

# File Configuration
files:
  rawExtension: ".RAF"  # Your RAW file extension
  jpegExtension: ".JPG" # Your JPEG file extension
  rawFolder: "raw"      # RAW files subfolder
  sameDir: false        # true if RAWs are in same directory

# Process Configuration
process:
  targetMegapixels: 10.0 # Target size for JPEG compression
  jpegQuality: 95        # JPEG quality (0-100)
```

## Requirements

- Go 1.16 or higher
- Operating systems: macOS, Linux, Windows

## License

This project is licensed under the [MIT License](LICENSE.md).