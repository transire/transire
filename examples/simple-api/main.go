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

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/transire/transire/pkg/transire"
)

func main() {
	// Create Transire app
	app := transire.New()

	// Get Chi router - use exactly like normal Chi
	r := app.Router()

	// Standard Chi middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// Standard Chi routes
	r.Get("/", homeHandler)
	r.Get("/health", healthHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/users", createUserHandler)
		r.Get("/users/{id}", getUserHandler)
		r.Put("/users/{id}", updateUserHandler)
		r.Delete("/users/{id}", deleteUserHandler)
	})

	// Register queue and schedule handlers
	app.RegisterQueueHandler(&EmailQueueHandler{})
	app.RegisterQueueHandler(&NotificationQueueHandler{})
	app.RegisterScheduleHandler(&CleanupHandler{})

	// Run the app (works locally and in Lambda)
	if err := app.Run(context.Background()); err != nil {
		panic(err)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"message": "Welcome to Transire Simple API - HOT RELOAD WORKS!",
		"version": "2.0.0",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "transire-simple-api",
	})
}

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// TODO: Validate and save user to database
	user.ID = "generated-id"

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	// TODO: Fetch user from database
	user := User{
		ID:    userID,
		Name:  "John Doe",
		Email: "john@example.com",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func updateUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	var updates User
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// TODO: Update user in database
	updates.ID = userID

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updates)
}

func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	// TODO: Delete user from database
	_ = userID

	w.WriteHeader(http.StatusNoContent)
}
