package parse

import (
	"errors"
	"fmt"
)

var ErrSyntax = errors.New("syntax error")
var ErrParse = errors.New("parse error")

func Errorf(kind error, format string, args ...interface{}) error {
	return fmt.Errorf("%w: "+format, append([]interface{}{kind}, args...)...)
}
