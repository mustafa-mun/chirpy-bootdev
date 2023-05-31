package main

import (
	"net/http"
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
