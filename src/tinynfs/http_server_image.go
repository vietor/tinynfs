package tinynfs

import (
	"bytes"
	"fmt"
	"golang.org/x/image/draw"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
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

func (self *HttpServer) parseThumbnailPath(filepath string) (string, int, int, error) {
	n := strings.LastIndex(filepath, "_")
	size := filepath[n+1:]
	if _, ok := self.config.ImageThumbnailSizes[size]; !ok {
		return "", 0, 0, ErrThumbnailSize
	}
	width, height := self.parseThumbnailSize(size)
	if width == 0 || height == 0 {
		return "", 0, 0, ErrThumbnailSize
	}
	return filepath[:n], int(width), int(height), nil
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
		originpath, askwidth, askheight, err = self.parseThumbnailPath(filepath)
		if err != nil {
			xerr = err
			return
		}
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

	imagedata := buffer.Bytes()
	err = self.storage.WriteFile(fmt.Sprintf("%s_%dx%d", originpath, askwidth, askheight), filemime, fmt.Sprintf("%dx%d", fixwidth, fixheight), imagedata)
	if err != nil {
		xerr = err
		return
	}
	xmime = filemime
	xdata = imagedata
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
	imagedata, err := ioutil.ReadAll(dataimage)
	if err != nil {
		xerr = err
		return
	}

	reader := bytes.NewReader(imagedata)
	config, format, err := image.DecodeConfig(reader)
	if err != nil {
		xerr = ErrMediaType
		return
	}

	randtext := RandHex(10)
	filepath := self.config.ImageFilePath + strings.ToUpper(randtext[0:2]+"/"+randtext[2:4]) + "/" + randtext[5:] + TimeHex(1)

	err = self.storage.WriteFile(filepath, strings.ToLower("image/"+format), fmt.Sprintf("%dx%d", config.Width, config.Height), imagedata)
	if err != nil {
		xerr = err
		return
	}
	xdata["size"] = len(imagedata)
	xdata["width"] = config.Width
	xdata["height"] = config.Height
	xdata["image_url"] = filepath
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
	imagedata, err := ioutil.ReadAll(dataimage)
	if err != nil {
		xerr = err
		return
	}

	reader := bytes.NewReader(imagedata)
	config, format, err := image.DecodeConfig(reader)
	if err != nil {
		xerr = ErrMediaType
		return
	}

	err = self.storage.WriteFile(filepath, strings.ToLower("image/"+format), fmt.Sprintf("%dx%d", config.Width, config.Height), imagedata)
	if err != nil {
		xerr = err
		return
	}
	xdata["size"] = len(imagedata)
	xdata["width"] = config.Width
	xdata["height"] = config.Height
	xdata["image_url"] = filepath
}
