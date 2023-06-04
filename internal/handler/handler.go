package handler

import (
	"strings"
	"errors"
	"encoding/json"
	"net/http"

)

// Add CORS headers to response
func MiddlewareCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func ValidateReqBody(str string, badWords []string) (string, error){
	lowered_str := strings.ToLower(str)
	for _, word := range badWords {
		if strings.Contains(lowered_str, word) {
			return "", errors.New("request body includes bad words") 
		}
	}
	return str, nil
}

func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	dat, err := json.Marshal(payload)
	if err != nil {
	w.WriteHeader(500)
	return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func RespondWithError(w http.ResponseWriter, code int, errorStr string) {
	type returnVals struct {
		Error string `json:"error"`
	}
	respBody := returnVals{
		Error: errorStr,
	}
	dat, err := json.Marshal(respBody)
	if err != nil {
	w.WriteHeader(500)
	return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}
