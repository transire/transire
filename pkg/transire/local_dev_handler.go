package transire

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// devHandler wraps the user's router with dev endpoints
type devHandler struct {
	app        *App
	userRouter chi.Router
}

// newDevHandler creates a dev handler that wraps the user router
func newDevHandler(app *App) http.Handler {
	handler := &devHandler{
		app:        app,
		userRouter: app.Router(),
	}

	// Create a new router for dev endpoints
	r := chi.NewRouter()

	// Mount user routes at root
	r.Mount("/", handler.userRouter)

	// Mount dev API endpoints under /__dev
	r.Route("/__dev", func(r chi.Router) {
		r.Post("/queues/send", handler.handleQueueSend)
		r.Get("/queues/list", handler.handleQueuesList)
		r.Post("/schedules/execute", handler.handleScheduleExecute)
		r.Get("/schedules/list", handler.handleSchedulesList)
		r.Get("/health", handler.handleDevHealth)
	})

	return r
}

// handleQueueSend handles sending a test message to a queue
func (h *devHandler) handleQueueSend(w http.ResponseWriter, r *http.Request) {
	var req struct {
		QueueName string `json:"queue_name"`
		Message   string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Find the queue handler
	handler := h.app.FindQueueHandler(req.QueueName)
	if handler == nil {
		http.Error(w, fmt.Sprintf("Queue handler '%s' not found", req.QueueName), http.StatusNotFound)
		return
	}

	// Create a test message
	message := &localMessage{
		id:            fmt.Sprintf("dev-msg-%d", time.Now().UnixNano()),
		body:          []byte(req.Message),
		attributes:    make(map[string]string),
		deliveryCount: 1,
		enqueuedAt:    time.Now(),
	}

	log.Printf("[DEV] Sending message to queue '%s': %s", req.QueueName, message.ID())

	// Process the message in a goroutine (async like real queue processing)
	go func() {
		ctx := context.Background()
		failedIDs, err := handler.HandleMessages(ctx, []Message{message})
		if err != nil {
			log.Printf("[DEV] Error processing message %s: %v", message.ID(), err)
		} else if len(failedIDs) > 0 {
			log.Printf("[DEV] Message %s failed to process", message.ID())
		} else {
			log.Printf("[DEV] Message %s processed successfully", message.ID())
		}
	}()

	// Return success immediately
	response := map[string]interface{}{
		"success":    true,
		"queue_name": req.QueueName,
		"message_id": message.ID(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleScheduleExecute handles manual execution of a scheduled task
func (h *devHandler) handleScheduleExecute(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ScheduleName string `json:"schedule_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Find the schedule handler
	handler := h.app.FindScheduleHandler(req.ScheduleName)
	if handler == nil {
		http.Error(w, fmt.Sprintf("Schedule handler '%s' not found", req.ScheduleName), http.StatusNotFound)
		return
	}

	// Create a test schedule event
	event := ScheduleEvent{
		Name:          req.ScheduleName,
		ScheduledTime: time.Now(),
		EventID:       fmt.Sprintf("dev-schedule-%d", time.Now().UnixNano()),
	}

	log.Printf("[DEV] Executing schedule '%s': %s", req.ScheduleName, event.EventID)

	// Execute the handler in a goroutine (async like real schedule execution)
	go func() {
		ctx := context.Background()
		if err := handler.HandleSchedule(ctx, event); err != nil {
			log.Printf("[DEV] Error executing schedule %s: %v", req.ScheduleName, err)
		} else {
			log.Printf("[DEV] Schedule %s executed successfully", req.ScheduleName)
		}
	}()

	// Return success immediately
	response := map[string]interface{}{
		"success":       true,
		"schedule_name": req.ScheduleName,
		"event_id":      event.EventID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleQueuesList returns all registered queue handlers with their configurations
func (h *devHandler) handleQueuesList(w http.ResponseWriter, r *http.Request) {
	queueHandlers := h.app.GetQueueHandlers()

	queues := make([]map[string]interface{}, 0, len(queueHandlers))
	for _, handler := range queueHandlers {
		config := handler.Config()
		queues = append(queues, map[string]interface{}{
			"name":                       handler.QueueName(),
			"batch_size":                 config.BatchSize,
			"visibility_timeout_seconds": config.VisibilityTimeoutSeconds,
			"max_receive_count":          config.MaxReceiveCount,
			"wait_time_seconds":          config.WaitTimeSeconds,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"queues": queues,
	})
}

// handleSchedulesList returns all registered schedule handlers with their configurations
func (h *devHandler) handleSchedulesList(w http.ResponseWriter, r *http.Request) {
	schedHandlers := h.app.GetScheduleHandlers()

	schedules := make([]map[string]interface{}, 0, len(schedHandlers))
	for _, handler := range schedHandlers {
		config := handler.Config()
		schedules = append(schedules, map[string]interface{}{
			"name":            handler.Name(),
			"schedule":        handler.Schedule(),
			"timezone":        config.Timezone,
			"enabled":         config.Enabled,
			"timeout_seconds": config.TimeoutSeconds,
			"retry_attempts":  config.RetryAttempts,
			"retry_delay":     config.RetryDelay.Seconds(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"schedules": schedules,
	})
}

// handleDevHealth returns dev API health status
func (h *devHandler) handleDevHealth(w http.ResponseWriter, r *http.Request) {
	queueHandlers := h.app.GetQueueHandlers()
	schedHandlers := h.app.GetScheduleHandlers()

	response := map[string]interface{}{
		"status":            "healthy",
		"queue_handlers":    len(queueHandlers),
		"schedule_handlers": len(schedHandlers),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
