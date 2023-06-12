package flatgeobuf

import (
	"fmt"
)

// safeFlatBuffersInteraction runs a function that interacts with
// FlatBuffers, trapping any panic that occurs and converting it to a
// normal Go error.
//
// This function exists because FlatBuffer's Go code doesn't use
// standard Go error handling, allegedly for performance reasons, and
// consequently any invalid attempt to interact with FlatBuffer data
// may trigger a panic.
func safeFlatBuffersInteraction(f func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: flatbuffers: %v", r)
		}
	}()
	f()
	return
}
