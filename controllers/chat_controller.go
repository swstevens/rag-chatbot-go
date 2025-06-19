package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"chatbot/models"
)

// HelloHandler processes POST requests and returns a modified greeting (extracted from main.go)
func (c *Controller) HelloHandler(w http.ResponseWriter, r *http.Request) {
	// Handle both JSON and form data
	var req models.HelloRequest

	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		// Handle JSON requests (for API usage)
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.HelloResponse{
				Response: "Invalid JSON format",
				Status:   "error",
			})
			return
		}

		// Return JSON response
		w.Header().Set("Content-Type", "application/json")
		response := c.processHelloRequest(req)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)

	} else {
		// Handle form submissions (from HTML form)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		req.Name = r.FormValue("name")
		req.Message = r.FormValue("message")

		// Validate input
		if strings.TrimSpace(req.Name) == "" {
			http.Error(w, "Name field is required", http.StatusBadRequest)
			return
		}

		// Process the request
		response := c.processHelloRequest(req)

		// Render HTML response using template
		c.renderTemplate(w, "views/response.html", response)
	}
}

// ChatHandler processes chat requests using the chatbot service
func (c *Controller) ChatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.ChatRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ChatResponse{
			Message: "Invalid JSON format",
			Status:  "error",
		})
		return
	}

	// Validate message
	if strings.TrimSpace(req.Message) == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ChatResponse{
			Message: "Message cannot be empty",
			Status:  "error",
		})
		return
	}

	// Generate session ID if not provided
	if req.SessionID == "" {
		req.SessionID = c.generateSessionID()
	}

	// Process message through chatbot service
	response := c.chatbot.ProcessMessage(req.Message, req.SessionID, req.History)

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// processHelloRequest processes the hello request logic (extracted from main.go)
func (c *Controller) processHelloRequest(req models.HelloRequest) models.HelloResponse {
	name := strings.TrimSpace(req.Name)

	// Basic input transformation - demonstrating request processing
	var responseText string

	if req.Message != "" {
		// If message is provided, create a more complex response
		message := strings.TrimSpace(req.Message)
		responseText = fmt.Sprintf("Hello %s! You said: '%s'. That's interesting!",
			name, message)
	} else {
		// Simple greeting
		responseText = fmt.Sprintf("Hello %s! Welcome to the RAG chatbot development server.", name)
	}

	return models.HelloResponse{
		Response: responseText,
		Status:   "success",
	}
}
