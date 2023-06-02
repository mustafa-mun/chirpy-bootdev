package database

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"sync"
)

type DB struct {
	path string
	mux  *sync.RWMutex
}

type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
	Users map[int]User `json:"users"`
}

type Chirp struct {
	ID   int    `json:"id"`
	Body string `json:"body"`
}

type User struct {
	ID   int    `json:"id"`
	Email string `json:"email"`
}

// NewDB creates a new database connection
// and creates the database file if it doesn't exist
func NewDB(path string) (*DB, error) {
	newDb := DB{path: path, mux: &sync.RWMutex{}}
	// If database file already exists 
	if _, err := os.Stat(path); err == nil {
		return &newDb, nil
	}

	// Create database file
	f, err := os.Create(newDb.path)
	if err != nil {
		return nil, errors.New("an error occurred when creating the database file")
	}
	defer f.Close()

	chripMp := make(map[int]Chirp)
	usrMp := make(map[int]User)
	structure := DBStructure{Chirps: chripMp, Users: usrMp}

	// write structure 
	newDb.WriteDB(structure)

	return &newDb, nil
}

// This will have a more optimal solution
var chirpIdCount int = 0

// CreateChirp creates a new chirp and saves it to disk
func (db *DB) CreateChirp(body string) (Chirp, error) {
	db.mux.Lock()
	defer db.mux.Unlock()

	// Read database file
	structure, err := db.LoadDB()

	if err != nil {
		return Chirp{}, err
	}

	// Access the Chirps map
	chirps := structure.Chirps

	// Initialize chirps map if it is nil
	if chirps == nil {
		chirps = make(map[int]Chirp)
	}

	chirpIdCount += 1
	newChirp := Chirp{ID: chirpIdCount, Body: body}
	chirps[chirpIdCount] = newChirp

	// Update the chirpIdCount in the DBStructure
	structure.Chirps = chirps
	
	// Write the updated data to the database file
	db.WriteDB(structure)

	return newChirp, nil
}

var userIdCount = 0

// CreateChirp creates a new chirp and saves it to disk
func (db *DB) CreateUser(email string) (User, error) {
	db.mux.Lock()
	defer db.mux.Unlock()

	// Read database file
	structure, err := db.LoadDB()

	if err != nil {
		return User{}, err
	}

	// Access the Users map
	users := structure.Users

	// Initialize users map if it is nil
	if users == nil {
		users = make(map[int]User)
	}

	userIdCount += 1
	newUser := User{ID: userIdCount, Email: email}
	users[userIdCount] = newUser

	// Update the idCount in the DBStructure
	structure.Users = users
	
	// Write the updated data to the database file
	db.WriteDB(structure)

	return newUser, nil
}


// GetChirps returns all chirps in the database
func (db *DB) GetChirps() ([]Chirp, error) {
	// Read database file
	structure, err := db.LoadDB()
	if err != nil {
		return nil, err
	}

	// Access the Chirps map
	chirps := structure.Chirps

	chirpsArray := make([]Chirp, 0, len(chirps))

	for _, value := range chirps {
		chirpsArray = append(chirpsArray, value)
	}

	sort.Slice(chirpsArray, func(i, j int) bool {
    return chirpsArray[i].ID < chirpsArray[j].ID
	})

	return chirpsArray, nil
}

// loadDB reads the database file into memory
func (db *DB) LoadDB() (DBStructure, error) {
	// Read database file
	data, err := os.ReadFile("database.json")
	if err != nil {
		return DBStructure{}, err
	}

	// Decode JSON data into DBStructure object
	var structure DBStructure
	err = json.Unmarshal(data, &structure)
	if err != nil {
		return DBStructure{}, err
	}

	return structure, nil
}

// writeDB writes the database file to disk
func (db *DB) WriteDB(dbStructure DBStructure) error  {
	data, err := json.Marshal(dbStructure)
	if err != nil {
		return errors.New("an error occurred when encoding database structure to JSON")
	}

	err = os.WriteFile("database.json", data, 0644)
	if err != nil {
		return errors.New("an error occurred when writing data to the database file")
	}

	return nil
}