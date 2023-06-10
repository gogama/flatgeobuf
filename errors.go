package flatgeobuf

import (
	"errors"
	"fmt"
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
