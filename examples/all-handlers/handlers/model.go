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
	"context"
	"encoding/json"
	"fmt"

	"github.com/transire/transire"
)

const (
	WorkQueue          = "work-events"
	NotificationsQueue = "notifications"
	NotificationLog    = "notification-log"
)

type WorkPayload struct {
	Source string `json:"source"`
	Detail string `json:"detail"`
}

type NotificationPayload struct {
	Stage  string `json:"stage"`
	Detail string `json:"detail"`
}

func sendWork(ctx context.Context, sender transire.QueueSender, payload WorkPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal work payload: %w", err)
	}
	return sender.Send(ctx, WorkQueue, body)
}

func sendNotification(ctx context.Context, sender transire.QueueSender, payload NotificationPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal notification payload: %w", err)
	}
	return sender.Send(ctx, NotificationsQueue, body)
}
