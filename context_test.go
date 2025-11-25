// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package transire

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type noopSender struct{}

func (noopSender) Send(ctx context.Context, queue string, payload []byte) error { return nil }

func TestRequestContextInjectsAndRetrieves(t *testing.T) {
	sender := noopSender{}
	mw := InjectContext(sender)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, ok := RequestContext(r)
		if !ok {
			t.Fatalf("context not found")
		}
		if ctx.Queues == nil {
			t.Fatalf("queue sender missing")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected OK, got %d", rr.Result().StatusCode)
	}
}

func TestRequestContextAbsent(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if _, ok := RequestContext(req); ok {
		t.Fatalf("expected no context present")
	}
}
