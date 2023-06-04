package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/mustafa-mun/chirpy-bootdev/internal/bcrypt"
	"github.com/mustafa-mun/chirpy-bootdev/internal/database"
	"github.com/mustafa-mun/chirpy-bootdev/internal/sys"
)

type apiConfig struct {
	fileserverHits int
}

func main() {

	sys.EnableDebugMode()
	sys.LoadDotenv()
	jwtSecret := os.Getenv("JWT_SECRET")
	fmt.Println(jwtSecret)
	initDB()
	
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
	apiRouter.Get("/chirps", getChirpsHandler)
	apiRouter.Get("/chirps/{chirpId}", getSingleChirpHandler)
	apiRouter.Post("/chirps", postChirpHandler)
	apiRouter.Post("/users", postUserHandler)
	apiRouter.Post("/login", loginHandler)
	adminRouter.Get("/metrics", apiCfg.metricsHandler)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: corsMux,
	}
	server.ListenAndServe()
}



// Create new database
var db *database.DB

func initDB() {
	// Initialize the database connection
	var err error
	db, err = database.NewDB("database.json")
	if err != nil {
			log.Fatal(err.Error())
	}
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

func getChirpsHandler(w http.ResponseWriter, r *http.Request) {
	// Get all chirps
	chirps, err := db.GetChirps()
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "an error occurred when getting chirps")
		return
	}
	// send chirps with JSON
	respondWithJSON(w, http.StatusOK, chirps)
}

func getSingleChirpHandler(w http.ResponseWriter, r *http.Request) {
	// get chirpId from url parameter
	id := chi.URLParam(r, "chirpId")
	
	// Read database file
	structure, err := db.LoadDB()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	intId, err := strconv.Atoi(id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	chirp, ok := structure.Chirps[intId]
  // chirp not found
	if !ok {
		respondWithError(w, http.StatusNotFound, "not found")
	} 
  // chirp found
	respondWithJSON(w, http.StatusOK, chirp)

}

func postChirpHandler(w http.ResponseWriter, r *http.Request) {
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

	// Create new Chirp with database package

	type returnVals struct {
		Id int `json:"id"`
		Body string `json:"body"`
	}

	// validate the request body
	badWords := []string{"kerfuffle", "sharbert", "fornax"}
	reqBody, err := validateReqBody(params.Body, badWords)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Create and save the new chirp
	newChirp, err := db.CreateChirp(reqBody)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Return new chirp as a json
	respBody := returnVals{
			Id: newChirp.ID,
			Body: newChirp.Body,
	}
	respondWithJSON(w, http.StatusCreated, respBody)
}


func postUserHandler(w http.ResponseWriter, r *http.Request) {
	// First decode the json request body
	type parameters struct {
		// these tags indicate how the keys in the JSON should be mapped to the struct fields
		// the struct fields must be exported (start with a capital letter) if you want them parsed
		Password string `json:"password"`
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		// handle decode parameters error 
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	// Create new User with database package

	// Create and save the new user
	newUser, err := db.CreateUser(params.Password, params.Email)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	type returnVals struct {
		Id int `json:"id"`
		Email string `json:"email"`
	}

	// Return new user as a json
	respBody := returnVals{
			Id: newUser.ID,
			Email: newUser.Email,
	}
	respondWithJSON(w, http.StatusCreated, respBody)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	// First decode the json request body
	type parameters struct {
		// these tags indicate how the keys in the JSON should be mapped to the struct fields
		// the struct fields must be exported (start with a capital letter) if you want them parsed
		Password string `json:"password"`
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		// handle decode parameters error 
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	// Find the user by email 
	usr := (*database.User)(nil)

	// Read database file
	structure, err := db.LoadDB()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	users := structure.Users

	for _, user := range users {
		// If user is found, set usr variable to user
		if user.Email == params.Email {
			usr = &user
		}
	}

	// If usr is nil, user is not found
	if usr == nil {
		respondWithError(w, http.StatusNotFound, "user not found")
		return
	}

	// User is found, check the password 
	err = bcrypt.CompareHashPassword(usr.Password, params.Password)

	if err != nil {
		// Password is wrong
		respondWithError(w, http.StatusUnauthorized, "passwords do not match")
		return
	}

	// Password is true, return the user (without the password)
	type returnVals struct {
		Id int `json:"id"`
		Email string `json:"email"`
	}

	respBody := returnVals{
			Id: usr.ID,
			Email: usr.Email,
	}
	respondWithJSON(w, http.StatusOK, respBody)
}



func validateReqBody(str string, badWords []string) (string, error){
	lowered_str := strings.ToLower(str)
	for _, word := range badWords {
		if strings.Contains(lowered_str, word) {
			return "", errors.New("request body includes bad words") 
		}
	}
	return str, nil
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
