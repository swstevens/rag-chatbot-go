package services

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"chatbot/models"
)

// LLMProvider represents the type of LLM provider
type LLMProvider string

const (
	ProviderLocal   LLMProvider = "local"
	ProviderChatGPT LLMProvider = "chatgpt"
	ProviderDummy   LLMProvider = "dummy"
)

// Chatbot handles chat processing and response generation
type Chatbot struct {
	initialized       bool
	startTime         time.Time
	llmService        *LLMService
	chatgptService    *ChatGPTService
	currentProvider   LLMProvider
	preferredProvider LLMProvider
}

// NewChatbot creates a new chatbot instance with specified provider preference
func NewChatbot(preferredProvider LLMProvider) *Chatbot {
	// Initialize both services
	llmService := NewLLMService("", "")
	chatgptService := NewChatGPTService()

	// Determine which provider to use based on preference and availability
	var currentProvider LLMProvider

	switch preferredProvider {
	case ProviderChatGPT:
		if chatgptService.IsAvailable() {
			currentProvider = ProviderChatGPT
			log.Printf("ChatGPT service available! Using model: %s", chatgptService.GetModel())
		} else {
			log.Printf("ChatGPT service not available (missing OPENAI_API_KEY), checking local LLM...")
			if llmService.IsAvailable() {
				currentProvider = ProviderLocal
				log.Printf("Falling back to local LLM: %s", llmService.GetModel())
			} else {
				currentProvider = ProviderDummy
				log.Printf("No LLM services available, using dummy responses")
			}
		}
	case ProviderLocal:
		if llmService.IsAvailable() {
			currentProvider = ProviderLocal
			log.Printf("Local LLM service available! Using model: %s", llmService.GetModel())
		} else {
			log.Printf("Local LLM service not available, checking ChatGPT...")
			if chatgptService.IsAvailable() {
				currentProvider = ProviderChatGPT
				log.Printf("Falling back to ChatGPT: %s", chatgptService.GetModel())
			} else {
				currentProvider = ProviderDummy
				log.Printf("No LLM services available, using dummy responses")
			}
		}
	default:
		// Auto-detect best available (prefer local, then ChatGPT)
		if llmService.IsAvailable() {
			currentProvider = ProviderLocal
			log.Printf("Auto-detected local LLM: %s", llmService.GetModel())
		} else if chatgptService.IsAvailable() {
			currentProvider = ProviderChatGPT
			log.Printf("Auto-detected ChatGPT: %s", chatgptService.GetModel())
		} else {
			currentProvider = ProviderDummy
			log.Printf("No LLM services available, using dummy responses")
		}
	}

	log.Printf("Chatbot initialized with provider: %s (preferred: %s)", currentProvider, preferredProvider)

	return &Chatbot{
		initialized:       true,
		startTime:         time.Now(),
		llmService:        llmService,
		chatgptService:    chatgptService,
		currentProvider:   currentProvider,
		preferredProvider: preferredProvider,
	}
}

// ProcessMessage processes a user message and returns a response
func (c *Chatbot) ProcessMessage(message string, sessionID string, history []models.ChatMessage) models.ChatResponse {
	// Clean the input message
	message = strings.TrimSpace(message)

	// Generate context (Phase 4 will replace with real document search)
	context := c.generateDummyContext(message)

	// Try to generate response using available providers
	response, usedProvider := c.generateResponse(message, context, history)

	// Create response with context and sources
	sources := c.generateDummySources()

	chatResponse := models.ChatResponse{
		Message:   response,
		SessionID: sessionID,
		Context:   context,
		Sources:   sources,
		Status:    "success",
		Timestamp: time.Now(),
	}

	// Log which provider was used
	log.Printf("Response generated using provider: %s", usedProvider)

	return chatResponse
}

// generateResponse attempts to generate a response using available providers with smart fallbacks
func (c *Chatbot) generateResponse(message string, context []string, history []models.ChatMessage) (string, LLMProvider) {
	// Try current/preferred provider first
	switch c.currentProvider {
	case ProviderChatGPT:
		if response, err := c.chatgptService.GenerateResponse(message, context, history); err == nil {
			return response, ProviderChatGPT
		} else {
			log.Printf("ChatGPT failed: %v, trying local LLM", err)
			if response, err := c.llmService.GenerateResponse(message, context, history); err == nil {
				return response, ProviderLocal
			} else {
				log.Printf("Local LLM also failed: %v, using dummy response", err)
			}
		}
	case ProviderLocal:
		if response, err := c.llmService.GenerateResponse(message, context, history); err == nil {
			return response, ProviderLocal
		} else {
			log.Printf("Local LLM failed: %v, trying ChatGPT", err)
			if response, err := c.chatgptService.GenerateResponse(message, context, history); err == nil {
				return response, ProviderChatGPT
			} else {
				log.Printf("ChatGPT also failed: %v, using dummy response", err)
			}
		}
	case ProviderDummy:
		// Already set to dummy mode, just use dummy response
		return c.generateDummyResponse(message, len(history)), ProviderDummy
	}

	// All providers failed, fall back to dummy response
	return c.generateDummyResponse(message, len(history)), ProviderDummy
}

// generateDummyResponse creates a dummy response based on the user message (fallback)
func (c *Chatbot) generateDummyResponse(message string, historyLength int) string {
	message = strings.ToLower(message)

	// Response patterns based on message content
	if strings.Contains(message, "hello") || strings.Contains(message, "hi") || strings.Contains(message, "hey") {
		greetings := []string{
			"Hello! I'm your RAG chatbot. I can use local LLM or ChatGPT when available!",
			"Hi there! I'm powered by AI when possible, with smart fallbacks.",
			"Hey! I'm in Phase 3+ development - now with multiple LLM providers!",
		}
		return greetings[rand.Intn(len(greetings))]
	}

	if strings.Contains(message, "llm") || strings.Contains(message, "model") || strings.Contains(message, "ai") {
		return fmt.Sprintf("I can use multiple AI providers: Local LLM (Ollama), ChatGPT, or smart dummy responses. Currently using: %s", c.currentProvider)
	}

	if strings.Contains(message, "chatgpt") || strings.Contains(message, "openai") {
		if c.chatgptService.IsAvailable() {
			return "ChatGPT is available! I can use it for high-quality responses via the OpenAI API."
		} else {
			return "ChatGPT is not configured (missing OPENAI_API_KEY). I'm using local LLM or dummy responses instead."
		}
	}

	if strings.Contains(message, "document") || strings.Contains(message, "file") || strings.Contains(message, "search") {
		return "I'm designed for document search and RAG! Right now I'm using dummy context, but I'll soon be able to search through your actual documents and use that information with my AI."
	}

	if strings.Contains(message, "rag") || strings.Contains(message, "retrieval") {
		return "RAG (Retrieval-Augmented Generation) is my specialty! I combine document search with AI generation. Currently using simulated retrieval, but real document processing is coming in Phase 4!"
	}

	// Default response with provider status
	return fmt.Sprintf("I received: \"%s\". Currently using %s provider. I support local LLM, ChatGPT, and smart fallbacks!", message, c.currentProvider)
}

// generateDummyContext creates dummy context snippets (Phase 4 will replace with real search)
func (c *Chatbot) generateDummyContext(message string) []string {
	contexts := []string{
		"[Dummy Context] This would normally be a relevant snippet from your documents...",
		"[Mock Context] In the future, this will contain actual text from files that match your query.",
		"[Placeholder Context] Document processing will extract relevant passages here.",
	}

	// Sometimes return empty context to simulate no matches
	if rand.Float64() < 0.3 {
		return []string{}
	}

	// Usually return 1-2 context snippets
	numContexts := rand.Intn(2) + 1
	if numContexts > len(contexts) {
		numContexts = len(contexts)
	}

	return contexts[:numContexts]
}

// generateDummySources creates dummy source file names (Phase 4 will replace with real files)
func (c *Chatbot) generateDummySources() []string {
	sources := []string{
		"dummy-document-1.txt",
		"sample-file.md",
		"mock-content.pdf",
		"placeholder-doc.txt",
	}

	// Sometimes return no sources
	if rand.Float64() < 0.3 {
		return []string{}
	}

	// Usually return 1-2 sources
	numSources := rand.Intn(2) + 1
	if numSources > len(sources) {
		numSources = len(sources)
	}

	return sources[:numSources]
}

// GetStatus returns the current status of the chatbot
func (c *Chatbot) GetStatus() map[string]interface{} {
	status := map[string]interface{}{
		"status":             "active",
		"mode":               "multi_provider",
		"initialized":        c.initialized,
		"uptime":             time.Since(c.startTime).String(),
		"phase":              "3+",
		"current_provider":   string(c.currentProvider),
		"preferred_provider": string(c.preferredProvider),
	}

	// Add provider-specific status
	status["providers"] = map[string]interface{}{
		"local":   c.llmService.GetStatus(),
		"chatgpt": c.chatgptService.GetStatus(),
	}

	status["capabilities"] = []string{
		"multi_provider_llm",
		"automatic_fallback",
		"conversation_tracking",
		"context_simulation",
	}

	status["coming_soon"] = []string{
		"document_processing",
		"real_search",
		"full_rag_pipeline",
	}

	return status
}

// GetCurrentProvider returns the currently active provider
func (c *Chatbot) GetCurrentProvider() LLMProvider {
	return c.currentProvider
}

// IsReady checks if the chatbot is ready to process messages
func (c *Chatbot) IsReady() bool {
	return c.initialized
}

// Reset resets the chatbot state (useful for testing)
func (c *Chatbot) Reset() {
	c.startTime = time.Now()
	c.initialized = true
}

// RefreshProviders attempts to reconnect to all LLM services and update current provider
func (c *Chatbot) RefreshProviders() {
	localAvailable := c.llmService.IsAvailable()
	chatgptAvailable := c.chatgptService.IsAvailable()

	// Re-evaluate current provider based on availability and preference
	switch c.preferredProvider {
	case ProviderChatGPT:
		if chatgptAvailable {
			c.currentProvider = ProviderChatGPT
		} else if localAvailable {
			c.currentProvider = ProviderLocal
		} else {
			c.currentProvider = ProviderDummy
		}
	case ProviderLocal:
		if localAvailable {
			c.currentProvider = ProviderLocal
		} else if chatgptAvailable {
			c.currentProvider = ProviderChatGPT
		} else {
			c.currentProvider = ProviderDummy
		}
	default:
		// Auto-detect
		if localAvailable {
			c.currentProvider = ProviderLocal
		} else if chatgptAvailable {
			c.currentProvider = ProviderChatGPT
		} else {
			c.currentProvider = ProviderDummy
		}
	}

	log.Printf("Provider refreshed. Current: %s, Local: %v, ChatGPT: %v", c.currentProvider, localAvailable, chatgptAvailable)
}
