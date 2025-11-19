//go:build lambda

// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lambda

import (
	"time"

	"github.com/aws/aws-lambda-go/events"
)

// Message implements the transire.Message interface for SQS messages
type Message struct {
	id         string
	body       []byte
	attributes map[string]events.SQSMessageAttribute
}

func (m *Message) ID() string {
	return m.id
}

func (m *Message) Body() []byte {
	return m.body
}

func (m *Message) Attributes() map[string]string {
	attrs := make(map[string]string)
	for key, attr := range m.attributes {
		if attr.StringValue != nil {
			attrs[key] = *attr.StringValue
		}
	}
	return attrs
}

func (m *Message) DeliveryCount() int {
	// SQS doesn't directly expose delivery count, would need to track separately
	return 1
}

func (m *Message) EnqueuedAt() time.Time {
	// Would need to extract from message attributes or approximate
	return time.Now()
}
