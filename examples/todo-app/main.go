// Copyright (c) 2024 Transire Contributors
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/transire/transire/pkg/transire"
)

func main() {
	// Create Transire app
	app := transire.New()

	// Initialize in-memory storage
	store := NewTodoStore()

	// Get Chi router
	r := app.Router()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// Routes
	r.Get("/", homeHandler)
	r.Get("/health", healthHandler)

	r.Route("/api/v1/todos", func(r chi.Router) {
		r.Post("/", createTodoHandler(store))
		r.Get("/", listTodosHandler(store))
		r.Get("/{id}", getTodoHandler(store))
		r.Put("/{id}", updateTodoHandler(store))
		r.Delete("/{id}", deleteTodoHandler(store))
		r.Post("/{id}/complete", completeTodoHandler(store))
	})

	// Register handlers
	app.RegisterQueueHandler(&TodoReminderQueue{store: store})
	app.RegisterQueueHandler(&TodoNotificationQueue{store: store})
	app.RegisterScheduleHandler(&CleanupCompletedTodosSchedule{store: store})
	app.RegisterScheduleHandler(&DailyTodoSummarySchedule{store: store})

	// Run the app
	if err := app.Run(context.Background()); err != nil {
		panic(err)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"message": "Welcome to Transire TODO API",
		"version": "1.0.0",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "transire-todo-api",
	})
}

// Todo represents a todo item
type Todo struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	DueDate     time.Time `json:"due_date,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TodoStore provides in-memory storage for todos
type TodoStore struct {
	mu    sync.RWMutex
	todos map[string]*Todo
}

func NewTodoStore() *TodoStore {
	return &TodoStore{
		todos: make(map[string]*Todo),
	}
}

func (s *TodoStore) Create(todo *Todo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.todos[todo.ID] = todo
}

func (s *TodoStore) Get(id string) (*Todo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	todo, ok := s.todos[id]
	return todo, ok
}

func (s *TodoStore) List() []*Todo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	todos := make([]*Todo, 0, len(s.todos))
	for _, todo := range s.todos {
		todos = append(todos, todo)
	}
	return todos
}

func (s *TodoStore) Update(todo *Todo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.todos[todo.ID] = todo
}

func (s *TodoStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.todos, id)
}

func createTodoHandler(store *TodoStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Title       string    `json:"title"`
			Description string    `json:"description"`
			DueDate     time.Time `json:"due_date,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if input.Title == "" {
			http.Error(w, "Title is required", http.StatusBadRequest)
			return
		}

		todo := &Todo{
			ID:          uuid.New().String(),
			Title:       input.Title,
			Description: input.Description,
			Completed:   false,
			DueDate:     input.DueDate,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		store.Create(todo)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(todo)
	}
}

func listTodosHandler(store *TodoStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		todos := store.List()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"todos": todos,
			"count": len(todos),
		})
	}
}

func getTodoHandler(store *TodoStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		todo, ok := store.Get(id)
		if !ok {
			http.Error(w, "Todo not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(todo)
	}
}

func updateTodoHandler(store *TodoStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		todo, ok := store.Get(id)
		if !ok {
			http.Error(w, "Todo not found", http.StatusNotFound)
			return
		}

		var updates struct {
			Title       *string    `json:"title,omitempty"`
			Description *string    `json:"description,omitempty"`
			Completed   *bool      `json:"completed,omitempty"`
			DueDate     *time.Time `json:"due_date,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if updates.Title != nil {
			todo.Title = *updates.Title
		}
		if updates.Description != nil {
			todo.Description = *updates.Description
		}
		if updates.Completed != nil {
			todo.Completed = *updates.Completed
		}
		if updates.DueDate != nil {
			todo.DueDate = *updates.DueDate
		}
		todo.UpdatedAt = time.Now()

		store.Update(todo)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(todo)
	}
}

func deleteTodoHandler(store *TodoStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if _, ok := store.Get(id); !ok {
			http.Error(w, "Todo not found", http.StatusNotFound)
			return
		}

		store.Delete(id)
		w.WriteHeader(http.StatusNoContent)
	}
}

func completeTodoHandler(store *TodoStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		todo, ok := store.Get(id)
		if !ok {
			http.Error(w, "Todo not found", http.StatusNotFound)
			return
		}

		todo.Completed = true
		todo.UpdatedAt = time.Now()
		store.Update(todo)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(todo)
	}
}
