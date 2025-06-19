package controllers

import (
	"encoding/json"
	"net/http"
)

// IndexHandler serves our main HTML page (extracted from main.go)
func (c *Controller) IndexHandler(w http.ResponseWriter, r *http.Request) {
	// Render index template (no data needed for static content)
	c.renderTemplate(w, "views/index.html", nil)
}

// HealthHandler provides a health check endpoint (extracted from main.go)
func (c *Controller) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	chatbotStatus := c.chatbot.GetStatus()
	discordStatus := c.discordService.GetStatus()

	health := map[string]interface{}{
		"status":    "healthy",
		"phase":     "3+",
		"component": "mvc-with-chatbot-and-discord",
		"endpoints": []string{"/", "/chat", "/health"},
		"chatbot":   chatbotStatus,
		"discord":   discordStatus,
	}

	json.NewEncoder(w).Encode(health)
}
