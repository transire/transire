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
	"fmt"
	"net/http"

	"github.com/transire/transire"
)

func RegisterHTTP(app *transire.App) {
	app.Router().Get("/", func(w http.ResponseWriter, r *http.Request) {
		if ctx, ok := transire.RequestContext(r); ok {
			msg := r.URL.Query().Get("msg")
			if msg == "" {
				msg = "hello from http"
			}
			if err := ctx.Queues.Send(r.Context(), WorkQueue, []byte(msg)); err != nil {
				http.Error(w, "failed to enqueue", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(fmt.Sprintf("queued: %s", msg)))
			return
		}
		http.Error(w, "missing transire context", http.StatusInternalServerError)
	})
}
