package tinynfs

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func (self *HttpServer) startFile() {
	var (
		serveMux = http.NewServeMux()
		server   = &http.Server{
			Handler: serveMux,
		}
	)
	serveMux.HandleFunc("/get", self.handleFileGet)
	serveMux.HandleFunc("/upload", self.handleFileUpload)
	serveMux.HandleFunc("/delete", self.handleFileDelete)
	serveMux.HandleFunc("/admin/snapshot", self.handleAdminSnapshot)
	err := server.Serve(self.fileListener)
	if err != nil && !self.closed {
		fmt.Println(err)
	}
}

func (self *HttpServer) handleFileGet(res http.ResponseWriter, req *http.Request) {
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

	filepath := req.FormValue("filepath")
	if !strings.HasPrefix(filepath, "/") || strings.HasSuffix(filepath, "/") {
		xerr = ErrParam
		return
	}

	filemime, _, filedata, err := self.storage.ReadFile(filepath)
	if err != nil {
		xerr = err
		return
	}
	xmime = filemime
	xdata = filedata
}

func (self *HttpServer) handleFileUpload(res http.ResponseWriter, req *http.Request) {
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

	datafile, dataheader, err := req.FormFile("filedata")
	if err != nil {
		xerr = ErrParam
		return
	}
	filedata, err := ioutil.ReadAll(datafile)
	if err != nil {
		xerr = err
		return
	}
	filemime := dataheader.Header.Get("Content-Type")
	err = self.storage.WriteFile(filepath, filemime, "", filedata, &WriteOptions{
		Overwrite: req.Method == "PUT",
	})
	if err != nil {
		xerr = err
		return
	}
	xdata["size"] = len(filedata)
	xdata["mime"] = filemime
	xdata["filepath"] = filepath
}

func (self *HttpServer) handleFileDelete(res http.ResponseWriter, req *http.Request) {
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

	filepath := req.FormValue("filepath")
	if !strings.HasPrefix(filepath, "/") || strings.HasSuffix(filepath, "/") {
		xerr = ErrParam
		return
	}

	err := self.storage.DeleteFile(filepath)
	if err != nil {
		xerr = err
		return
	}
	xdata["filepath"] = filepath
}

func (self *HttpServer) handleAdminSnapshot(res http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		http.Error(res, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var (
		xerr  error
		xdata = map[string]interface{}{}
	)
	defer self.sendJsonData(res, req, &xerr, xdata)

	ssfile, err := self.storage.Snapshot(true)
	if err != nil {
		xerr = err
		return
	}
	xdata["filename"] = ssfile
}
