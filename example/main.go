package main

import (
	"encoding/json"
	"log"
	"net/http"

	"go.opencensus.io/examples/exporter"
	"go.opencensus.io/trace"
	"go.opencensus.io/zpages"

	"github.com/garsue/otwgen/example/domain"
	wrapper "github.com/garsue/otwgen/example/wrapper"
)

func main() {
	ex, err := exporter.NewLogExporter(exporter.Options{})
	if err != nil {
		log.Panic(err)
	}
	trace.RegisterExporter(ex)
	zpages.Handle(http.DefaultServeMux, "/debug")

	http.HandleFunc("/example", func(w http.ResponseWriter, r *http.Request) {
		header, err := wrapper.NewService(domain.NewService()).GetContent(r.Context())
		if err != nil {
			log.Println(err)
		}
		if err := json.NewEncoder(w).Encode(header); err != nil {
			log.Println(err)
		}
	})
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
