package main

import (
	"net/http"

	"github.com/ONSdigital/dp-frontend-upload-prototype/config"
	"github.com/ONSdigital/dp-frontend-upload-prototype/handlers"
	"github.com/ONSdigital/go-ns/log"
	"github.com/gorilla/mux"
)

func main() {
	cfg := config.Get()

	r := mux.NewRouter()
	r.Path("/").Methods("GET").HandlerFunc(handlers.Landing)
	r.Path("/upload").Methods("POST").HandlerFunc(handlers.Upload)
	r.Path("/upload").Methods("GET").HandlerFunc(handlers.CheckUploaded)
	r.Path("/{uri:.*}").Handler(http.FileServer(http.Dir(".")))

	log.Debug("listening...", log.Data{
		"address":    cfg.BindAddr,
		"access_key": cfg.AWSAccessKey,
		"secret_key": cfg.AWSSecretKey,
	})

	if err := http.ListenAndServe(":3000", r); err != nil {
		log.Error(err, nil)
	}
}
