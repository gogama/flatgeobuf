// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_stateful_close(t *testing.T) {
	t.Run("Already Closed", func(t *testing.T) {
		s := stateful{err: ErrClosed}

		err := s.close(nil)

		assert.Same(t, ErrClosed, err)
		assert.Same(t, ErrClosed, s.err)
	})

	t.Run("Not a Closer", func(t *testing.T) {
		var s stateful

		err := s.close(nil)

		assert.NoError(t, err)
		assert.Same(t, ErrClosed, s.err)
	})

	t.Run("Closer", func(t *testing.T) {
		t.Run("Close Error", func(t *testing.T) {
			expectedErr := errors.New("it's not closing time")
			r := newMockReadCloser(t)
			r.
				On("Close").
				Return(expectedErr).
				Once()
			var s stateful

			actualErr := s.close(r)

			assert.Same(t, expectedErr, actualErr)
			assert.Same(t, ErrClosed, s.err)
			r.AssertExpectations(t)
		})

		t.Run("Close Success", func(t *testing.T) {
			t.Run("No Previous Error", func(t *testing.T) {
				r := newMockReadCloser(t)
				r.
					On("Close").
					Return(nil).
					Once()
				var s stateful

				err := s.close(r)

				assert.Nil(t, err)
				assert.Same(t, ErrClosed, s.err)
				r.AssertExpectations(t)
			})

			t.Run("Had Previous Error", func(t *testing.T) {
				w := newMockWriteCloser(t)
				w.
					On("Close").
					Return(nil).
					Once()
				s := stateful{err: errors.New("previous error")}

				err := s.close(w)

				assert.Nil(t, err)
				assert.Same(t, ErrClosed, s.err)
				w.AssertExpectations(t)
			})
		})
	})
}

func Test_stateful_toState(t *testing.T) {
	variations := []struct {
		name string
		tt   transitionType
	}{
		{"Outside", outside},
		{"Inside", inside},
	}

	t.Run("Already in Error State", func(t *testing.T) {
		for _, variation := range variations {
			t.Run(variation.name, func(t *testing.T) {
				expectedErr := errors.New("some previous error")
				s := stateful{err: expectedErr}

				actualErr := s.toState(uninitialized, beforeMagic, variation.tt)

				assert.Same(t, expectedErr, actualErr)
				assert.Equal(t, uninitialized, s.state)
			})
		}
	})

	t.Run("Successful Transition", func(t *testing.T) {
		for _, variation := range variations {
			t.Run(variation.name, func(t *testing.T) {
				const before = afterHeader
				const after = eof
				s := stateful{state: before}

				err := s.toState(before, after, variation.tt)

				assert.NoError(t, err)
				assert.Equal(t, after, s.state)
			})
		}
	})

	t.Run("Unsuccessful Transition", func(t *testing.T) {
		t.Run("Inside Panic", func(t *testing.T) {
			var s stateful

			assert.PanicsWithValue(t, "flatgeobuf: logic error: failed inside transition 0x52 -> 0x22: real state is 0x0", func() {
				_ = s.toState(eof, afterHeader, inside)
			})
		})

		t.Run("Outside Error", func(t *testing.T) {
			var s stateful

			err := s.toState(eof, afterHeader, outside)

			assert.EqualError(t, err, "flatgeobuf: unexpected state")
		})
	})
}

func Test_stateful_toErr(t *testing.T) {
	t.Run("Already in Error State", func(t *testing.T) {
		s := stateful{err: errors.New("a lingering problem")}

		assert.PanicsWithValue(t, "flatgeobuf: logic error: already in error state", func() {
			_ = s.toErr(errors.New("another problem"))
		})
	})

	t.Run("Successful Transition", func(t *testing.T) {
		expectedErr := errors.New("an unprecedented problem")
		var s stateful

		actualErr := s.toErr(expectedErr)

		assert.Same(t, expectedErr, actualErr)
	})
}
