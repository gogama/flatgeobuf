// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package packedrtree

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrors(t *testing.T) {
	t.Run("textErr", func(t *testing.T) {
		assert.Error(t, textErr("foo"), errors.New("packedrtree: foo"))
	})

	t.Run("fmtErr", func(t *testing.T) {
		assert.Error(t, fmtErr("bar", "baz", 11), errors.New("packedrtree: my bar is baz-ed to 11"))
	})

	t.Run("wrapErr", func(t *testing.T) {
		cause := errors.New("the root cause")
		err := wrapErr("the error is %q by", cause, "caused")

		assert.ErrorIs(t, err, cause)
		assert.Equal(t, err.Error(), `packedrtree: the error is "caused" by: the root cause`)
	})

	t.Run("textPanic", func(t *testing.T) {
		assert.PanicsWithValue(t, "packedrtree: foo", func() {
			textPanic("foo")
		})
	})

	t.Run("fmtPanic", func(t *testing.T) {
		assert.PanicsWithValue(t, "packedrtree: my bar is baz-ed to 10", func() {
			fmtPanic("my %s is %s-ed to %d", "bar", "baz", 10)
		})
	})
}
