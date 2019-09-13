package main

import (
	"log"
	"net/http"

	"github.com/gowww/router"
)

func main() {
	rt := router.New()

	// serve static files in the ./www subdirectory
	rt.Get("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("www"))))

	// this is just for the index.html file in ./www
	rt.Get("/", http.FileServer(http.Dir("www")))

	log.Println("Listening URL: https://localhost:4343")
	log.Fatal(http.ListenAndServeTLS(":4343", "cert.pem", "key.pem", rt))

}
