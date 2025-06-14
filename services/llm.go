package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Stream  bool                   `json:"stream"`
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
		model = "tinyllama:latest" // Optimized for Raspberry Pi
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

	// Create request
	request := OllamaRequest{
		Model:  l.model,
		Prompt: prompt,
		Stream: false,
		Options: map[string]interface{}{
			"temperature":    0.7,
			"max_tokens":     300, // Reduced for Pi performance
			"top_p":          0.9,
			"repeat_penalty": 1.1,
			"num_ctx":        1024, // Smaller context window for Pi
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

	// Parse response
	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if ollamaResp.Error != "" {
		return "", fmt.Errorf("LLM returned error: %s", ollamaResp.Error)
	}

	return ollamaResp.Response, nil
}

// buildPrompt constructs a prompt for the LLM with context and history
func (l *LLMService) buildPrompt(message string, context []string, history []models.ChatMessage) string {
	var prompt bytes.Buffer

	// System prompt
	prompt.WriteString("You are a helpful AI assistant that answers questions based on provided context. ")
	prompt.WriteString("Use the context information to provide accurate and relevant answers. ")
	prompt.WriteString("If the context doesn't contain relevant information, say so and provide a general helpful response.\n\n")

	// Add context if available
	if len(context) > 0 {
		prompt.WriteString("Context information:\n")
		for i, ctx := range context {
			prompt.WriteString(fmt.Sprintf("%d. %s\n", i+1, ctx))
		}
		prompt.WriteString("\n")
	}

	// Add conversation history (limit to last 6 messages to avoid token limits)
	if len(history) > 0 {
		prompt.WriteString("Previous conversation:\n")
		start := 0
		if len(history) > 6 {
			start = len(history) - 6
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

	// Add current user message
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
