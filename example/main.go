package main

import (
	"fmt"
	"os"

	"github.com/sankofalab/novasend-go-sdk"
)

func main() {
	client, err := novasend.NewClient(novasend.Config{
		APIKey:    os.Getenv("NOVASEND_API_KEY"),
		SecretKey: os.Getenv("NOVASEND_SECRET_KEY"),
		BaseURL:   os.Getenv("NOVASEND_BASE_URL"),
	})
	if err != nil {
		fmt.Printf("Erreur d'initialisation: %v\n", err)
		return
	}

	fmt.Printf("Client Novasend prêt: %T\n", client)
}
