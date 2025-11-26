package handlers

import (
	"fmt"
	"log"

	"github.com/transire/transire"
)

func RegisterQueues(app *transire.App) {
	app.RegisterQueueHandler(WorkQueue, func(ctx transire.Context, msg transire.Message) error {
		log.Printf("work queue received: %s", string(msg.Body))
		return ctx.Queues.Send(ctx, AuditQueue, []byte(fmt.Sprintf("audit: %s", msg.Body)))
	})

	app.RegisterQueueHandler(AuditQueue, func(ctx transire.Context, msg transire.Message) error {
		log.Printf("audit log: %s", string(msg.Body))
		return nil
	})
}
