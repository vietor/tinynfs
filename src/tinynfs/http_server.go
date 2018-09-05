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

func (self *HttpServer) asHttpStatusCode(err error) int {
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

func (self *HttpServer) sendByteData(res http.ResponseWriter, req *http.Request, err *error, mime *string, data *[]byte) {
	if *err != nil {
		statusCode := self.asHttpStatusCode(*err)
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

func (self *HttpServer) sendJsonData(res http.ResponseWriter, req *http.Request, err *error, data map[string]interface{}) {
	res.Header().Set("Content-Type", "application/json;charset=utf-8")
	result := map[string]interface{}{}
	if *err == nil {
		result["code"] = 0
		result["data"] = data
	} else {
		result["code"] = 1
		result["error"] = (*err).Error()
		res.WriteHeader(self.asHttpStatusCode(*err))
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
