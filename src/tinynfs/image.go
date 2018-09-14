package tinynfs

import (
	"bytes"
	"golang.org/x/image/draw"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"math"
)

var (
	defaultScaler = draw.BiLinear
)

func init() {
	image.RegisterFormat("gif", "gif", gif.Decode, gif.DecodeConfig)
	image.RegisterFormat("png", "png", png.Decode, png.DecodeConfig)
	image.RegisterFormat("jpeg", "jpeg", jpeg.Decode, jpeg.DecodeConfig)
}

func scaleImageSize(owidth int, oheight int, awidth int, aheight int) (int, int) {
	if owidth == awidth && oheight == aheight {
		return owidth, oheight
	}
	if owidth > oheight {
		if owidth > awidth {
			oheight = int(math.Floor(float64(awidth) * float64(oheight) / float64(owidth)))
			owidth = awidth
		}
	} else if owidth < oheight {
		if oheight > aheight {
			owidth = int(math.Floor(float64(aheight) * float64(owidth) / float64(oheight)))
			oheight = aheight
		}
	} else {
		side := awidth
		if awidth > aheight {
			side = aheight
		}
		owidth = side
		oheight = side
	}
	return owidth, oheight
}

func ImageParseBuffer(data []byte, maxSide int, maxSize int) (int, int, string, []byte, error) {
	reader := bytes.NewReader(data)
	origin, format, err := image.Decode(reader)
	if err != nil {
		return 0, 0, "", nil, ErrMediaType
	}
	width := origin.Bounds().Dx()
	height := origin.Bounds().Dy()

	if format == "gif" {
		// ignore optimize
	} else if maxSide > 0 && (width > maxSide || height > maxSide) {
		width, height = scaleImageSize(width, height, maxSide, maxSide)
		target := image.NewRGBA(image.Rect(0, 0, width, height))
		defaultScaler.Scale(target, target.Bounds(), origin, origin.Bounds(), draw.Over, nil)
		buffer := bytes.NewBuffer(nil)
		if format == "jpeg" {
			if err := jpeg.Encode(buffer, target, nil); err != nil {
				return 0, 0, "", nil, err
			}
		} else {
			if err := png.Encode(buffer, target); err != nil {
				return 0, 0, "", nil, err
			}
		}
		data = buffer.Bytes()
	} else if maxSize > 0 && len(data) > maxSize {
		buffer := bytes.NewBuffer(nil)
		if format == "jpeg" {
			if err := jpeg.Encode(buffer, origin, nil); err != nil {
				return 0, 0, "", nil, err
			}
		} else {
			if err := png.Encode(buffer, origin); err != nil {
				return 0, 0, "", nil, err
			}
		}
		data = buffer.Bytes()
	}
	return width, height, format, data, nil
}

func ImageScaleBuffer(data []byte, awidth int, aheight int) (int, int, string, []byte, error) {
	reader := bytes.NewReader(data)
	origin, format, err := image.Decode(reader)
	if err != nil {
		return 0, 0, "", nil, ErrMediaType
	}
	owidth := origin.Bounds().Dx()
	oheight := origin.Bounds().Dy()

	width, height := scaleImageSize(owidth, oheight, awidth, aheight)
	target := image.NewRGBA(image.Rect(0, 0, width, height))
	defaultScaler.Scale(target, target.Bounds(), origin, origin.Bounds(), draw.Over, nil)
	buffer := bytes.NewBuffer(nil)
	if format == "jpeg" {
		if err := jpeg.Encode(buffer, target, nil); err != nil {
			return 0, 0, "", nil, err
		}
	} else { // Gif to png
		if err := png.Encode(buffer, target); err != nil {
			return 0, 0, "", nil, err
		}
		format = "png"
	}
	return width, height, format, buffer.Bytes(), nil
}
