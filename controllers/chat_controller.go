package controllers

import (
	"encoding/json"
	"net/http"
	"strings"

	"chatbot/models"
)

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
