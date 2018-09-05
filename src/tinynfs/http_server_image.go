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

func resizeImage(filemime string, filedata []byte, originwidth int, originheight int, askwidth int, askheight int) (string, []byte, int, int, error) {
	reader := bytes.NewReader(filedata)
	origin, _, err := image.Decode(reader)
	if err != nil {
		return "", nil, 0, 0, ErrMediaType
	}

	// Calculate target image size
	fixwidth := originwidth
	fixheight := originheight
	if fixwidth > fixheight {
		if fixwidth > askwidth {
			fixheight = int(math.Floor(float64(askwidth) * float64(fixheight) / float64(fixwidth)))
			fixwidth = askwidth
		}
	} else if fixwidth < fixheight {
		if fixheight > askheight {
			fixwidth = int(math.Floor(float64(askheight) * float64(fixwidth) / float64(fixheight)))
			fixheight = askheight
		}
	} else {
		side := askwidth
		if askwidth > askheight {
			side = askheight
		}
		fixwidth = side
		fixheight = side
	}

	// Scale the image
	target := image.NewRGBA(image.Rect(0, 0, fixwidth, fixheight))
	scaler := draw.ApproxBiLinear
	if fixwidth*5 < originwidth || fixheight*5 < originheight {
		scaler = draw.BiLinear
	}
	scaler.Scale(target, target.Bounds(), origin, origin.Bounds(), draw.Over, nil)
	buffer := bytes.NewBuffer(nil)
	if filemime == "image/jpeg" {
		jopt := &jpeg.Options{
			Quality: 70,
		}
		err = jpeg.Encode(buffer, target, jopt)
		if err != nil {
			return "", nil, 0, 0, err
		}
	} else { // Gif to png
		err = png.Encode(buffer, target)
		if err != nil {
			return "", nil, 0, 0, err
		}
		filemime = "image/png"
	}

	return filemime, buffer.Bytes(), fixwidth, fixheight, nil
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
	serveMux.HandleFunc("/upload/file", self.handleImageUploadFile)
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
	if req.Method != "GET" {
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
		askwidth   int
		askheight  int
		originpath string
		filemime   string
		filedata   []byte
		metadata   string
	)

	if m, _ := regexp.MatchString("_[0-9]+x[0-9]+$", filepath); m {
		n := strings.LastIndex(filepath, "_")
		size := filepath[n+1:]
		if _, ok := self.config.ImageThumbnailSizes[size]; !ok {
			xerr = ErrThumbnailSize
			return
		}
		width, height := self.parseImageSize(size)
		if width == 0 || height == 0 {
			xerr = ErrThumbnailSize
			return
		}
		originpath = filepath[:n]
		askwidth = int(width)
		askheight = int(height)
	}

	// Read thumbnail file
	filemime, metadata, filedata, err = self.storage.ReadFile(filepath)
	if err == nil {
		xmime = filemime
		xdata = filedata
		return
	} else if err != ErrNotExist || len(originpath) < 1 {
		xerr = err
		return
	}

	// Read origin file
	filemime, metadata, filedata, err = self.storage.ReadFile(originpath)
	if err != nil {
		xerr = err
		return
	}
	originwidth, originheight := self.parseImageSize(metadata)
	if originwidth == 0 || originheight == 0 {
		xerr = ErrThumbnailSize
		return
	}
	// Ignore image scale
	if originwidth < askwidth && originheight < askheight {
		xmime = filemime
		xdata = filedata
		return
	}

	// Create thumbnail image
	imagemime, imagedata, fixwidth, fixheight, err := resizeImage(filemime, filedata, originwidth, originheight, askwidth, askheight)
	if err != nil {
		xerr = err
		return
	}
	filepath = fmt.Sprintf("%s_%dx%d", originpath, askwidth, askheight)
	metadata = fmt.Sprintf("%dx%d", fixwidth, fixheight)
	woptions := &WriteOptions{
		Overwrite: false,
	}
	err = self.storage.WriteFileEx(filepath, imagemime, metadata, imagedata, woptions)
	if err != nil && err != ErrExist {
		xerr = err
		return
	}
	xmime = filemime
	xdata = imagedata
}

func (self *HttpServer) writeImageToImage(filepath string, dataimage io.Reader, options *WriteOptions) (map[string]interface{}, error) {
	imagedata, err := ioutil.ReadAll(dataimage)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(imagedata)
	config, format, err := image.DecodeConfig(reader)
	if err != nil {
		return nil, ErrMediaType
	}
	if len(filepath) < 1 {
		filepath = self.config.ImageFilePath + RandHex(10) + TimeHex(0)
	}
	imagemime := strings.ToLower("image/" + format)
	metadata := fmt.Sprintf("%dx%d", config.Width, config.Height)
	err = self.storage.WriteFileEx(filepath, imagemime, metadata, imagedata, options)
	if err != nil {
		return nil, err
	}
	imageout := map[string]interface{}{}
	imageout["size"] = len(imagedata)
	imageout["width"] = config.Width
	imageout["height"] = config.Height
	imageout["image_url"] = filepath
	return imageout, nil
}

func (self *HttpServer) handleImageUpload(res http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" && req.Method != "PUT" {
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
	imageout, err := self.writeImageToImage("", dataimage, &WriteOptions{
		Overwrite: req.Method == "PUT",
	})
	if err != nil {
		xerr = err
		return
	}
	for k, v := range imageout {
		xdata[k] = v
	}
}

func (self *HttpServer) handleImageUploadMore(res http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" && req.Method != "PUT" {
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

	woptions := &WriteOptions{
		Overwrite: req.Method == "PUT",
	}
	for key, mfiles := range req.MultipartForm.File {
		dataimage, err := mfiles[0].Open()
		if err != nil {
			xdata[key] = map[string]string{
				"error": err.Error(),
			}
			continue
		}
		imageout, err := self.writeImageToImage("", dataimage, woptions)
		if err != nil {
			xdata[key] = map[string]string{
				"error": err.Error(),
			}
			continue
		}
		xdata[key] = imageout
	}
}

func (self *HttpServer) handleImageUploadFile(res http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" && req.Method != "PUT" {
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

	filepath := req.FormValue("filepath")
	if !strings.HasPrefix(filepath, "/") || strings.HasSuffix(filepath, "/") {
		xerr = ErrParam
		return
	}
	dataimage, _, err := req.FormFile("imagedata")
	if err != nil {
		xerr = ErrParam
		return
	}
	imageout, err := self.writeImageToImage(filepath, dataimage, &WriteOptions{
		Overwrite: req.Method == "PUT",
	})
	if err != nil {
		xerr = err
		return
	}
	for k, v := range imageout {
		xdata[k] = v
	}
}
