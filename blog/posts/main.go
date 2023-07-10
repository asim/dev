package main

import (
	"posts/handler"

	"micro.dev/v4/service"
	"micro.dev/v4/service/logger"
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
