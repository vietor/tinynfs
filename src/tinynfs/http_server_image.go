package tinynfs

import (
	"bytes"
	"fmt"
	"golang.org/x/image/draw"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

func init() {
	image.RegisterFormat("gif", "gif", gif.Decode, gif.DecodeConfig)
	image.RegisterFormat("png", "png", png.Decode, png.DecodeConfig)
	image.RegisterFormat("jpeg", "jpeg", jpeg.Decode, jpeg.DecodeConfig)
}

func getScaledSize(owidth int, oheight int, awidth int, aheight int) (int, int) {
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

func (self *HttpServer) startImage() {
	var (
		serveMux = http.NewServeMux()
		server   = &http.Server{
			Handler: serveMux,
		}
	)
	serveMux.HandleFunc("/", self.handleImageGet)
	serveMux.HandleFunc("/upload", self.handleImageUpload)
	serveMux.HandleFunc("/uploads", self.handleImageUploadMore)
	err := server.Serve(self.imageListener)
	if err != nil && !self.closed {
		fmt.Println(err)
	}
}

func (self *HttpServer) parseImageSize(size string) (int, int) {
	fields := strings.Split(size, "x")
	if len(fields) != 2 {
		return 0, 0
	}
	width, err := strconv.ParseInt(fields[0], 10, 32)
	if err != nil {
		return 0, 0
	}
	height, err := strconv.ParseInt(fields[1], 10, 32)
	if err != nil {
		return 0, 0
	}
	return int(width), int(height)
}

func (self *HttpServer) handleImageGet(res http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" && req.Method != "HEAD" {
		http.Error(res, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var (
		xerr  error
		xmime string
		xdata []byte
	)
	defer self.sendByteData(res, req, &xerr, &xmime, &xdata)

	filepath := req.URL.Path
	if strings.HasSuffix(filepath, "/") {
		xerr = ErrParam
		return
	}

	var (
		err        error
		awidth     int
		aheight    int
		mimedata   string
		metadata   string
		imagedata  []byte
		originpath string
	)

	if m, _ := regexp.MatchString("_[0-9]+x[0-9]+$", filepath); m {
		n := strings.LastIndex(filepath, "_")
		size := filepath[n+1:]
		if _, ok := self.config.ImageThumbnailSizes[size]; !ok {
			xerr = ErrThumbnailSize
			return
		}
		awidth, aheight = self.parseImageSize(size)
		if awidth == 0 || aheight == 0 {
			xerr = ErrThumbnailSize
			return
		}
		originpath = filepath[:n]
	}

	// Read thumbnail file
	mimedata, metadata, imagedata, err = self.storage.ReadFile(filepath)
	if err == nil {
		xmime = mimedata
		xdata = imagedata
		return
	} else if err != ErrNotExist || len(originpath) < 1 {
		xerr = err
		return
	}

	// Read origin file
	mimedata, metadata, imagedata, err = self.storage.ReadFile(originpath)
	if err != nil {
		xerr = err
		return
	}
	owidth, oheight := self.parseImageSize(metadata)
	if owidth == 0 || oheight == 0 {
		xerr = ErrThumbnailSize
		return
	}
	// Ignore image scale
	if owidth < awidth && oheight < aheight {
		xmime = mimedata
		xdata = imagedata
		return
	}

	// Create thumbnail image
	reader := bytes.NewReader(imagedata)
	origin, _, err := image.Decode(reader)
	if err != nil {
		xerr = ErrMediaType
		return
	}
	width, height := getScaledSize(owidth, oheight, awidth, aheight)
	target := image.NewRGBA(image.Rect(0, 0, width, height))
	scaler := draw.ApproxBiLinear
	if width*5 < owidth || height*5 < oheight {
		scaler = draw.BiLinear
	}
	scaler.Scale(target, target.Bounds(), origin, origin.Bounds(), draw.Over, nil)
	buffer := bytes.NewBuffer(nil)
	if mimedata == "image/jpeg" {
		if err := jpeg.Encode(buffer, target, nil); err != nil {
			xerr = err
			return
		}
	} else { // Gif to png
		if err := png.Encode(buffer, target); err != nil {
			xerr = err
			return
		}
		mimedata = "image/png"
	}
	imagedata = buffer.Bytes()

	filepath = fmt.Sprintf("%s_%dx%d", originpath, awidth, aheight)
	metadata = fmt.Sprintf("%dx%d", width, height)
	options := &WriteOptions{
		Overwrite: false,
	}
	if err := self.storage.WriteFile(filepath, mimedata, metadata, imagedata, options); err != nil && err != ErrExist {
		xerr = err
		return
	}
	xmime = mimedata
	xdata = imagedata
}

func (self *HttpServer) saveImageToStorage(dataimage io.Reader) (map[string]interface{}, error) {
	data, err := ioutil.ReadAll(dataimage)
	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(data)
	origin, format, err := image.Decode(reader)
	if err != nil {
		return nil, ErrMediaType
	}
	width := origin.Bounds().Dx()
	height := origin.Bounds().Dy()

	if format == "gif" {
		// ignore optimize
	} else if self.config.ImageOtimizeSide > 0 && (width > self.config.ImageOtimizeSide || height > self.config.ImageOtimizeSide) {
		width, height = getScaledSize(width, height, self.config.ImageOtimizeSide, self.config.ImageOtimizeSide)
		target := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.BiLinear.Scale(target, target.Bounds(), origin, origin.Bounds(), draw.Over, nil)
		buffer := bytes.NewBuffer(nil)
		if format == "jpeg" {
			if err := jpeg.Encode(buffer, target, nil); err != nil {
				return nil, err
			}
		} else {
			if err := png.Encode(buffer, target); err != nil {
				return nil, err
			}
		}
		data = buffer.Bytes()
	} else if self.config.ImageOtimizeSize > 0 && len(data) > self.config.ImageOtimizeSize {
		buffer := bytes.NewBuffer(nil)
		if format == "jpeg" {
			if err := jpeg.Encode(buffer, origin, nil); err != nil {
				return nil, err
			}
		} else {
			if err := png.Encode(buffer, origin); err != nil {
				return nil, err
			}
		}
		data = buffer.Bytes()
	}

	mimedata := "image/" + format
	metadata := fmt.Sprintf("%dx%d", width, height)
	filepath := self.config.ImageFilePath + RandHex(10) + TimeHex(0)
	if err := self.storage.WriteFile(filepath, mimedata, metadata, data, nil); err != nil {
		return nil, err
	}

	imageout := map[string]interface{}{}
	imageout["size"] = len(data)
	imageout["mime"] = mimedata
	imageout["width"] = width
	imageout["height"] = height
	imageout["image_url"] = filepath
	return imageout, nil
}

func (self *HttpServer) handleImageUpload(res http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		http.Error(res, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var (
		xerr  error
		xdata = map[string]interface{}{}
	)
	defer self.sendJsonData(res, req, &xerr, xdata)

	if err := self.parseRequestBody(req); err != nil {
		xerr = err
		return
	}

	dataimage, _, err := req.FormFile("imagedata")
	if err != nil {
		xerr = ErrParam
		return
	}
	imageout, err := self.saveImageToStorage(dataimage)
	if err != nil {
		xerr = err
		return
	}
	for k, v := range imageout {
		xdata[k] = v
	}
}

func (self *HttpServer) handleImageUploadMore(res http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		http.Error(res, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var (
		xerr  error
		xdata = map[string]interface{}{}
	)
	defer self.sendJsonData(res, req, &xerr, xdata)

	if err := self.parseRequestBody(req); err != nil {
		xerr = err
		return
	}

	for key, mfiles := range req.MultipartForm.File {
		dataimage, err := mfiles[0].Open()
		if err != nil {
			xdata[key] = map[string]string{
				"error": err.Error(),
			}
			continue
		}
		imageout, err := self.saveImageToStorage(dataimage)
		if err != nil {
			xdata[key] = map[string]string{
				"error": err.Error(),
			}
			continue
		}
		xdata[key] = imageout
	}
}
