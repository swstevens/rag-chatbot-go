package controllers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"chatbot/models"
	"chatbot/services"
)

// Controller handles all the business logic (extracted from main.go Server methods)
type Controller struct {
	chatbot        *services.Chatbot
	discordService *services.DiscordService
}

// NewController creates a new controller instance
func NewController(llmProvider services.LLMProvider, enableSearch bool) *Controller {
	// Initialize chatbot service with specified provider and search capability
	chatbot := services.NewChatbot(llmProvider, enableSearch)

	// Initialize Discord service
	discordService := services.NewDiscordService(chatbot)

	return &Controller{
		chatbot:        chatbot,
		discordService: discordService,
	}
}

// StartServices starts all background services (Discord bot, etc.)
func (c *Controller) StartServices(enableDiscord bool) error {
	// Start Discord service only if enabled via flag AND properly configured
	if enableDiscord && c.discordService.IsEnabled() {
		if err := c.discordService.Start(); err != nil {
			log.Printf("Failed to start Discord service: %v", err)
			return err
		}
	} else if enableDiscord && !c.discordService.IsEnabled() {
		log.Printf("Discord service requested but not properly configured (missing DISCORD_BOT_TOKEN)")
	} else {
		log.Printf("Discord service disabled via command line flag")
	}

	return nil
}

// StopServices stops all background services
func (c *Controller) StopServices() error {
	if c.discordService != nil {
		return c.discordService.Stop()
	}
	return nil
}

// IndexHandler serves our main HTML page (extracted from main.go)
func (c *Controller) IndexHandler(w http.ResponseWriter, r *http.Request) {
	// Render index template (no data needed for static content)
	c.renderTemplate(w, "views/index.html", nil)
}

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
		"endpoints": []string{"/", "/hello", "/chat", "/health"},
		"chatbot":   chatbotStatus,
		"discord":   discordStatus,
	}

	json.NewEncoder(w).Encode(health)
}

// renderTemplate renders an HTML template with data
func (c *Controller) renderTemplate(w http.ResponseWriter, templatePath string, data interface{}) {
	// Get absolute path
	absPath, err := filepath.Abs(templatePath)
	if err != nil {
		log.Printf("Error getting absolute path for template %s: %v", templatePath, err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	// Parse template
	tmpl, err := template.ParseFiles(absPath)
	if err != nil {
		log.Printf("Error parsing template %s: %v", templatePath, err)
		http.Error(w, "Template parsing error", http.StatusInternalServerError)
		return
	}

	// Set content type
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	// Execute template
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing template %s: %v", templatePath, err)
		return
	}
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

// generateSessionID creates a simple session ID
func (c *Controller) generateSessionID() string {
	// Simple session ID generation - in production, use proper UUID
	return fmt.Sprintf("sess_%d", time.Now().UnixNano())
}
