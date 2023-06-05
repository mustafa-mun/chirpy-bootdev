package controller

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
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

type ReturnUserVals struct {
	Id int `json:"id"`
	Email string `json:"email"`
	IsChirpyRed bool `json:"is_chirpy_red"`
}

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

	// Check auth
	tokenObj, er := cfg.CheckJwtToken(w, r)
	if er != nil {
		handler.RespondWithError(w, http.StatusUnauthorized, er.Error())
		return
	}
	// decode the json request body
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
		AuthorId int `json:"author_id"`
	}

	// validate the request body
	badWords := []string{"kerfuffle", "sharbert", "fornax"}
	reqBody, err := handler.ValidateReqBody(params.Body, badWords)
	if err != nil {
		handler.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get author id from JWT token
	authorId := tokenObj.Claims.(jwt.MapClaims)["sub"].(string)
	intId, err := strconv.Atoi(authorId)
	if err != nil {
		handler.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	// Create and save the new chirp
	newChirp, err := db.CreateChirp(reqBody, intId)

	if err != nil {
		handler.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Return new chirp as a json
	respBody := returnVals{
			Id: newChirp.ID,
			Body: newChirp.Body,
			AuthorId: intId,
	}
	handler.RespondWithJSON(w, http.StatusCreated, respBody)	
}

func (cfg *ApiConfig) DeleteChirpHandler(w http.ResponseWriter, r *http.Request) {
	// Check auth
	
	tokenObj, err := cfg.CheckJwtToken(w, r)

	if err != nil {
		handler.RespondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// take id from url parameter
	id := chi.URLParam(r, "chirpID")
	intId, err := strconv.Atoi(id)

	if err != nil {
		handler.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	authorId := tokenObj.Claims.(jwt.MapClaims)["sub"].(string)
	intAuthorId, err := strconv.Atoi(authorId)

	if err != nil {
		handler.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	err = db.DeleteChirp(intId, intAuthorId)

	if err != nil {
		handler.RespondWithError(w, http.StatusForbidden, err.Error())
		return
	}

	type returnVals struct {
		Status string `json:"status"`
	}
	respBody := returnVals{
	Status: "ok",
	}	

	handler.RespondWithJSON(w, http.StatusOK, respBody)
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

	
	// Return new user as a json
	respBody := ReturnUserVals{
			Id: newUser.ID,
			Email: newUser.Email,
			IsChirpyRed: newUser.IsChirpyRed,
	}
	handler.RespondWithJSON(w, http.StatusCreated, respBody)
}


func (cfg *ApiConfig) LoginHandler(w http.ResponseWriter, r *http.Request) {
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

	// Password is true, create access and refresh jwt tokens
	accessToken, err := cfg.createToken("chirpy-access", strconv.Itoa(usr.ID), 3600)

	if err != nil {
		handler.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}


	// Sign the token with secret key
	refreshToken, err := cfg.createToken("chirpy-refresh", strconv.Itoa(usr.ID), 5184000)

	if err != nil {
		handler.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}


	// Return logged user with JWT token
	type returnVals struct {
		Id int `json:"id"`
		Email string `json:"email"`
		IsChirpyRed bool `json:"is_chirpy_red"`
		Token string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}
	respBody := returnVals{
			Id: usr.ID,
			Email: usr.Email,
			IsChirpyRed: usr.IsChirpyRed,
			Token: accessToken,
			RefreshToken: refreshToken,
	}

	handler.RespondWithJSON(w, http.StatusOK, respBody)
}


func (cfg *ApiConfig) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	token := strings.Split(authHeader, " ")[1]

	tokenObj, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// Provide the key or validation logic for verifying the token
		// For example, if you're using HMAC:
		return []byte(cfg.JwtSecret), nil
	})
	if err != nil {
		handler.RespondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	if !tokenObj.Valid {
		handler.RespondWithError(w, http.StatusUnauthorized, "Invalid token")
		return
	}


	// Check if token is a refresh token 
	if tokenObj.Claims.(jwt.MapClaims)["iss"].(string) == "chirpy-refresh" {
		handler.RespondWithError(w, http.StatusUnauthorized, "token is a refresh token")
		return
	}

	// Token is valid, get the user id on jwtClaims

	userId := tokenObj.Claims.(jwt.MapClaims)["sub"].(string)

	type parameters struct {
		// these tags indicate how the keys in the JSON should be mapped to the struct fields
		// the struct fields must be exported (start with a capital letter) if you want them parsed
		Password string `json:"password"`
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	decoder.Decode(&params)

	// Handle user updating

	intId, err := strconv.Atoi(userId)
	if err != nil {
		handler.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	updatedUser, err := db.UpdateUser(params.Email, params.Password, intId)
	if err != nil {
		handler.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// return updated user

	respBody := ReturnUserVals{
			Id: updatedUser.ID,
			Email: updatedUser.Email,
			IsChirpyRed: updatedUser.IsChirpyRed,
	}

	handler.RespondWithJSON(w, http.StatusOK, respBody)

}


func (cfg *ApiConfig) RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	

	structure, err := db.LoadDB()

	if err != nil {
		handler.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tokenObj, err := cfg.CheckJwtToken(w, r)
	if err != nil {
		handler.RespondWithError(w, http.StatusUnauthorized, "Invalid token")
		return
	}


	if tokenObj.Claims.(jwt.MapClaims)["iss"].(string) != "chirpy-refresh" {
		handler.RespondWithError(w, http.StatusUnauthorized, "token is not a refresh token")
		return
	}

	// Check if token is revoked
	revokedTokens := structure.RevokedTokens
	_, ok := revokedTokens[tokenObj.Raw]
	if ok {
		handler.RespondWithError(w, http.StatusUnauthorized, "Revoked token")
		return
	}

	// Token is valid create new access token
	userId := tokenObj.Claims.(jwt.MapClaims)["sub"].(string)

	accessToken, err := cfg.createToken("chirpy-access", userId, 3600)

	if err != nil {
		handler.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return new token
	type returnVals struct {
		Token string `json:"token"`
	}
	respBody := returnVals{
			Token: accessToken ,
	}
	
	handler.RespondWithJSON(w, http.StatusOK, respBody)
}


func (cfg *ApiConfig) RevokeTokenHandler(w http.ResponseWriter, r *http.Request) {
	
	structure, err := db.LoadDB()

	if err != nil {
		handler.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tokenObj, err := cfg.CheckJwtToken(w, r)
	if err != nil {
		handler.RespondWithError(w, http.StatusUnauthorized, "Invalid token")
		return
	}


	if tokenObj.Claims.(jwt.MapClaims)["iss"].(string) != "chirpy-refresh" {
		handler.RespondWithError(w, http.StatusUnauthorized, "token is not a refresh token")
		return
	}
	
	// Check if token is revoked
	revokedTokens := structure.RevokedTokens
	_, ok := revokedTokens[tokenObj.Raw]
	if ok {
		handler.RespondWithError(w, http.StatusUnauthorized, "Revoked token")
		return
	}

	// Revoke the token
	revokedTokens[tokenObj.Raw] = tokenObj.Raw
	structure.RevokedTokens = revokedTokens
	// Write the updated data to the database file
	db.WriteDB(structure)

	// return the revoked token 
	// Return new token
	type returnVals struct {
		RevokedToken string `json:"revoked_token"`
	}
	respBody := returnVals{
			RevokedToken: tokenObj.Raw,
	}
	handler.RespondWithJSON(w, http.StatusOK, respBody)
}

func (cfg *ApiConfig) createToken(issuer, subject string, expireDate int) (string, error){

	newAccessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer: issuer,
		IssuedAt: jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(time.Second * time.Duration(expireDate))},
		Subject: subject,
	})

	// Sign the token with secret key
	accessToken, err := newAccessToken.SignedString([]byte(cfg.JwtSecret))

	if err != nil {
		return "", err
	}

	return accessToken, nil
}


func (cfg *ApiConfig) CheckJwtToken(w http.ResponseWriter, r *http.Request) (*jwt.Token, error){
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
    // Handle the case when Authorization header is missing or empty
		return nil, errors.New("jwt token missing")
	}
	token := strings.Split(authHeader, " ")[1]

	tokenObj, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// Provide the key or validation logic for verifying the token
		// For example, if you're using HMAC:
		return []byte(cfg.JwtSecret), nil
	})
	if err != nil {
		return nil, err
	}

	if !tokenObj.Valid {
		handler.RespondWithError(w, http.StatusUnauthorized, "Invalid token")
		return nil, err
	}

	return tokenObj, nil
}


func(cfg *ApiConfig) PolkaWebhooksHandler(w http.ResponseWriter, r *http.Request) {

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
    // Handle the case when Authorization header is missing or empty
		handler.RespondWithError(w, http.StatusUnauthorized, "missing api key")
		return
	}
	apiKey := strings.Split(authHeader, " ")[1]

	if apiKey != os.Getenv("POLKA_KEY") {
		handler.RespondWithError(w, http.StatusUnauthorized, "invalid api key")
		return
	}

	// Take body params
	type parameters struct {
		Event string `json:"event"`
		Data map[string]int `json:"data"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		// handle decode parameters error 
		handler.RespondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	respBody := struct {
		Status int
	} {
		Status: http.StatusOK,
	}

	if params.Event != "user.upgraded" {
		handler.RespondWithJSON(w, http.StatusOK, respBody)
		return
	}

	// get users
	structure, err := db.LoadDB() 

	if err != nil {
		handler.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	users := structure.Users

	// Check if user exists

	user, ok := users[params.Data["user_id"]]

	if !ok {
		handler.RespondWithError(w, http.StatusNotFound, "not found")
		return
	}

	// User is found, make user chirpy red member
	user.IsChirpyRed = true
	users[params.Data["user_id"]] = user
	structure.Users = users

	db.WriteDB(structure)

	// Send ok status with empty json body
	handler.RespondWithJSON(w, http.StatusOK, make(map[string]interface{}))
}