package errkit

import "fmt"

func Wrap(target error, op string, cause error) error {
	if cause == nil {
		return nil
	}

	return fmt.Errorf("%w: %s: %s", target, op, cause.Error())
}
