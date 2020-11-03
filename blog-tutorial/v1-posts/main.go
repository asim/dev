package main

import (
	"posts/handler"

	"github.com/micro/micro/v3/service"
	"github.com/micro/micro/v3/service/logger"
)

func main() {
	// Create the service
	srv := service.New(
		service.Name("posts"),
	)

	// Register Handler
	srv.Handle(handler.NewPosts())

	// Run service
	if err := srv.Run(); err != nil {
		logger.Fatal(err)
	}
}
