// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import _ "embed"

var (
	//go:embed "../script/version-flatc.txt"
	flatcVersion string
	//go:embed "../script/version-schema.txt"
	schemaVersion string
)
