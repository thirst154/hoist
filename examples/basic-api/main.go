package main

import (
	"encoding/json"
	"net/http"

	hoist "github.com/thirst154/hoist/sdk"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "pong",
		})
	})

	hoist.Start(mux)
}
