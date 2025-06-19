package controllers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"chatbot/services"
)

// Controller handles all the business logic (extracted from main.go Server methods)
type Controller struct {
	chatbot        *services.Chatbot
	discordService *services.DiscordService
}

// NewController creates a new controller instance
func NewController(llmProvider services.LLMProvider, enableSearch bool, enableRAG bool) *Controller {
	// Initialize chatbot service with specified provider and search capability
	chatbot := services.NewChatbot(llmProvider, enableSearch, enableRAG)

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

// generateSessionID creates a simple session ID
func (c *Controller) generateSessionID() string {
	// Simple session ID generation - in production, use proper UUID
	return fmt.Sprintf("sess_%d", time.Now().UnixNano())
}
