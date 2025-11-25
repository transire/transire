// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package transire

import (
	"context"
	"net/http"
)

type contextKey string

const httpContextKey contextKey = "transire-http-context"

// InjectContext adds a Transire context into each HTTP request.
func InjectContext(sender QueueSender) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := Context{
				Context: r.Context(),
				Queues:  sender,
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), httpContextKey, ctx)))
		})
	}
}

// RequestContext extracts the Transire context from an HTTP request when present.
func RequestContext(r *http.Request) (Context, bool) {
	value := r.Context().Value(httpContextKey)
	if value == nil {
		return Context{}, false
	}
	if ctx, ok := value.(Context); ok {
		return ctx, true
	}
	return Context{}, false
}
