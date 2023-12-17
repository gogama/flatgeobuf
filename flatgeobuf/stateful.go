// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import "io"

type stateful struct {
	state state
	err   error
}

type state int

const (
	uninitialized state = 0x00
	beforeMagic   state = 0x11
	beforeHeader  state = 0x21
	afterHeader   state = 0x22
	beforeIndex   state = 0x31
	afterIndex    state = 0x32
	inData        state = 0x42
	eof           state = 0x52
)

type transitionType int

const (
	// An outside transition is the first state transition of a public
	// method. If a public method has state transitions A -> B and then
	// B -> C and C -> D, then A -> B is an outside transition.
	outside transitionType = 0
	// An inside transition is any state transition after the first one
	// in a public method. If a public method has state transitions
	// A -> B and then B -> C and C -> D, then B -> C and C -> D are
	// inside transitions.
	inside transitionType = 1
)

func (s *stateful) close(a any) error {
	if s.err == ErrClosed {
		return ErrClosed
	}

	s.err = ErrClosed

	if c, ok := a.(io.Closer); ok {
		if err := c.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (s *stateful) toState(expected, to state, tt transitionType) (err error) {
	// Always fail if the reader's already in the error state.
	if s.err != nil {
		return s.err
	}

	// Happy path to state transition is when reader is in the expected
	// state.
	if s.state == expected {
		s.state = to
		return nil
	}

	// Panic if an inside transition failed, as this represents a
	// programming logic error.
	if tt == inside {
		fmtPanic("logic error: failed inside transition 0x%x -> 0x%x: real state is 0x%x", expected, to, s.state)
	}

	// Indicate that the state transition is invalid.
	return errUnexpectedState
}

func (s *stateful) toErr(err error) error {
	if s.err != nil {
		textPanic("logic error: already in error state")
	}

	s.err = err
	return err
}
