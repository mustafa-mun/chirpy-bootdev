package sys

import (
	"flag"
	"fmt"
	"log"
	"os"
	"github.com/joho/godotenv"
)



func LoadDotenv() {
	err := godotenv.Load()
  if err != nil {
    log.Fatal("Error loading .env file")
  }
}

func EnableDebugMode() {
	dbg := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()

	if *dbg {
		err := os.Remove("database.json")
		if err != nil {
			fmt.Println(os.ErrNotExist)
		}
	}
}