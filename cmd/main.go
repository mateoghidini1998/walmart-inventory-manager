package main

import (
	"fmt"
	"log"
	"walmart-inventory-manager/internal/application"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load("C:/Users/User/Desktop/walmart-inventory-manager/.env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	a, err := application.NewApplication()
	if err != nil {
		log.Fatalf("Error initializing application: %v", err)
	}

	err = a.SetUp()
	if err != nil {
		log.Fatalf("Error setting up application: %v", err)
	}

	err = a.Run()
	if err != nil {
		log.Fatalf("Error running application: %v", err)
	}

	err = a.TearDown()
	if err != nil {
		fmt.Printf("Error tearing down application: %v\n", err)
	}
}
