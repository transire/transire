package main

import (
	"context"
	"log"

	"example.com/transire-all-handlers-cli/handlers"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/transire/transire"
	"github.com/transire/transire/dispatcher"
)

func main() {
	app := transire.New()
	app.Router().Use(middleware.Logger)

	handlers.RegisterHTTP(app)
	handlers.RegisterQueues(app)
	handlers.RegisterSchedules(app)

	d, err := dispatcher.Auto()
	if err != nil {
		log.Fatal(err)
	}
	app.SetDispatcher(d)

	if err := app.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
