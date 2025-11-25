package handlers

import (
	"fmt"
	"log"
	"time"

	"github.com/transire/transire"
)

func RegisterSchedules(app *transire.App) {
	app.RegisterScheduleHandler("heartbeat", time.Minute, func(ctx transire.Context, at time.Time) error {
		log.Printf("heartbeat at %s", at.UTC())
		return sendWork(ctx, ctx.Queues, WorkPayload{
			Source: "schedule",
			Detail: fmt.Sprintf("heartbeat at %s", at.UTC().Format(time.RFC3339)),
		})
	})
}
