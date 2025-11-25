package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/transire/transire"
)

func RegisterQueues(app *transire.App) {
	app.RegisterQueueHandler(WorkQueue, func(ctx transire.Context, msg transire.Message) error {
		var payload WorkPayload
		if err := json.Unmarshal(msg.Body, &payload); err != nil {
			payload = WorkPayload{
				Source: "queue",
				Detail: string(msg.Body),
			}
		}

		log.Printf("work queue received from %s: %s", payload.Source, payload.Detail)

		summary := SummaryPayload{
			Source: payload.Source,
			Steps:  []string{fmt.Sprintf("work accepted: %s", payload.Detail)},
		}
		body, err := json.Marshal(summary)
		if err != nil {
			return fmt.Errorf("marshal summary: %w", err)
		}

		return ctx.Queues.Send(ctx, SummaryQueue, body)
	})

	app.RegisterQueueHandler(SummaryQueue, func(ctx transire.Context, msg transire.Message) error {
		var payload SummaryPayload
		if err := json.Unmarshal(msg.Body, &payload); err != nil {
			payload = SummaryPayload{
				Source: "summary",
				Steps:  []string{string(msg.Body)},
			}
		}

		payload.Steps = append(payload.Steps, "forwarded to log")
		log.Printf("summary received for %s: %s", payload.Source, strings.Join(payload.Steps, " -> "))

		logPayload := LogPayload{
			Message: strings.Join(payload.Steps, " -> "),
		}
		body, err := json.Marshal(logPayload)
		if err != nil {
			return fmt.Errorf("marshal log payload: %w", err)
		}

		return ctx.Queues.Send(ctx, LogQueue, body)
	})

	app.RegisterQueueHandler(LogQueue, func(ctx transire.Context, msg transire.Message) error {
		var payload LogPayload
		if err := json.Unmarshal(msg.Body, &payload); err != nil {
			payload = LogPayload{Message: string(msg.Body)}
		}
		log.Printf("log queue received: %s", payload.Message)
		return nil
	})
}
