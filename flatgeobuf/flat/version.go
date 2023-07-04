// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flat

import _ "embed"

var (
	//go:embed "version-flatc.txt"
	flatcVersion string
	//go:embed "version-schema.txt"
	schemaVersion string
)
