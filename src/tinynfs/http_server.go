package tinynfs

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
)

type HttpServer struct {
	closed        bool
	config        *Network
	storage       *FileSystem
	fileListener  net.Listener
	imageListener net.Listener
}

func (self *HttpServer) Close() {
	self.closed = true
	if self.fileListener != nil {
		self.fileListener.Close()
	}
	if self.imageListener != nil {
		self.imageListener.Close()
	}
}

func (self *HttpServer) sendByteData(res http.ResponseWriter, req *http.Request, err *error, mime *string, data *[]byte) {
	if *err != nil {
		statusCode := toStatusCode(*err)
		http.Error(res, (*err).Error(), statusCode)
	} else {
		header := res.Header()
		if len(*mime) > 0 {
			header.Set("Content-Type", *mime)
		} else {
			header.Set("Content-Type", "application/octet-stream")
		}
		header.Set("Content-Length", strconv.Itoa(len(*data)))
		res.Write(*data)
	}
}

func (self *HttpServer) sendJsonData(res http.ResponseWriter, req *http.Request, err *error, data map[string]interface{}) {
	res.Header().Set("Content-Type", "application/json;charset=utf-8")
	result := map[string]interface{}{}
	if *err == nil {
		result["code"] = 0
		result["data"] = data
	} else {
		result["code"] = toErrorCode(*err)
		result["error"] = (*err).Error()
		res.WriteHeader(toStatusCode(*err))
	}
	json.NewEncoder(res).Encode(result)
}

func (self *HttpServer) parseRequestBody(req *http.Request) error {
	if err := req.ParseMultipartForm(32 * 1024 * 1024); err != nil {
		return err
	}
	return nil
}

func NewHttpServer(storage *FileSystem, config *Network) (*HttpServer, error) {
	fileListener, err := net.Listen(config.Tcp, config.FileBind)
	if err != nil {
		return nil, err
	}
	imageListener, err := net.Listen(config.Tcp, config.ImageBind)
	if err != nil {
		fileListener.Close()
		return nil, err
	}

	srv := &HttpServer{
		config:        config,
		storage:       storage,
		fileListener:  fileListener,
		imageListener: imageListener,
	}

	go srv.startFile()
	go srv.startImage()
	return srv, nil
}
