package tinynfs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
)

type HttpServer struct {
	fs     *FileSystem
	apiSrv net.Listener
}

func (self *HttpServer) startApi() {
	var (
		serveMux = http.NewServeMux()
		server   = &http.Server{
			Handler: serveMux,
		}
	)
	serveMux.HandleFunc("/get", self.handleApiGet)
	serveMux.HandleFunc("/upload", self.handleApiUpload)
	err := server.Serve(self.apiSrv)
	if err != nil {
		fmt.Println(err)
	}
}

func (self *HttpServer) Close() {
	if self.apiSrv != nil {
		self.apiSrv.Close()
	}
}

func (self *HttpServer) getHttpStatusCode(err error) int {
	if err == os.ErrPermission {
		return http.StatusForbidden
	} else if err == os.ErrNotExist {
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}

func (self *HttpServer) httpSendByteData(res http.ResponseWriter, req *http.Request, err *error, mime *string, data *[]byte) {
	if *err != nil {
		statusCode := self.getHttpStatusCode(*err)
		http.Error(res, http.StatusText(statusCode), statusCode)
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
	result := map[string]interface{}{}
	if *err == nil {
		result["code"] = 0
		result["data"] = data
	} else {
		result["code"] = 1
		result["message"] = (*err).Error()
		res.WriteHeader(self.getHttpStatusCode(*err))
	}
	res.Header().Set("Content-Type", "application/json;charset=utf-8")
	json.NewEncoder(res).Encode(result)
}

func (self *HttpServer) handleApiGet(res http.ResponseWriter, req *http.Request) {
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

	filename := req.FormValue("filename")
	if !strings.HasPrefix(filename, "/") || strings.HasSuffix(filename, "/") {
		xerr = ErrHttpParam
		return
	}

	filemime, filedata, err := self.fs.ReadFile(filename)
	if err != nil {
		xerr = err
		return
	}
	xmime = filemime
	xdata = filedata
}

func (self *HttpServer) handleApiUpload(res http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		http.Error(res, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var (
		xerr  error
		xdata = map[string]interface{}{}
	)
	defer self.httpSendJsonData(res, req, &xerr, xdata)

	filename := req.FormValue("filename")
	if !strings.HasPrefix(filename, "/") || strings.HasSuffix(filename, "/") {
		xerr = ErrHttpParam
		return
	}

	datafile, dataheader, err := req.FormFile("filedata")
	if err != nil {
		xerr = ErrHttpParam
		return
	}
	filedata, err := ioutil.ReadAll(datafile)
	if err != nil {
		xerr = err
		return
	}
	filemime := dataheader.Header.Get("Content-Type")
	err = self.fs.WriteFile(filename, filemime, filedata)
	if err != nil {
		xerr = err
		return
	}
	xdata["size"] = len(filedata)
	xdata["mime"] = filemime
	xdata["filename"] = filename
}

func NewHttpServer(fs *FileSystem, ipaddr string) (srv *HttpServer, err error) {
	srv = &HttpServer{
		fs: fs,
	}

	if srv.apiSrv, err = net.Listen("tcp4", ipaddr); err != nil {
		return nil, err
	}

	go srv.startApi()
	return srv, nil
}