package tinynfs

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
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

func (self *HttpServer) getImageFilePath() string {
	token := make([]byte, 10)
	rand.Read(token)
	return self.config.ImageFilePath + fmt.Sprintf("%x%x", token, time.Now().Unix())
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

	width, height, format, imagedata, err := ImageScaleBuffer(imagedata, awidth, aheight)
	if err != nil {
		xerr = err
		return
	}

	mimedata = "image/" + format
	metadata = fmt.Sprintf("%dx%d", width, height)
	filepath = fmt.Sprintf("%s_%dx%d", originpath, awidth, aheight)
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

func (self *HttpServer) saveImageToStorage(stream io.Reader) (map[string]interface{}, error) {
	imagedata, err := ioutil.ReadAll(stream)
	if err != nil {
		return nil, err
	}

	width, height, format, imagedata, err := ImageParseBuffer(imagedata, self.config.ImageOtimizeSide, self.config.ImageOtimizeSize)
	if err != nil {
		return nil, err
	}

	mimedata := "image/" + format
	metadata := fmt.Sprintf("%dx%d", width, height)
	filepath := self.getImageFilePath()
	if err := self.storage.WriteFile(filepath, mimedata, metadata, imagedata, nil); err != nil {
		return nil, err
	}

	imageout := map[string]interface{}{}
	imageout["size"] = len(imagedata)
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
