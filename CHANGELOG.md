# Changelog
All notable changes to this project will be documented in this file.

## [Unreleased]
### Added
- `-dry-run` flag to show all actions without changing any files
- Ratings stored as XMP attributes are recognized

### Changed
- Missing settings in the configuration file keep their default values;
  configured `ratingActions` replace the default rating actions
- The configuration is rejected if a file extension is empty or a rating
  action deletes the JPEG but keeps the RAW or deletes and compresses the JPEG
- File extensions are matched case-insensitively
- Warnings are always shown (on stderr); unrated JPEGs and JPEGs without RAW
  are only reported in verbose mode
- Hidden directories are skipped
- Resized JPEGs are replaced atomically and keep their file permissions

### Fixed
- JPEG compression always failed
- Microsoft ratings were converted incorrectly (e.g. 99 became 4 stars)
- `noJpegAction.deleteRaw: false` was ignored
- Progress bars did not match the processed files
- Compression dropped the ICC color profile and other metadata segments;
  extended XMP could replace the EXIF data

## [1.0.0] - 2024-05-04
### Added
- Initial release
- Support for rating-based JPEG/RAW management
- XMP metadata handling (embedded and sidecar)
- JPEG compression functionality
- Progress bar visualization
- Configurable file extensions and directories