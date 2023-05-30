package main

import (
	"net/http"
)

func main() {
	port := "8080"
	mux := http.NewServeMux()
	corsMux := middlewareCors(mux)
	mux.Handle("/", http.FileServer(http.Dir(".")))
	server := &http.Server{
		Addr:    ":" + port,
		Handler: corsMux,
	}
	server.ListenAndServe()
}

// Add CORS headers to response
func middlewareCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
