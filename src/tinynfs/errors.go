package tinynfs

import "errors"

var (
	ErrParam         = errors.New("bad parameters")
	ErrDiskFully     = errors.New("not enough disk space")
	ErrTimestamp     = errors.New("unacceptable timestamp")
	ErrMediaType     = errors.New("unsupported media type")
	ErrThumbnailSize = errors.New("unacceptable thumbnail size")
)
