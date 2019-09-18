package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gowww/router"
)

var es = mustInitElasticsearch()

func getServices(w http.ResponseWriter, r *http.Request) {
	// request list of services to
	services, err := getServicNames(es)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(services); err != nil {
		log.Println("error encode response:", err)
	}
}

func main() {

	rt := router.New()

	rt.Get("/api/v0/services", http.HandlerFunc(getServices))

	// serve static files in the ./www subdirectory
	rt.Get("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("www"))))

	// this is just for the index.html file in ./www
	rt.Get("/", http.FileServer(http.Dir("www")))

	log.Println("Listening URL: http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", rt))

}
