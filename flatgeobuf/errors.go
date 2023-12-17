// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import (
	"errors"
	"fmt"
)

var (
	// ErrNoIndex is returned when attempting to do an index read or
	// search on a FlatGeobuf file that has no index.
	ErrNoIndex = textErr("no index")
	// ErrNotSeekable is returned from a FileReader's Rewind method if
	// its underlying stream does not implement io.Seeker.
	ErrNotSeekable = textErr("can't rewind: reader is not an io.Seeker")
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
	errIndexNotWritten     = "header specifies index but no index written"
	errReadPastIndex       = "read position is past index"
	errWritePastIndex      = "write position is past index"
	errSeekingData         = "failed to seek to data section"
	errIndexSize           = "failed to calculate index size"
	errDiscardIndex        = "failed to read past index"
)

const packageName = "flatgeobuf: "

func textErr(text string) error {
	return errors.New(packageName + text)
}

func fmtErr(format string, a ...any) error {
	return fmt.Errorf(packageName+format, a...)
}

func wrapErr(format string, err error, a ...any) error {
	return fmt.Errorf(packageName+format+": %w", append(a, err)...)
}

func textPanic(text string) {
	panic(packageName + text)
}

func fmtPanic(format string, a ...any) {
	panic(fmt.Sprintf(packageName+format, a...))
}
