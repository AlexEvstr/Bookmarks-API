package main

import (
	"log"
	"net/http"
	"time"

	bookmarks "bookmarks-api"
)

func main() {
	store := bookmarks.NewStore()
	handler := bookmarks.NewHandler(store)
	server := &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("bookmarks API listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}
