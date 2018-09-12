package tinynfs

import (
	"errors"
	"net/http"
	"os"
)

var (
	ErrExist      = os.ErrExist
	ErrNotExist   = os.ErrNotExist
	ErrPermission = os.ErrPermission

	ErrParam         = errors.New("bad parameters")
	ErrTimestamp     = errors.New("unacceptable timestamp")
	ErrMediaType     = errors.New("unsupported media type")
	ErrThumbnailSize = errors.New("unacceptable thumbnail size")

	ErrIndexStorageBusy   = errors.New("index storage already lock")
	ErrIndexStorageFully  = errors.New("index storage disk space fully")
	ErrVolumeStorageBusy  = errors.New("volume storage already lock")
	ErrVolumeStorageFully = errors.New("volume storage disk space fully")
)

var (
	errorCodes = map[error]int{
		ErrParam:              101,
		ErrPermission:         102,
		ErrExist:              103,
		ErrNotExist:           104,
		ErrMediaType:          105,
		ErrThumbnailSize:      106,
		ErrIndexStorageFully:  201,
		ErrVolumeStorageFully: 202,
	}
	httpStatusCodes = map[error]int{
		ErrParam:         http.StatusBadRequest,
		ErrPermission:    http.StatusForbidden,
		ErrExist:         http.StatusForbidden,
		ErrNotExist:      http.StatusNotFound,
		ErrMediaType:     http.StatusUnsupportedMediaType,
		ErrThumbnailSize: http.StatusBadRequest,
	}
)

func toErrorCode(err error) int {
	code, ok := errorCodes[err]
	if !ok {
		code = 100 // undefined error code
	}
	return code
}

func toStatusCode(err error) int {
	code, ok := httpStatusCodes[err]
	if !ok {
		code = http.StatusInternalServerError
	}
	return code
}
