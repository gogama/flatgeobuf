package packedrtree

import (
	"errors"
	"fmt"
)

const packageName = "packedrtree: "

func textErr(text string) error {
	return errors.New(packageName + text)
}

func fmtErr(format string, a ...interface{}) error { // TODO: Delete if unused
	return fmt.Errorf(packageName+format, a...)
}

func wrapErr(text string, err error, a ...interface{}) error { // TODO: Delete if unused
	return fmt.Errorf(packageName+text+": %w", append(a, err)...)
}

func textPanic(text string) {
	panic(packageName + text)
}

func fmtPanic(format string, a ...interface{}) { // TODO: Delete if unused
	panic(fmt.Sprintf(packageName+format, a...))
}
