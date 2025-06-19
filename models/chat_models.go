package models

import "time"

// HelloRequest represents the hello endpoint request
type HelloRequest struct {
	Name    string `json:"name" form:"name"`
	Message string `json:"message,omitempty" form:"message"`
}

// HelloResponse represents the hello endpoint response
type HelloResponse struct {
	Response string `json:"response"`
	Status   string `json:"status"`
}

// ChatRequest represents an incoming chat request
type ChatRequest struct {
	BaseRequest
	Message string        `json:"message"`
	History []ChatMessage `json:"history,omitempty"`
}

// ChatMessage represents a single message in conversation history
type ChatMessage struct {
	Role      string    `json:"role"` // "user" or "assistant"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// ChatResponse represents the response from the chatbot
type ChatResponse struct {
	BaseResponse
	Message   string    `json:"message"`
	SessionID string    `json:"session_id"`
	Context   []string  `json:"context,omitempty"` // Retrieved document snippets
	Sources   []string  `json:"sources,omitempty"` // Source document names
	Status    string    `json:"status"`            // "success" or "error",
	Timestamp time.Time `json:"timestamp"`         // Response timestamp
}

// LLMProvider represents the type of LLM provider
type LLMProvider string

const (
	ProviderLocal   LLMProvider = "local"
	ProviderChatGPT LLMProvider = "chatgpt"
	ProviderDummy   LLMProvider = "dummy"
)
