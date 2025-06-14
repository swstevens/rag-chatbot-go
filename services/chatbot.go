package services

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"chatbot/models"
)

// Chatbot handles chat processing and response generation
type Chatbot struct {
	initialized bool
	startTime   time.Time
	llmService  *LLMService
	useLLM      bool
}

// NewChatbot creates a new chatbot instance
func NewChatbot() *Chatbot {
	// Initialize LLM service
	llmBaseURL := os.Getenv("LLM_BASE_URL")
	llmModel := os.Getenv("LLM_MODEL")
	llmService := NewLLMService(llmBaseURL, llmModel)

	// Check if LLM is available (but don't let this determine behavior)
	initialLLMCheck := llmService.IsAvailable()
	if initialLLMCheck {
		log.Printf("LLM service detected! Model: %s", llmService.GetModel())
		log.Printf("Will attempt to use LLM for all responses with fallback to dummy responses")
	} else {
		log.Printf("LLM service not detected at startup (URL: %s)", llmService.baseURL)
		log.Printf("Will still attempt LLM for each message - may succeed if Ollama starts later")
		log.Printf("To enable LLM:")
		log.Printf("1. Install Ollama: https://ollama.ai")
		log.Printf("2. Run: ollama serve")
		log.Printf("3. Pull a model: ollama pull %s", llmService.GetModel())
	}

	return &Chatbot{
		initialized: true,
		startTime:   time.Now(),
		llmService:  llmService,
		useLLM:      true, // Always try LLM, regardless of initial check
	}
}

// ProcessMessage processes a user message and returns a response
func (c *Chatbot) ProcessMessage(message string, sessionID string, history []models.ChatMessage) models.ChatResponse {
	// Clean the input message
	message = strings.TrimSpace(message)

	var response string
	var usedLLM bool

	// ALWAYS try LLM first, regardless of startup state
	context := c.generateDummyContext(message) // Phase 4 will replace with real search

	llmResponse, err := c.llmService.GenerateResponse(message, context, history)
	if err != nil {
		// LLM failed, log and fall back to dummy
		log.Printf("LLM generation failed: %v, using dummy response", err)
		response = c.generateDummyResponse(message, len(history))
		usedLLM = false
	} else {
		// LLM succeeded
		response = llmResponse
		usedLLM = true
		// Update our state to reflect LLM is working
		c.useLLM = true
	}

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

	// Add metadata about which system was used
	if usedLLM {
		log.Printf("Response generated using LLM model: %s", c.llmService.GetModel())
	} else {
		log.Printf("Response generated using dummy fallback")
	}

	return chatResponse
}

// generateDummyResponse creates a dummy response based on the user message (fallback)
func (c *Chatbot) generateDummyResponse(message string, historyLength int) string {
	message = strings.ToLower(message)

	// Response patterns based on message content
	if strings.Contains(message, "hello") || strings.Contains(message, "hi") || strings.Contains(message, "hey") {
		greetings := []string{
			"Hello! I'm your RAG chatbot. I can use local LLM models when available!",
			"Hi there! I'm powered by local AI when possible, with smart fallbacks.",
			"Hey! I'm in Phase 3+ development - now with LLM integration capabilities!",
		}
		return greetings[rand.Intn(len(greetings))]
	}

	if strings.Contains(message, "llm") || strings.Contains(message, "model") {
		if c.useLLM {
			return fmt.Sprintf("I'm currently using the local LLM model: %s. It's working great!", c.llmService.GetModel())
		} else {
			return "I can use local LLM models like Llama2 via Ollama, but none are currently available. I'm using smart dummy responses instead!"
		}
	}

	if strings.Contains(message, "document") || strings.Contains(message, "file") || strings.Contains(message, "search") {
		return "I'm designed for document search and RAG! Right now I'm using dummy context, but I'll soon be able to search through your actual documents and use that information with my LLM."
	}

	if strings.Contains(message, "rag") || strings.Contains(message, "retrieval") {
		return "RAG (Retrieval-Augmented Generation) is my specialty! I combine document search with AI generation. Currently using simulated retrieval, but real document processing is coming in Phase 4!"
	}

	// Default response with LLM status
	if c.useLLM {
		return fmt.Sprintf("I received your message: \"%s\". I'm using local LLM (%s) with dummy document context. Real document search coming soon!", message, c.llmService.GetModel())
	} else {
		return fmt.Sprintf("I received your message: \"%s\". I'm using dummy responses since no local LLM is available. Install Ollama to enable AI responses!", message)
	}
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
	// Check current LLM availability (real-time check)
	currentLLMAvailable := c.llmService.IsAvailable()

	status := map[string]interface{}{
		"status":         "active",
		"mode":           "always_try_llm",
		"initialized":    c.initialized,
		"uptime":         time.Since(c.startTime).String(),
		"phase":          "3+",
		"llm_preference": "always_attempt",
	}

	if currentLLMAvailable {
		status["capabilities"] = []string{
			"local_llm_responses",
			"conversation_tracking",
			"context_simulation",
			"smart_fallback",
		}
		status["llm_status"] = c.llmService.GetStatus()
	} else {
		status["capabilities"] = []string{
			"llm_retry_each_message",
			"conversation_tracking",
			"context_simulation",
			"dummy_fallback",
		}
		status["llm_status"] = map[string]interface{}{
			"status":       "unavailable_now",
			"note":         "Will retry LLM on each message - start Ollama to enable",
			"target_model": c.llmService.GetModel(),
		}
	}

	status["coming_soon"] = []string{
		"document_processing",
		"real_search",
		"full_rag_pipeline",
	}

	return status
}

// IsReady checks if the chatbot is ready to process messages
func (c *Chatbot) IsReady() bool {
	return c.initialized
}

// Reset resets the chatbot state (useful for testing)
func (c *Chatbot) Reset() {
	c.startTime = time.Now()
	c.initialized = true
	c.useLLM = c.llmService.IsAvailable()
}

// RefreshLLMConnection attempts to reconnect to the LLM service
func (c *Chatbot) RefreshLLMConnection() bool {
	c.useLLM = c.llmService.IsAvailable()
	if c.useLLM {
		log.Printf("LLM service reconnected! Using model: %s", c.llmService.GetModel())
	}
	return c.useLLM
}
