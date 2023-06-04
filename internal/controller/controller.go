package controller

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"text/template"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mustafa-mun/chirpy-bootdev/internal/bcrypt"
	"github.com/mustafa-mun/chirpy-bootdev/internal/database"
	"github.com/mustafa-mun/chirpy-bootdev/internal/handler"
)

// Create new database
var db *database.DB

func InitDB() {
	// Initialize the database connection
	var err error
	db, err = database.NewDB("database.json")
	if err != nil {
			log.Fatal(err.Error())
	}
}

type ApiConfig struct {
	FileserverHits int
	JwtSecret string
}

func (cfg *ApiConfig) HealthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (cfg *ApiConfig) MiddlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		cfg.FileserverHits += 1
		next.ServeHTTP(w, r)
	})
}

type Context struct {
	Hits int
}

func (cfg *ApiConfig) MetricsHandler(w http.ResponseWriter, r *http.Request) {
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
		Hits: cfg.FileserverHits,
	}
  templates.Lookup("doc").Execute(w, context)
}

func (cfg *ApiConfig) GetChirpsHandler(w http.ResponseWriter, r *http.Request) {
	// Get all chirps
	chirps, err := db.GetChirps()
	if err != nil {
		handler.RespondWithError(w, http.StatusBadRequest, "an error occurred when getting chirps")
		return
	}
	// send chirps with JSON
	handler.RespondWithJSON(w, http.StatusOK, chirps)
}

func (cfg *ApiConfig) GetSingleChirpHandler(w http.ResponseWriter, r *http.Request) {
	// get chirpId from url parameter
	id := chi.URLParam(r, "chirpId")
	
	// Read database file
	structure, err := db.LoadDB()
	if err != nil {
		handler.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	intId, err := strconv.Atoi(id)
	if err != nil {
		handler.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	chirp, ok := structure.Chirps[intId]
  // chirp not found
	if !ok {
		handler.RespondWithError(w, http.StatusNotFound, "not found")
	} 
  // chirp found
	handler.RespondWithJSON(w, http.StatusOK, chirp)

}

func (cfg *ApiConfig) PostChirpHandler(w http.ResponseWriter, r *http.Request) {
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
		handler.RespondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	if len(params.Body) > 140 {
		handler.RespondWithError(w, http.StatusBadRequest, "Chirp is too long")
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
	reqBody, err := handler.ValidateReqBody(params.Body, badWords)
	if err != nil {
		handler.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Create and save the new chirp
	newChirp, err := db.CreateChirp(reqBody)

	if err != nil {
		handler.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Return new chirp as a json
	respBody := returnVals{
			Id: newChirp.ID,
			Body: newChirp.Body,
	}
	handler.RespondWithJSON(w, http.StatusCreated, respBody)	
}


func (cfg *ApiConfig) PostUserHandler(w http.ResponseWriter, r *http.Request) {
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
		handler.RespondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	// Create new User with database package

	// Create and save the new user
	newUser, err := db.CreateUser(params.Password, params.Email)

	if err != nil {
		handler.RespondWithError(w, http.StatusBadRequest, err.Error())
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
	handler.RespondWithJSON(w, http.StatusCreated, respBody)
}

func (cfg ApiConfig) LoginHandler(w http.ResponseWriter, r *http.Request) {
	// First decode the json request body
	type parameters struct {
		// these tags indicate how the keys in the JSON should be mapped to the struct fields
		// the struct fields must be exported (start with a capital letter) if you want them parsed
		Password string `json:"password"`
		Email string `json:"email"`
		ExpiresInSeconds int `json:"expires_in_seconds"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		// handle decode parameters error 
		handler.RespondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	// If expiration date is not given or bigger than 24 hours
	if params.ExpiresInSeconds == 0 || params.ExpiresInSeconds > 86400 {
		params.ExpiresInSeconds = 86400
	} 

	// Find the user by email 
	usr := (*database.User)(nil)

	// Read database file
	structure, err := db.LoadDB()
	if err != nil {
		handler.RespondWithError(w, http.StatusInternalServerError, err.Error())
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
		handler.RespondWithError(w, http.StatusNotFound, "user not found")
		return
	}

	// User is found, check the password 
	err = bcrypt.CompareHashPassword(usr.Password, params.Password)

	if err != nil {
		// Password is wrong
		handler.RespondWithError(w, http.StatusUnauthorized, "passwords do not match")
		return
	}

	// Password is true, create jwt token
	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer: "chirpy",
		IssuedAt: jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(time.Second * time.Duration(params.ExpiresInSeconds))},
		Subject: strconv.Itoa(usr.ID),
	})

	// Sign the token with secret key
	token, err := newToken.SignedString([]byte(cfg.JwtSecret))

	if err != nil {
		handler.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return logged user with JWT token
	type returnVals struct {
		Id int `json:"id"`
		Email string `json:"email"`
		Token string `json:"token"`
	}
	respBody := returnVals{
			Id: usr.ID,
			Email: usr.Email,
			Token: token,
	}

	handler.RespondWithJSON(w, http.StatusOK, respBody)
}
