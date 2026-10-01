package main

import (
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "listen address")
	flag.Parse()
	sms, err := NewSMSClient()
	if err != nil {
		log.Fatal(err)
	}
	service := AccessService{SMS: sms}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /access/start", func(w http.ResponseWriter, r *http.Request) {
		var access AssetAccess
		if err := json.NewDecoder(r.Body).Decode(&access); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		reply, err := service.Start(r.Context(), access)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, reply)
	})
	mux.HandleFunc("POST /access/verify", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			AssetAccess
			Code string `json:"code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		decision, err := service.Verify(r.Context(), input.AssetAccess, input.Code)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, decision)
	})
	log.Printf("creator access service listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}

func writeServiceError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		status = apiErr.HTTPStatus
		if status < 400 || status >= 500 {
			status = http.StatusBadGateway
		}
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write response: %v", err)
	}
}
