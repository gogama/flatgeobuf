// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package packedrtree

import (
	"errors"
	"fmt"
)

const packageName = "packedrtree: "

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
