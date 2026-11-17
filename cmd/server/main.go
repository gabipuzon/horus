package main

import (
	"log"

	"github.com/gabipuzon/horus/internal/server"
)

func main() {
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
