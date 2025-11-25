package handlers

import (
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

		detail := r.URL.Query().Get("msg")
		if detail == "" {
			detail = "hello from http handler"
		}
		payload := WorkPayload{
			Source: "http",
			Detail: detail,
		}
		if err := sendWork(r.Context(), ctx.Queues, payload); err != nil {
			http.Error(w, "failed to enqueue work", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(fmt.Sprintf("queued work: %s", detail)))
	})
}
