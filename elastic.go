package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/pkg/errors"
)

func mustInitElasticsearch() *elasticsearch.Client {
	esConfig := os.Getenv("ESCONFIG")
	if len(esConfig) == 0 {
		log.Fatalln("missing ESCONFIG environment variable \n(e.g. ESCONFIG=\"user;pass;http://localhost:9200\" or \n  ESCONFIG=\"user;pass;http://localhost:9200;http://localhost:9201\")")
	}
	cfgParams := strings.Split(esConfig, ";")
	if len(cfgParams) < 3 {
		log.Fatalln("missing param in ESCONFIG: (e.g. ESCONFIG=\"user;pass;http://localhost:9200\")")
	}

	cfg := elasticsearch.Config{
		Addresses: cfgParams[2:],
		Username:  cfgParams[0],
		Password:  cfgParams[1],
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: 10 * time.Second,
			DialContext:           (&net.Dialer{Timeout: time.Second}).DialContext,
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS11,
				// ...
			},
		},
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("Error creating the elasticsearch client: %s", err)
	}
	// res, err := es.Info()
	// if err != nil {
	// 	log.Fatal("Info:", err)
	// }
	// log.Println(res)
	return es
}

func getServices(es *elasticsearch.Client) ([]string, error) {
	// Build request
	var buf bytes.Buffer
	query := map[string]interface{}{
		"size": 0,
		"aggs": map[string]interface{}{
			"componentNames": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "componentname.keyword",
					"size":  100,
				},
			},
		},
	}
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, errors.Wrap(err, "encoding elasticsearch query")
	}
	// Perform the search request.
	res, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex("france-grille-*"),
		es.Search.WithBody(&buf),
		// es.Search.WithTrackTotalHits(true),
		// es.Search.WithPretty(),
	)
	if err != nil {
		return nil, errors.Wrap(err, "elasticsearch response")
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return nil, errors.Wrap(err, "parsing elasticsearch error response")
		}
		return nil, errors.Errorf("elasticsearch: [%s] %s: %s",
			res.Status(),
			e["error"].(map[string]interface{})["type"],
			e["error"].(map[string]interface{})["reason"])
	}

	var r map[string]interface{}

	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, errors.Wrap(err, "parsing elasticsearch response")
	}

	bucket := r["aggregations"].(map[string]interface{})["componentNames"].(map[string]interface{})["buckets"].([]interface{})
	services := make([]string, len(bucket))
	for i := range bucket {
		services[i] = bucket[i].(map[string]interface{})["key"].(string)
	}
	sort.Strings(services)
	return services, nil
}

func getMessages(es *elasticsearch.Client, service string, n int) ([]interface{}, error) {
	// Build request
	var buf bytes.Buffer
	query := map[string]interface{}{
		"size": n,
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"componentname.keyword": service,
			},
		},
		"sort": []interface{}{
			map[string]interface{}{
				"utime": map[string]interface{}{
					"order": "desc",
				},
			},
		},
	}
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, errors.Wrap(err, "encoding elasticsearch query")
	}
	// Perform the search request.
	res, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex("france-grille-*"),
		es.Search.WithBody(&buf),
		// es.Search.WithPretty(),
	)
	if err != nil {
		return nil, errors.Wrap(err, "elasticsearch response")
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return nil, errors.Wrap(err, "parsing elasticsearch error response")
		}
		return nil, errors.Errorf("elasticsearch: [%s] %s: %s",
			res.Status(),
			e["error"].(map[string]interface{})["type"],
			e["error"].(map[string]interface{})["reason"])
	}

	var r map[string]interface{}

	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, errors.Wrap(err, "parsing elasticsearch response")
	}

	hits := r["hits"].(map[string]interface{})["hits"].([]interface{})
	messages := make([]interface{}, len(hits))
	for i := range hits {
		message := hits[i].(map[string]interface{})["_source"].(map[string]interface{})
		delete(message, "@timestamp")
		delete(message, "@version")
		delete(message, "port")
		delete(message, "componentindex")
		delete(message, "utime")
		host := message["host"].(string)
		pos := strings.Index(host, ".")
		if pos != -1 {
			host = host[:pos]
		}
		message["host"] = host
		messages[n-i-1] = message
	}

	return messages, nil
}
