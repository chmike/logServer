package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gowww/router"
)

const defaultNbrMessages = 10

var es = mustInitElasticsearch()

func apiServices(w http.ResponseWriter, r *http.Request) {
	log.Println("GET", r.RequestURI)
	// request list of services to
	services, err := getServices(es)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		log.Println("error:", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(services); err != nil {
		log.Println("error encode response:", err)
	}
}

func apiMessages(w http.ResponseWriter, r *http.Request) {
	log.Println("GET", r.RequestURI)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	service := router.Parameter(r, "*")
	n := defaultNbrMessages
	if nStr, ok := r.URL.Query()["n"]; ok {
		var err error
		n, err = strconv.Atoi(nStr[0])
		if err != nil {
			n = defaultNbrMessages
		}
	}
	messages, err := getMessages(es, service, n) // <====== Implement this
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		log.Println("error:", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(messages); err != nil {
		log.Println("error encode response:", err)
	}
}

func main() {

	rt := router.New()

	rt.Get("/api/v0/services", http.HandlerFunc(apiServices))
	rt.Get("/api/v0/msgs/", http.HandlerFunc(apiMessages))

	// serve static files in the ./www subdirectory
	rt.Get("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("www"))))

	// this is just for the index.html file in ./www
	rt.Get("/", http.FileServer(http.Dir("www")))

	log.Println("Listening URL: http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", rt))

}
