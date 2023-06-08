package flatgeobuf

import _ "embed"

var (
	//go:embed version-flatc.txt
	flatcVersion string
	//go:embed version-schema.txt
	schemaVersion string
)
