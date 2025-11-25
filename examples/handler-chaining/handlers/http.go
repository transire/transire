package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/transire/transire"
)

func RegisterHTTP(app *transire.App) {
	app.Router().Get("/", func(w http.ResponseWriter, r *http.Request) {
		ctx, ok := transire.RequestContext(r)
		if !ok {
			http.Error(w, "missing transire context", http.StatusInternalServerError)
			return
		}

		msg := r.URL.Query().Get("msg")
		if msg == "" {
			msg = "hello from http"
		}

		body, err := json.Marshal(WorkPayload{
			Source: "http",
			Detail: msg,
		})
		if err != nil {
			http.Error(w, "failed to encode payload", http.StatusInternalServerError)
			return
		}
		if err := ctx.Queues.Send(r.Context(), WorkQueue, body); err != nil {
			http.Error(w, "failed to enqueue", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(fmt.Sprintf("queued work for: %s", msg)))
	})
}
