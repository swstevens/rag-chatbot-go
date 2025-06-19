package controllers

import (
	"chatbot/models"
	"encoding/json"
	"net/http"
	"strings"
)

// RAGHandler processes RAG queries
func (c *Controller) RAGHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.RAGRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid JSON format",
		})
		return
	}

	// Validate query
	if strings.TrimSpace(req.Query) == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Query cannot be empty",
		})
		return
	}

	// Set default limit
	if req.Limit <= 0 {
		req.Limit = 5
	}

	// Process query through chatbot service (which will use RAG if enabled)
	ragResponse := c.chatbot.ProcessRAGQuery(req.Query, req.ChannelID, req.Limit)

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ragResponse)
}
