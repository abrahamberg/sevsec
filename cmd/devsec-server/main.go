package main

import (
	"log"

	"github.com/abrahamberg/devsec/internal/server/config"
	serverhttp "github.com/abrahamberg/devsec/internal/server/http"
)

func main() {

	cfg := config.Load()

	server := serverhttp.NewServer(cfg)

	if err := server.Run(); err != nil {
		log.Fatal(err)
	}

}
