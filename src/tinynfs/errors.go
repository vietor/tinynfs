package tinynfs

import "errors"

var (
	ErrParam    = errors.New("bad parameters")
	ErrDiskFull = errors.New("disk maybe full")
)
