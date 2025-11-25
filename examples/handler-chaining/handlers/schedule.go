package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/transire/transire"
)

func RegisterSchedules(app *transire.App) {
	app.RegisterScheduleHandler("heartbeat", time.Minute, func(ctx transire.Context, at time.Time) error {
		log.Printf("heartbeat at %s", at.UTC())
		payload := WorkPayload{
			Source: "schedule",
			Detail: fmt.Sprintf("heartbeat at %s", at.UTC().Format(time.RFC3339)),
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal heartbeat: %w", err)
		}
		return ctx.Queues.Send(ctx, WorkQueue, body)
	})
}
