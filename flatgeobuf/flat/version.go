// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flat

import (
	_ "embed"
)

var (
	//go:embed "version-flatc.txt"
	flatcVersion string
	//go:embed "version-schema.txt"
	schemaVersion string
	// Version documents the software versions used to build package
	// flat.
	Version = struct {
		// Flatc contains the version of the FlatBuffers compiler,
		// flatc, used to build package flat.
		Flatc string
		// FlatGeobufSchema contains the version of the FlatGeobuf
		// FlatBuffers schema used to build package flat.
		FlatGeobufSchema string
	}{
		Flatc:            flatcVersion,
		FlatGeobufSchema: schemaVersion,
	}
)
