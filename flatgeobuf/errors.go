// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import (
	"errors"
	"fmt"
)

var (
	// ErrNoIndex is returned when attempting to perform an index search
	// on a FlatGeobuf file which does not contain an index.
	ErrNoIndex = textErr("no index")
	// ErrClosed is returned when attempting to perform an operation on
	// a FileReader or FileWriter which has been closed.
	ErrClosed = textErr("closed")

	errEndOfData       = textErr("end of data section")
	errUnexpectedState = textErr("unexpected state")
)

const (
	errHeaderNotCalled     = "must call Header()"
	errHeaderAlreadyCalled = "Header() has already been called"
	errHeaderNodeSizeZero  = "header node size 0 indicates no index"
	errIndexNotWritten     = "header requires index but no index written"
	errReadPastIndex       = "read position is past index"
	errWritePastIndex      = "write position is past index"
)

const packageName = "flatgeobuf: "

func textErr(text string) error {
	return errors.New(packageName + text)
}

func fmtErr(format string, a ...interface{}) error {
	return fmt.Errorf(packageName+format, a...)
}

func wrapErr(text string, err error, a ...interface{}) error {
	return fmt.Errorf(packageName+text+": %w", append(a, err)...)
}

func textPanic(text string) {
	panic(packageName + text)
}

func fmtPanic(format string, a ...interface{}) {
	panic(fmt.Sprintf(packageName+format, a...))
}
