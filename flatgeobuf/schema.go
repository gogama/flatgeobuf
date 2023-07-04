// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import "github.com/gogama/flatgeobuf/flatgeobuf/flat"

// TODO: Docs
type Schema interface {
	// TODO: Docs
	ColumnsLength() int
	// TODO: Docs
	Columns(obj *flat.Column, j int) bool
}
