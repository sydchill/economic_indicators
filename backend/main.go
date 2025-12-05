package main

import (
	"economic_indicator/api"
	"economic_indicator/config"
	"economic_indicator/db"
	"log"
	"net/http"
)

func main() {
	cfg := config.Load()

	bunDB := db.Open(cfg.DBDSN)

	apiServer := api.New(bunDB)
	router := apiServer.Router()

	log.Printf("backend listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, router); err != nil {
		log.Fatal(err)
	}
}
