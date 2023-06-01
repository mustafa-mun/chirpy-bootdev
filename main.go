package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"text/template"

	"github.com/go-chi/chi/v5"
)

type apiConfig struct {
	fileserverHits int
}

func main() {
	port := "8080"
	filepathRoot := "."
	r := chi.NewRouter()
	apiRouter := chi.NewRouter()
	adminRouter := chi.NewRouter()
	corsMux := middlewareCors(r)
	apiCfg := &apiConfig{fileserverHits: 0}

	fsHandler := apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot))))
	r.Handle("/app", fsHandler)
	r.Handle("/app/*", fsHandler)
	r.Mount("/api", apiRouter)
	r.Mount("/admin", adminRouter)


	apiRouter.Get("/healthz", healthzHandler)
	apiRouter.Post("/validate_chirp", validateChirpHandler)
	adminRouter.Get("/metrics", apiCfg.metricsHandler)

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
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		cfg.fileserverHits += 1
		next.ServeHTTP(w, r)
	})
}

type Context struct {
	Hits int
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	const doc = `
	<html>

	<body>
			<h1>Welcome, Chirpy Admin</h1>
			<p>Chirpy has been visited {{.Hits}} times!</p>
	</body>
	
	</html>
	`

	w.Header().Add("Content Type", "text/html")
	// The template name "template" does not matter here
	templates := template.New("template")
	// "doc" is the constant that holds the HTML content
	templates.New("doc").Parse(doc)
	context := Context{
		Hits: cfg.fileserverHits,
	}
  templates.Lookup("doc").Execute(w, context)
}

func validateChirpHandler(w http.ResponseWriter, r *http.Request) {
	// First decode the json request body
	type parameters struct {
		// these tags indicate how the keys in the JSON should be mapped to the struct fields
		// the struct fields must be exported (start with a capital letter) if you want them parsed
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		// handle decode parameters error 
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	if len(params.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	// Chirp is valid

	type returnVals struct {
		Cleaned_body string `json:"cleaned_body"`
	}

	// clean the body 
	badWords := []string{"kerfuffle", "sharbert", "fornax"}
	cleaned_str := cleanString(params.Body, badWords)


	respBody := returnVals{
			Cleaned_body: cleaned_str,
	}
	respondWithJSON(w, 200, respBody)
}


func cleanString(str string, badWords []string) string{
	strSlice := strings.Split(str, " ")

	for i := 0; i < len(badWords); i++ {
		for j := 0; j < len(strSlice); j++ {
			lowered_word := strings.ToLower(strSlice[j])
			if badWords[i] == lowered_word {
				censoredWord := "****"
				strSlice[j] = censoredWord
			}
		}
	}
	return strings.Join(strSlice, " ")
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	dat, err := json.Marshal(payload)
	if err != nil {
	w.WriteHeader(500)
	return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func respondWithError(w http.ResponseWriter, code int, errorStr string) {
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

