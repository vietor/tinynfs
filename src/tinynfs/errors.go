package tinynfs

import "errors"

var (
	ErrParam         = errors.New("bad parameters")
	ErrDiskFull      = errors.New("disk maybe full")
	ErrTimestamp     = errors.New("unacceptable timestamp")
	ErrMediaType     = errors.New("unsupported media type")
	ErrThumbnailSize = errors.New("unacceptable thumbnail size")
)
