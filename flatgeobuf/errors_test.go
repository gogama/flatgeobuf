// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTextErr(t *testing.T) {
	err := textErr("foo")

	assert.Error(t, err, "flatgeobuf: foo")
}

func TestFmtErr(t *testing.T) {
	err := fmtErr("foo: %s/%s %.2f", "bar", "baz", 3.14159)

	assert.Error(t, err, "flatgeobuf: foo: bar/baz 3.14")
}

func TestWrapErr(t *testing.T) {
	cause := errors.New("foo")

	err := wrapErr("%s (%d)", cause, "bar", -15)

	assert.Error(t, err, "flatgeobuf: bar (-15): foo")
	assert.ErrorIs(t, err, cause)
}

func TestTextPanic(t *testing.T) {
	assert.PanicsWithValue(t, "flatgeobuf: foo", func() {
		textPanic("foo")
	})
}

func TestFmtPanic(t *testing.T) {
	assert.PanicsWithValue(t, "flatgeobuf: bar% false 21 baz", func() {
		fmtPanic("bar%% %t %d %s", false, 21, "baz")
	})
}
