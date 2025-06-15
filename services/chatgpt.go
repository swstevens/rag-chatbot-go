package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"chatbot/models"
)

// ChatGPTService handles communication with OpenAI's ChatGPT API
type ChatGPTService struct {
	apiKey        string
	baseURL       string
	model         string
	httpClient    *http.Client
	searchService *SearchService
}

// ChatGPTRequest represents a request to the ChatGPT API
type ChatGPTRequest struct {
	Model       string           `json:"model"`
	Messages    []ChatGPTMessage `json:"messages"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
	Stop        []string         `json:"stop,omitempty"`
}

// ChatGPTMessage represents a message in the ChatGPT format
type ChatGPTMessage struct {
	Role    string `json:"role"` // "system", "user", or "assistant"
	Content string `json:"content"`
}

// ChatGPTResponse represents a response from the ChatGPT API
type ChatGPTResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// NewChatGPTService creates a new ChatGPT service instance
func NewChatGPTService(enableSearch bool) *ChatGPTService {
	apiKey := os.Getenv("OPENAI_API_KEY")
	baseURL := os.Getenv("OPENAI_BASE_URL")
	model := os.Getenv("OPENAI_MODEL")

	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model == "" {
		model = "gpt-3.5-turbo" // Default to most cost-effective model
	}

	var searchService *SearchService
	if enableSearch {
		searchService = NewSearchService()
	}

	return &ChatGPTService{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		searchService: searchService,
	}
}

// GenerateResponse generates a response using ChatGPT
func (c *ChatGPTService) GenerateResponse(message string, context []string, history []models.ChatMessage) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("OpenAI API key not set")
	}

	// Check if we should search for current information
	var searchContext []string
	if c.searchService != nil && c.searchService.IsEnabled() && c.searchService.ShouldSearch(message) {
		searchResults, err := c.searchService.SearchForContext(message, 3)
		if err != nil {
			log.Printf("Search failed: %v", err)
		} else if len(searchResults) > 0 {
			searchContext = searchResults
			log.Printf("Added %d search results to context", len(searchResults))
		}
	}

	// Combine document context and search context
	allContext := append(context, searchContext...)

	// Build messages for ChatGPT format
	messages := c.buildMessages(message, allContext, history)

	// Create request
	request := ChatGPTRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   150, // Keep responses concise like local LLM
		Temperature: 0.7,
		Stop:        []string{"\n\nHuman:", "\nHuman:", "User:"},
	}

	// Convert to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", c.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request to ChatGPT: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var chatGPTResp ChatGPTResponse
	if err := json.Unmarshal(body, &chatGPTResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for API errors
	if chatGPTResp.Error != nil {
		return "", fmt.Errorf("ChatGPT API error: %s", chatGPTResp.Error.Message)
	}

	// Check if we have choices
	if len(chatGPTResp.Choices) == 0 {
		return "", fmt.Errorf("no response choices from ChatGPT")
	}

	response := chatGPTResp.Choices[0].Message.Content

	// Clean up the response
	response = c.cleanResponse(response)

	return response, nil
}

// buildMessages constructs messages array for ChatGPT API
func (c *ChatGPTService) buildMessages(message string, context []string, history []models.ChatMessage) []ChatGPTMessage {
	var messages []ChatGPTMessage

	// System message with instructions
	systemPrompt := "You are a helpful AI assistant. Provide concise, direct answers. " +
		"Keep responses under 2-3 sentences unless more detail is specifically requested. " +
		"Use provided context when relevant. Do not continue the conversation or ask follow-up questions."

	// Add context to system message if available
	if len(context) > 0 {
		systemPrompt += "\n\nContext:\n"
		for _, ctx := range context {
			systemPrompt += "- " + ctx + "\n"
		}

		// If search results are included, mention they're current
		hasSearchResults := false
		for _, ctx := range context {
			if strings.Contains(ctx, "[Search Result") {
				hasSearchResults = true
				break
			}
		}

		if hasSearchResults {
			systemPrompt += "\nNote: Some context includes current web search results for up-to-date information."
		}
	}

	messages = append(messages, ChatGPTMessage{
		Role:    "system",
		Content: systemPrompt,
	})

	// Add conversation history (limit to last 6 messages)
	start := 0
	if len(history) > 6 {
		start = len(history) - 6
	}

	for i := start; i < len(history); i++ {
		msg := history[i]
		role := msg.Role
		if role == "assistant" {
			role = "assistant"
		} else {
			role = "user"
		}

		messages = append(messages, ChatGPTMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	// Add current user message
	messages = append(messages, ChatGPTMessage{
		Role:    "user",
		Content: message,
	})

	return messages
}

// cleanResponse removes unwanted patterns from ChatGPT responses
func (c *ChatGPTService) cleanResponse(response string) string {
	// ChatGPT usually returns clean responses, but apply basic cleanup
	response = cleanLLMResponse(response)
	return response
}

// IsAvailable checks if the ChatGPT service is available
func (c *ChatGPTService) IsAvailable() bool {
	return c.apiKey != ""
}

// GetModel returns the current model
func (c *ChatGPTService) GetModel() string {
	return c.model
}

// SetModel changes the current model
func (c *ChatGPTService) SetModel(model string) {
	c.model = model
}

// GetStatus returns the status of the ChatGPT service
func (c *ChatGPTService) GetStatus() map[string]interface{} {
	status := map[string]interface{}{
		"base_url": c.baseURL,
		"model":    c.model,
		"timeout":  c.httpClient.Timeout.String(),
	}

	if c.IsAvailable() {
		status["status"] = "available"
		// Mask API key for security
		if len(c.apiKey) > 8 {
			status["api_key"] = c.apiKey[:4] + "..." + c.apiKey[len(c.apiKey)-4:]
		} else {
			status["api_key"] = "***"
		}
	} else {
		status["status"] = "unavailable"
		status["error"] = "OPENAI_API_KEY not set"
	}

	// Add search capability status
	if c.searchService != nil {
		status["search"] = c.searchService.GetStatus()
		status["search_enabled"] = c.searchService.IsEnabled()
	} else {
		status["search"] = map[string]interface{}{
			"status": "disabled",
			"note":   "Search not enabled for this instance",
		}
		status["search_enabled"] = false
	}

	return status
}
