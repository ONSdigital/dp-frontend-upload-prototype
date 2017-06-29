package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"

	"github.com/ONSdigital/dp-frontend-upload-prototype/config"
	"github.com/ONSdigital/go-ns/log"
	"github.com/ONSdigital/go-ns/server"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
)

var uploadStates map[string]map[int]bool

func main() {
	cfg := config.Get()

	uploadStates = make(map[string]map[int]bool)

	r := mux.NewRouter()
	r.Path("/").Methods("GET").HandlerFunc(landing)
	r.Path("/upload").Methods("POST").HandlerFunc(upload)
	r.Path("/upload").Methods("GET").HandlerFunc(checkUploaded)
	r.Path("/{uri:.*}").Handler(http.FileServer(http.Dir(".")))

	s := server.New(cfg.BindAddr, r)

	if err := s.ListenAndServe(); err != nil {
		log.Error(err, nil)
	}
}

func checkUploaded(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()

	resum := new(Resumable)

	if err := schema.NewDecoder().Decode(resum, req.Form); err != nil {
		log.ErrorR(req, err, nil)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if uploadStates[resum.FileName][resum.ChunkNumber] {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func landing(w http.ResponseWriter, req *http.Request) {
	t := template.New("landing").Funcs(template.FuncMap{
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"last": func(x int, a interface{}) bool {
			return x == reflect.ValueOf(a).Len()-1
		},
	})

	template.Must(t.ParseFiles("templates/landing.tmpl"))
	template.Must(t.ParseFiles("templates/header.tmpl", "templates/footer.tmpl"))

	if err := t.ExecuteTemplate(w, "landing", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}

// Resumable ...
type Resumable struct {
	ChunkNumber      int    `schema:"resumableChunkNumber"`
	ChunkSize        int    `schema:"resumableChunkSize"`
	CurrentChunkSize int    `schema:"resumableCurrentChunkSize"`
	TotalSize        int    `schema:"resumableTotalSize"`
	Type             string `schema:"resumableType"`
	Identifier       string `schema:"resumableIdentifier"`
	FileName         string `schema:"resumableFilename"`
	RelativePath     string `schema:"resumableRelativePath"`
	TotalChunks      int    `schema:"resumableTotalChunks"`
}

func upload(w http.ResponseWriter, req *http.Request) {

	req.ParseForm()

	resum := new(Resumable)

	if err := schema.NewDecoder().Decode(resum, req.Form); err != nil {
		log.ErrorR(req, err, nil)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer req.Body.Close()

	if uploadStates[resum.FileName] == nil {
		uploadStates[resum.FileName] = make(map[int]bool)
	}

	fu, _, err := req.FormFile("file")
	if err != nil {
		fmt.Println(err)
		return
	}

	b, err := ioutil.ReadAll(fu)
	if err != nil {
		fmt.Println(err)
		return
	}

	f, err := os.OpenFile(resum.FileName, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	// need to ensure that the chunk is written to the file in correct position
	f.Seek(int64((resum.ChunkNumber-1)*resum.ChunkSize), 0)

	f.Write(b)

	uploadStates[resum.FileName][resum.ChunkNumber] = true

	log.Debug("resumable", log.Data{"resum": resum})
}
