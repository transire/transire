// Copyright (c) 2025 Transire contributors
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

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
