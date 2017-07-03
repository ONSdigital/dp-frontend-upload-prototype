package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"

	"gopkg.in/amz.v1/aws"
	"gopkg.in/amz.v1/s3"

	"github.com/ONSdigital/dp-frontend-upload-prototype/config"
	"github.com/ONSdigital/go-ns/log"
	"github.com/gorilla/schema"
)

var uploadStates map[string]map[int]bool

func init() {
	if uploadStates == nil {
		uploadStates = make(map[string]map[int]bool)
	}
}

// Resumable represents resumable js upload query pararmeters
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

// Landing loads landing page for dp-frontend-upload-prototype
func Landing(w http.ResponseWriter, req *http.Request) {
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

// CheckUploaded checks to see if a chunk has been uploaded
func CheckUploaded(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()

	resum := new(Resumable)

	if err := schema.NewDecoder().Decode(resum, req.Form); err != nil {
		log.ErrorR(req, err, nil)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if uploadStates[resum.FileName][resum.ChunkNumber] {
		log.Debug("chunk number already uploaded", log.Data{"chunk_number": resum.ChunkNumber, "file_name": resum.FileName})
		w.WriteHeader(http.StatusOK)
	} else {
		log.Debug("chunk number not found in local cache", log.Data{"chunk_number": resum.ChunkNumber, "file_name": resum.FileName})
		w.WriteHeader(http.StatusNotFound)
	}
}

// Upload handles the uploading of a file chunk to the server
func Upload(w http.ResponseWriter, req *http.Request) {
	cfg := config.Get()

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
		log.ErrorR(req, err, nil)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	b, err := ioutil.ReadAll(fu)
	if err != nil {
		log.ErrorR(req, err, nil)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(cfg.AWSAccessKey) < 1 || len(cfg.AWSSecretKey) < 1 {
		uploadLocal(w, req, b, resum)
	} else {
		uploadToS3(w, req, b, resum)
	}

}

func uploadToS3(w http.ResponseWriter, req *http.Request, b []byte, resum *Resumable) {

	auth, err := aws.EnvAuth()
	if err != nil {
		log.ErrorR(req, err, nil)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	conn := s3.New(auth, aws.EUWest)
	buck := conn.Bucket("dp-frontend-upload-prototype")

	// No need for multi part upload - just upload directly to s3
	if resum.TotalChunks == 1 {
		err = buck.Put(resum.FileName, b, resum.Type, "public-read")
		if err != nil {
			log.ErrorR(req, err, nil)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.Info("uploaded file", log.Data{"file_name": resum.FileName, "size": resum.TotalSize})

		return
	}

	m, err := buck.Multi(resum.FileName, resum.Type, "public-read")
	if err != nil {
		log.ErrorR(req, err, nil)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if isChunkInCache(m, resum.ChunkNumber) {
		log.Info("chunk found in s3 multi cache", log.Data{
			"chunk_number": resum.ChunkNumber,
			"max_chunks":   resum.TotalChunks,
			"file_name":    resum.FileName,
		})
	} else {

		rs := bytes.NewReader(b)

		_, err = m.PutPart(resum.ChunkNumber, rs)
		if err != nil {
			log.ErrorR(req, err, nil)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.Info("chunk accepted", log.Data{
			"chunk_number": resum.ChunkNumber,
			"max_chunks":   resum.TotalChunks,
			"file_name":    resum.FileName,
		})

	}

	uploadStates[resum.FileName][resum.ChunkNumber] = true

	if len(uploadStates[resum.FileName]) == resum.TotalChunks {
		parts, err := m.ListParts()
		if err != nil {
			log.ErrorR(req, err, nil)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.Debug("number of parts in cache", log.Data{"n": len(parts)})

		// Only complete the number when all chunks are in the cache
		if len(parts) == resum.TotalChunks {
			if err := m.Complete(parts); err != nil {
				log.ErrorR(req, err, nil)
				if err := m.Abort(); err != nil {
					log.ErrorR(req, err, nil)
				}
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			log.Info("uploaded file", log.Data{"file_name": resum.FileName, "size": resum.TotalSize})

			return
		}

		fmt.Println("chunk numbers in cache: ")
		for _, p := range parts {
			fmt.Println(p.N)
		}

		log.Info("missing chunk(s) from s3 multi cache... telling client to retry upload", nil)

		// Expected to have all chunks in cache but some are missing, client will have to retry
		uploadStates[resum.FileName] = make(map[int]bool)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func uploadLocal(w http.ResponseWriter, req *http.Request, b []byte, resum *Resumable) {
	f, err := os.OpenFile(resum.FileName, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.ErrorR(req, err, nil)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// need to ensure that the chunk is written to the file in correct position
	f.Seek(int64((resum.ChunkNumber-1)*resum.ChunkSize), 0)

	if _, err := f.Write(b); err != nil {
		log.ErrorR(req, err, nil)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	uploadStates[resum.FileName][resum.ChunkNumber] = true
}

func isChunkInCache(multi *s3.Multi, n int) bool {
	parts, err := multi.ListParts()
	if err != nil {
		return false
	}

	for _, p := range parts {
		if p.N == n {
			return true
		}
	}

	return false
}
