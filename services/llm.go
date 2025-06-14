package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"net/http"
	"time"

	"chatbot/models"
)

// LLMService handles communication with local LLM models (like Ollama)
type LLMService struct {
	baseURL    string
	model      string
	httpClient *http.Client
	timeout    time.Duration
}

// OllamaRequest represents a request to the Ollama API
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// OllamaResponse represents a response from the Ollama API
type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
	Error    string `json:"error,omitempty"`
}

// NewLLMService creates a new LLM service instance
func NewLLMService(baseURL, model string) *LLMService {
	if baseURL == "" {
		baseURL = "http://localhost:11434" // Default Ollama URL
	}
	if model == "" {
		model = "tinyllama"
	}

	return &LLMService{
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Longer timeout for Pi
		},
		timeout: 120 * time.Second,
	}
}

// GenerateResponse generates a response using the local LLM
func (l *LLMService) GenerateResponse(message string, context []string, history []models.ChatMessage) (string, error) {
	// Build the prompt with context and history
	prompt := l.buildPrompt(message, context, history)
	
	// Create request with tighter controls
	request := OllamaRequest{
		Model:  l.model,
		Prompt: prompt,
		Stream: false,
		Options: map[string]interface{}{
			"temperature":     0.7,
			"max_tokens":      150,  // Much shorter responses
			"top_p":           0.9,
			"repeat_penalty":  1.2,  // Prevent repetition
			"num_ctx":         1024, // Smaller context window for Pi
			"stop":            []string{"\n\nHuman:", "\nHuman:", "User:", "\n\n"}, // Stop tokens
		},
	}

	// Convert to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request
	resp, err := l.httpClient.Post(l.baseURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to make request to LLM: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response and clean it up
	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if ollamaResp.Error != "" {
		return "", fmt.Errorf("LLM returned error: %s", ollamaResp.Error)
	}

	// Clean up the response to prevent self-conversation
	cleanResponse := l.cleanResponse(ollamaResp.Response)
	
	return cleanResponse, nil
}

// cleanResponse removes unwanted patterns from LLM responses
func (l *LLMService) cleanResponse(response string) string {
	// Trim whitespace
	response = strings.TrimSpace(response)
	
	// Stop at common conversation continuation patterns
	stopPatterns := []string{
		"\n\nHuman:",
		"\nHuman:",
		"\n\nUser:",
		"\nUser:",
		"\n\nQ:",
		"\nQ:",
		"\n\nQuestion:",
	}
	
	for _, pattern := range stopPatterns {
		if idx := strings.Index(response, pattern); idx != -1 {
			response = response[:idx]
		}
	}
	
	// Remove trailing punctuation that might indicate continuation
	response = strings.TrimRight(response, ".,!?;:")
	response = strings.TrimSpace(response)
	
	// Limit length as final safeguard (about 2-3 sentences)
	if len(response) > 300 {
		// Try to cut at sentence boundary
		sentences := strings.Split(response, ". ")
		if len(sentences) > 2 {
			response = strings.Join(sentences[:2], ". ") + "."
		} else {
			// Hard cut if no sentence boundaries
			response = response[:297] + "..."
		}
	}
	
	return response
}

// buildPrompt constructs a prompt for the LLM with context and history
func (l *LLMService) buildPrompt(message string, context []string, history []models.ChatMessage) string {
	var prompt bytes.Buffer

	// System prompt with clear instructions
	prompt.WriteString("You are a helpful AI assistant. Provide concise, direct answers. ")
	prompt.WriteString("Keep responses under 2-3 sentences unless more detail is specifically requested. ")
	prompt.WriteString("Use provided context when relevant. ")
	prompt.WriteString("Do not continue the conversation or ask follow-up questions.\n\n")

	// Add context if available
	if len(context) > 0 {
		prompt.WriteString("Context:\n")
		for _, ctx := range context {
			prompt.WriteString(fmt.Sprintf("- %s\n", ctx))
		}
		prompt.WriteString("\n")
	}

	// Add conversation history (limit to last 4 messages to avoid token limits)
	if len(history) > 0 {
		prompt.WriteString("Previous conversation:\n")
		start := 0
		if len(history) > 4 {
			start = len(history) - 4
		}
		
		for i := start; i < len(history); i++ {
			msg := history[i]
			if msg.Role == "user" {
				prompt.WriteString(fmt.Sprintf("Human: %s\n", msg.Content))
			} else if msg.Role == "assistant" {
				prompt.WriteString(fmt.Sprintf("Assistant: %s\n", msg.Content))
			}
		}
		prompt.WriteString("\n")
	}

	// Add current user message with clear instruction
	prompt.WriteString(fmt.Sprintf("Human: %s\n", message))
	prompt.WriteString("Assistant: ")

	return prompt.String()
}

// IsAvailable checks if the LLM service is available
func (l *LLMService) IsAvailable() bool {
	resp, err := l.httpClient.Get(l.baseURL + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == http.StatusOK
}

// GetAvailableModels returns a list of available models
func (l *LLMService) GetAvailableModels() ([]string, error) {
	resp, err := l.httpClient.Get(l.baseURL + "/api/tags")
	if err != nil {
		return nil, fmt.Errorf("failed to get models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var models []string
	for _, model := range result.Models {
		models = append(models, model.Name)
	}

	return models, nil
}

// GetStatus returns the status of the LLM service
func (l *LLMService) GetStatus() map[string]interface{} {
	status := map[string]interface{}{
		"base_url": l.baseURL,
		"model":    l.model,
		"timeout":  l.timeout.String(),
	}

	if l.IsAvailable() {
		status["status"] = "available"
		if models, err := l.GetAvailableModels(); err == nil {
			status["available_models"] = models
		}
	} else {
		status["status"] = "unavailable"
		status["error"] = "Cannot connect to LLM service"
	}

	return status
}

// SetModel changes the current model
func (l *LLMService) SetModel(model string) {
	l.model = model
}

// GetModel returns the current model
func (l *LLMService) GetModel() string {
	return l.model
}
