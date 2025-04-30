module github.com/frommie/rawmanager

go 1.21

require (
	github.com/disintegration/imaging v1.6.2
	github.com/dsoprea/go-jpeg-image-structure/v2 v2.0.0-20221012074422-4f3f7e934102
)

require (
	github.com/dsoprea/go-exif/v3 v3.0.0-20210428042052-dca55bf8ca15 // indirect
	github.com/dsoprea/go-iptc v0.0.0-20200609062250-162ae6b44feb // indirect
	github.com/dsoprea/go-logging v0.0.0-20200710184922-b02d349568dd // indirect
	github.com/dsoprea/go-photoshop-info-format v0.0.0-20200609050348-3db9b63b202c // indirect
	github.com/dsoprea/go-utility/v2 v2.0.0-20200717064901-2fccff4aa15e // indirect
	github.com/go-errors/errors v1.1.1 // indirect
	github.com/go-xmlfmt/xmlfmt v0.0.0-20191208150333-d5b6f63a941b // indirect
	github.com/golang/geo v0.0.0-20200319012246-673a6f80352d // indirect
	golang.org/x/image v0.0.0-20191009234506-e7c1f5e7dbb8 // indirect
	golang.org/x/net v0.0.0-20200707034311-ab3426394381 // indirect
	gopkg.in/yaml.v2 v2.3.0 // indirect
)

// Lokale Module
replace (
	github.com/frommie/rawmanager/constants => ./constants
	github.com/frommie/rawmanager/jpeg => ./jpeg
	github.com/frommie/rawmanager/processor => ./processor
	github.com/frommie/rawmanager/types => ./types
)
