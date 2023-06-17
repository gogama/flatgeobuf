package flatgeobuf

import "io"

type stateful struct {
	state state
	err   error
}

type state int

const (
	uninitialized state = 0x00
	invalid             = 0x01
	beforeMagic         = 0x11
	beforeHeader        = 0x21
	afterHeader         = 0x22
	beforeIndex         = 0x31
	afterIndex          = 0x32
	inData              = 0x42
	eof                 = 0x52
)

func (s *stateful) close(a interface{}) error {
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

func (s *stateful) sanityCheckState() {
	if s.state&invalid == invalid {
		fmtPanic("logic error: invalid state 0x%x", s.state)
	}
}

// FIXME: Internal state transitions within functions should panic, not return error.
func (s *stateful) toState(expected, to state) (err error) {
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

	// Check for bad internal state.
	s.sanityCheckState()

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
