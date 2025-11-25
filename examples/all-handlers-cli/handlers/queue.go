package handlers

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/transire/transire"
)

func RegisterQueues(app *transire.App) {
	app.RegisterQueueHandler(WorkQueue, func(ctx transire.Context, msg transire.Message) error {
		var work WorkPayload
		if err := json.Unmarshal(msg.Body, &work); err != nil {
			return fmt.Errorf("decode work message: %w", err)
		}

		return sendNotification(ctx, ctx.Queues, NotificationPayload{
			Stage:  "work-processed",
			Detail: fmt.Sprintf("%s via %s", work.Detail, work.Source),
		})
	})

	app.RegisterQueueHandler(NotificationsQueue, func(ctx transire.Context, msg transire.Message) error {
		var note NotificationPayload
		if err := json.Unmarshal(msg.Body, &note); err != nil {
			log.Printf("notification decode failed: %v", err)
			return err
		}
		if err := ctx.Queues.Send(ctx, NotificationLog, msg.Body); err != nil {
			return fmt.Errorf("forward notification: %w", err)
		}
		log.Printf("notification[%s]: %s", note.Stage, note.Detail)
		return nil
	})

	app.RegisterQueueHandler(NotificationLog, func(ctx transire.Context, msg transire.Message) error {
		log.Printf("notification-log received: %s", string(msg.Body))
		return nil
	})
}
