package qpm

import "fmt"

func WrapErrorf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	args = append(args, err)
	return fmt.Errorf(format+": %w", args...)
}