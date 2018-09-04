package tinynfs

import (
	"encoding/json"
	"net"
	"net/http"
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

func (self *HttpServer) getHttpStatusCode(err error) int {
	if err == ErrParam || err == ErrThumbnailSize {
		return http.StatusBadRequest
	} else if err == ErrPermission || err == ErrExist {
		return http.StatusForbidden
	} else if err == ErrNotExist {
		return http.StatusNotFound
	} else if err == ErrMediaType {
		return http.StatusUnsupportedMediaType
	}
	return http.StatusInternalServerError
}

func (self *HttpServer) httpSendByteData(res http.ResponseWriter, req *http.Request, err *error, mime *string, data *[]byte) {
	if *err != nil {
		statusCode := self.getHttpStatusCode(*err)
		http.Error(res, (*err).Error(), statusCode)
	} else {
		if len(*mime) > 0 {
			res.Header().Set("Content-Type", *mime)
		} else {
			res.Header().Set("Content-Type", "application/octet-stream")
		}
		res.Write(*data)
	}
}

func (self *HttpServer) httpSendJsonData(res http.ResponseWriter, req *http.Request, err *error, data map[string]interface{}) {
	res.Header().Set("Content-Type", "application/json;charset=utf-8")
	result := map[string]interface{}{}
	if *err == nil {
		result["code"] = 0
		result["data"] = data
	} else {
		result["code"] = 1
		result["message"] = (*err).Error()
		res.WriteHeader(self.getHttpStatusCode(*err))
	}
	json.NewEncoder(res).Encode(result)
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
