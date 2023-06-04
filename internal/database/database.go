package database

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"sync"

	"github.com/mustafa-mun/chirpy-bootdev/internal/bcrypt"
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
	Password string `json:"password"`
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
func (db *DB) CreateUser(password, email string) (User, error) {
	db.mux.Lock()
	defer db.mux.Unlock()
	userIdCount += 1

	createdUser, err := db.handleUserCreation(password, email, userIdCount)

	if err != nil {
		return User{}, err
	}
	return createdUser, nil
}

func (db *DB) UpdateUser(email, password string, userId int) (User, error) {
	db.mux.Lock()
	defer db.mux.Unlock()

	updatedUser, err := db.handleUserCreation(password, email, userId)

	if err != nil {
		return User{}, err
	}
	return updatedUser, nil
}

func (db *DB) handleUserCreation(password, email string, id int) (User, error) {
	// check if user is already exists
	err := db.checkDuplicateUser(email)
	if err != nil {
		return User{}, err
	}

	// Read database file
	structure, err := db.LoadDB()

	if err != nil {
		return User{}, err
	}

	// Access the Users map
	users := structure.Users

	hashedPassword, err := bcrypt.CreateHashedPassword(password)

	if err != nil {
		return User{}, err
	}

	user := User{ID: id, Password: hashedPassword, Email: email}
	users[id] = user

	// Update the idCount in the DBStructure
	structure.Users = users
	
	// Write the updated data to the database file
	db.WriteDB(structure)

	return user, nil
}


func (db *DB) checkDuplicateUser(email string) error {
	// Read database file
	structure, err := db.LoadDB()
	if err != nil {
		return err
	}

	users := structure.Users

	for _, user := range users {
		if user.Email == email {
			return errors.New("user already exists")
		}
	}

	return nil
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