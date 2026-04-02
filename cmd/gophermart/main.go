package main

import (
	"log"

	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatalf("Closed with error %v", err)
	}
}
