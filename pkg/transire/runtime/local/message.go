//go:build local

// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package local

import (
	"time"
)

// Message implements the transire.Message interface for local development
type Message struct {
	id            string
	body          []byte
	attributes    map[string]string
	deliveryCount int
	enqueuedAt    time.Time
}

func (m *Message) ID() string {
	return m.id
}

func (m *Message) Body() []byte {
	return m.body
}

func (m *Message) Attributes() map[string]string {
	return m.attributes
}

func (m *Message) DeliveryCount() int {
	return m.deliveryCount
}

func (m *Message) EnqueuedAt() time.Time {
	return m.enqueuedAt
}
