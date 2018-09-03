package tinynfs

import "errors"

var (
	ErrParam         = errors.New("bad parameters")
	ErrTimestamp     = errors.New("unacceptable timestamp")
	ErrMediaType     = errors.New("unsupported media type")
	ErrThumbnailSize = errors.New("unacceptable thumbnail size")

	ErrDiskFully          = errors.New("not enough disk space")
	ErrFileSystemBusy     = errors.New("file system already lock")
	ErrFileSystemFully    = errors.New("file system diskspace fully")
	ErrVolumeStorageBusy  = errors.New("volume storage diskspace fully")
	ErrVolumeStorageFully = errors.New("volume storage already lock")
)
