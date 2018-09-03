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
	"os"
	"regexp"
	"strconv"
	"strings"
)

func init() {
	image.RegisterFormat("gif", "gif", gif.Decode, gif.DecodeConfig)
	image.RegisterFormat("png", "png", png.Decode, png.DecodeConfig)
	image.RegisterFormat("jpeg", "jpeg", jpeg.Decode, jpeg.DecodeConfig)
}

var (
	imageScaler = draw.NearestNeighbor
)

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

func (self *HttpServer) parseThumbnailSize(size string) (int, int) {
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
	defer self.httpSendByteData(res, req, &xerr, &xmime, &xdata)

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
		width, height := self.parseThumbnailSize(size)
		if width == 0 || height == 0 {
			xerr = ErrThumbnailSize
			return
		}
		originpath = filepath[:n]
		askwidth = int(width)
		askheight = int(height)
	}

	filemime, metadata, filedata, err = self.storage.ReadFile(filepath)
	if err == nil {
		xmime = filemime
		xdata = filedata
		return
	} else if err != os.ErrNotExist || len(originpath) < 1 {
		xerr = err
		return
	}

	filemime, metadata, filedata, err = self.storage.ReadFile(originpath)
	if err != nil {
		xerr = err
		return
	}

	originwidth, originheight := self.parseThumbnailSize(metadata)
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

	reader := bytes.NewReader(filedata)
	origin, _, err := image.Decode(reader)
	if err != nil {
		xerr = ErrMediaType
		return
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

	target := image.NewRGBA(image.Rect(0, 0, fixwidth, fixheight))
	imageScaler.Scale(target, target.Bounds(), origin, origin.Bounds(), draw.Over, nil)

	buffer := bytes.NewBuffer(nil)
	if filemime == "image/jpeg" {
		jopt := &jpeg.Options{
			Quality: 70,
		}
		err = jpeg.Encode(buffer, target, jopt)
		if err != nil {
			xerr = err
			return
		}
	} else {
		err = png.Encode(buffer, target)
		if err != nil {
			xerr = err
			return
		}
		filemime = "image/png"
	}

	options := &WriteOptions{
		Overwrite: false,
	}
	imagedata := buffer.Bytes()
	err = self.storage.WriteFileEx(fmt.Sprintf("%s_%dx%d", originpath, askwidth, askheight), filemime, fmt.Sprintf("%dx%d", fixwidth, fixheight), imagedata, options)
	if err != nil && err != os.ErrExist {
		xerr = err
		return
	}
	xmime = filemime
	xdata = imagedata
}

func (self *HttpServer) storeImageToFile(filepath string, dataimage io.Reader, options *WriteOptions) (map[string]interface{}, error) {
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
		randtext := RandHex(10)
		filepath = self.config.ImageFilePath + randtext[0:2] + "/" + randtext[2:4] + "/" + randtext[5:] + TimeHex(1)
	}
	err = self.storage.WriteFileEx(filepath, strings.ToLower("image/"+format), fmt.Sprintf("%dx%d", config.Width, config.Height), imagedata, options)
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
	if req.Method != "POST" {
		http.Error(res, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var (
		xerr  error
		xdata = map[string]interface{}{}
	)
	defer self.httpSendJsonData(res, req, &xerr, xdata)

	dataimage, _, err := req.FormFile("imagedata")
	if err != nil {
		xerr = ErrParam
		return
	}
	imageout, err := self.storeImageToFile("", dataimage, nil)
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
	defer self.httpSendJsonData(res, req, &xerr, xdata)

	if err := req.ParseMultipartForm(32 * 1024 * 1024); err != nil {
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
		imageout, err := self.storeImageToFile("", dataimage, nil)
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
	if req.Method != "POST" {
		http.Error(res, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var (
		xerr  error
		xdata = map[string]interface{}{}
	)
	defer self.httpSendJsonData(res, req, &xerr, xdata)

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
	imageout, err := self.storeImageToFile(filepath, dataimage, &WriteOptions{
		Overwrite: false,
	})
	if err != nil {
		xerr = err
		return
	}
	for k, v := range imageout {
		xdata[k] = v
	}
}
