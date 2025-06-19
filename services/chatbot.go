package services

import (
	"fmt"
	"log"
	"math/rand"
	"path/filepath"
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
	initialized        bool
	startTime          time.Time
	llmService         *LLMService
	chatgptService     *ChatGPTService
	currentProvider    LLMProvider
	preferredProvider  LLMProvider
	lastProviderCheck  time.Time
	providerCheckCache map[LLMProvider]bool
	ragService         *RAGService
	enableRAG          bool
}

// NewChatbot creates a new chatbot instance with specified provider preference
func NewChatbot(preferredProvider LLMProvider, enableSearch bool, enableRAG bool) *Chatbot {
	var llmService *LLMService
	var chatgptService *ChatGPTService
	var currentProvider LLMProvider

	// Initialize provider cache
	providerCache := make(map[LLMProvider]bool)

	// Only initialize services we might actually use
	switch preferredProvider {
	case ProviderChatGPT:
		// ChatGPT-only mode: Skip Ollama entirely
		chatgptService = NewChatGPTService(enableSearch)

		if chatgptService.IsAvailable() {
			currentProvider = ProviderChatGPT
			providerCache[ProviderChatGPT] = true
			providerCache[ProviderLocal] = false
			searchStatus := "disabled"
			if enableSearch && chatgptService.searchService != nil && chatgptService.searchService.IsEnabled() {
				searchStatus = "enabled"
			}
			log.Printf("ChatGPT-only mode: Using model %s, search %s", chatgptService.GetModel(), searchStatus)
		} else {
			currentProvider = ProviderDummy
			providerCache[ProviderChatGPT] = false
			providerCache[ProviderLocal] = false
			log.Printf("ChatGPT not available (missing OPENAI_API_KEY), using dummy responses")
			log.Printf("No fallback to local LLM in ChatGPT-only mode")
		}

	case ProviderLocal:
		// Local-only mode: Skip ChatGPT entirely
		llmService = NewLLMService("", "")

		if llmService.IsAvailable() {
			currentProvider = ProviderLocal
			providerCache[ProviderLocal] = true
			providerCache[ProviderChatGPT] = false
			log.Printf("Local LLM-only mode: Using model %s", llmService.GetModel())
			if enableSearch {
				log.Printf("Note: Search not available in Local LLM-only mode")
			}
		} else {
			currentProvider = ProviderDummy
			providerCache[ProviderLocal] = false
			providerCache[ProviderChatGPT] = false
			log.Printf("Local LLM not available, using dummy responses")
			log.Printf("No fallback to ChatGPT in Local LLM-only mode")
		}

	default:
		// Auto-detect: Initialize both services
		llmService = NewLLMService("", "")
		chatgptService = NewChatGPTService(enableSearch)

		localAvailable := llmService.IsAvailable()
		chatgptAvailable := chatgptService.IsAvailable()

		providerCache[ProviderLocal] = localAvailable
		providerCache[ProviderChatGPT] = chatgptAvailable

		// Prefer local for speed in auto-detect mode
		if localAvailable {
			currentProvider = ProviderLocal
			log.Printf("Auto-detected local LLM: %s", llmService.GetModel())
		} else if chatgptAvailable {
			currentProvider = ProviderChatGPT
			searchStatus := "disabled"
			if enableSearch && chatgptService.searchService != nil && chatgptService.searchService.IsEnabled() {
				searchStatus = "enabled"
			}
			log.Printf("Auto-detected ChatGPT: %s, search %s", chatgptService.GetModel(), searchStatus)
		} else {
			currentProvider = ProviderDummy
			log.Printf("No LLM services available, using dummy responses")
		}
	}

	var ragService *RAGService
	if enableRAG {
		ragService = NewRAGService("./data", "chatbot_knowledge", true)
		if err := ragService.Initialize(); err != nil {
			log.Printf("Failed to initialize RAG service: %v", err)
			ragService = nil
			enableRAG = false
		} else {
			// Index documents on startup
			if err := ragService.IndexDocuments(); err != nil {
				log.Printf("Failed to index documents: %v", err)
			}
		}
	}

	log.Printf("Chatbot initialized: provider=%s, preferred=%s", currentProvider, preferredProvider)

	return &Chatbot{
		initialized:        true,
		startTime:          time.Now(),
		llmService:         llmService,
		chatgptService:     chatgptService,
		currentProvider:    currentProvider,
		preferredProvider:  preferredProvider,
		lastProviderCheck:  time.Now(),
		providerCheckCache: providerCache,
		ragService:         ragService,
		enableRAG:          enableRAG,
	}
}

// ProcessMessage processes a user message and returns a response
func (c *Chatbot) ProcessMessage(message string, sessionID string, history []models.ChatMessage) models.ChatResponse {
	// Clean the input message
	message = strings.TrimSpace(message)

	context := c.generateContextWithHistory(message, sessionID, history)

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

// generateResponse attempts to generate a response using the current provider (optimized)
func (c *Chatbot) generateResponse(message string, context []string, history []models.ChatMessage) (string, LLMProvider) {
	// Use current provider directly - no fallback checking to reduce latency
	switch c.currentProvider {
	case ProviderChatGPT:
		if c.chatgptService == nil {
			log.Printf("ChatGPT service not initialized")
			return c.generateDummyResponse(message, len(history)), ProviderDummy
		}

		if response, err := c.chatgptService.GenerateResponse(message, context, history); err == nil {
			return response, ProviderChatGPT
		} else {
			log.Printf("ChatGPT failed: %v", err)
			// In forced ChatGPT mode, don't try other providers
			if c.preferredProvider == ProviderChatGPT {
				log.Printf("ChatGPT-only mode: using dummy response (no fallback)")
				return c.generateDummyResponse(message, len(history)), ProviderDummy
			}
		}

	case ProviderLocal:
		if c.llmService == nil {
			log.Printf("Local LLM service not initialized")
			return c.generateDummyResponse(message, len(history)), ProviderDummy
		}

		if response, err := c.llmService.GenerateResponse(message, context, history); err == nil {
			return response, ProviderLocal
		} else {
			log.Printf("Local LLM failed: %v", err)
			// In forced local mode, don't try other providers
			if c.preferredProvider == ProviderLocal {
				log.Printf("Local LLM-only mode: using dummy response (no fallback)")
				return c.generateDummyResponse(message, len(history)), ProviderDummy
			}
		}
	}

	// Fast fallback to dummy - no provider switching during normal operation
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
		if c.chatgptService != nil && c.chatgptService.IsAvailable() {
			return "ChatGPT is available! I can use it for high-quality responses via the OpenAI API."
		} else {
			return "ChatGPT is not configured (missing OPENAI_API_KEY) or not initialized in this mode."
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

func (c *Chatbot) generateContext(message string, sessionID string) []string {
	var context []string

	// Add RAG context if enabled
	if c.enableRAG && c.ragService != nil {
		// Extract channel ID from session ID if it contains discord info
		channelID := ""
		if strings.Contains(sessionID, "discord_") {
			parts := strings.Split(sessionID, "_")
			if len(parts) >= 3 {
				channelID = parts[2] // discord_userID_channelID
			}
		}

		ragResponse, err := c.ragService.Query(message, channelID, 3)
		if err == nil && len(ragResponse.Documents) > 0 {
			for _, doc := range ragResponse.Documents {
				contextEntry := fmt.Sprintf("[RAG Context from %s] %s",
					filepath.Base(doc.Source), doc.Content)
				context = append(context, contextEntry)
			}
		}
	}

	// Add existing dummy context for compatibility
	dummyContext := c.generateDummyContext(message)
	context = append(context, dummyContext...)

	return context
}

func (c *Chatbot) ProcessRAGQuery(query string, channelID string, limit int) *models.RAGResponse {
	if !c.enableRAG {
		return &models.RAGResponse{
			BaseResponse: models.BaseResponse{
				Status:    models.StatusError,
				Error:     "RAG service not enabled",
				Timestamp: time.Now(),
			},
			Documents: []models.RAGDocument{},
			Query:     query,
			Context:   []string{},
			Total:     0,
		}
	}

	response, err := c.ragService.Query(query, channelID, limit)
	if err != nil {
		log.Printf("RAG query failed: %v", err)
		return &models.RAGResponse{
			BaseResponse: models.BaseResponse{
				Status:    models.StatusError,
				Error:     err.Error(),
				Timestamp: time.Now(),
			},
			Documents: []models.RAGDocument{},
			Query:     query,
			Context:   []string{},
			Total:     0,
		}
	}

	return response
}

func (c *Chatbot) generateContextWithHistory(message string, sessionID string, history []models.ChatMessage) []string {
	var context []string

	// Add RAG context if enabled
	if c.enableRAG {
		// Extract channel ID from Discord session ID format: discord_userID_channelID
		channelID := ""
		if strings.Contains(sessionID, "discord_") {
			parts := strings.Split(sessionID, "_")
			if len(parts) >= 3 {
				channelID = parts[2]
			}
		}

		// Get RAG context from documents
		ragResponse, err := c.ragService.Query(message, channelID, 3)
		if err == nil && len(ragResponse.Documents) > 0 {
			for _, doc := range ragResponse.Documents {
				contextEntry := fmt.Sprintf("[Document: %s] %s",
					filepath.Base(doc.Source), doc.Content)
				context = append(context, contextEntry)
			}
		}
	}

	// Add Discord message history as context
	if len(history) > 0 {
		context = append(context, "[Recent Channel Messages]")
		for _, msg := range history {
			context = append(context, msg.Content)
		}
	}

	// Add existing dummy context for compatibility (reduced since we have real context now)
	if len(context) == 0 {
		// Only add dummy context if we have no real context
		dummyContext := c.generateDummyContext(message)
		context = append(context, dummyContext...)
	}

	return context
}

// GetStatus returns the current status of the chatbot
func (c *Chatbot) GetStatus() map[string]interface{} {
	status := map[string]interface{}{
		"status":             "active",
		"mode":               c.getStatusMode(),
		"initialized":        c.initialized,
		"uptime":             time.Since(c.startTime).String(),
		"phase":              "3+",
		"current_provider":   string(c.currentProvider),
		"preferred_provider": string(c.preferredProvider),
	}

	// Add provider-specific status (only for initialized services)
	providers := make(map[string]interface{})

	if c.llmService != nil {
		providers["local"] = c.llmService.GetStatus()
	} else {
		providers["local"] = map[string]interface{}{
			"status": "not_initialized",
			"note":   "Skipped in ChatGPT-only mode",
		}
	}

	if c.chatgptService != nil {
		providers["chatgpt"] = c.chatgptService.GetStatus()
	} else {
		providers["chatgpt"] = map[string]interface{}{
			"status": "not_initialized",
			"note":   "Skipped in Local LLM-only mode",
		}
	}

	status["providers"] = providers

	// Set capabilities based on what's actually available
	capabilities := []string{"conversation_tracking", "context_simulation"}

	switch c.preferredProvider {
	case ProviderChatGPT:
		capabilities = append(capabilities, "chatgpt_only", "no_ollama_resources")
	case ProviderLocal:
		capabilities = append(capabilities, "local_llm_only", "no_chatgpt_resources")
	default:
		capabilities = append(capabilities, "multi_provider_llm", "automatic_fallback")
	}

	if c.ragService != nil {
		status["rag"] = c.ragService.GetStatus()
		status["rag_enabled"] = c.enableRAG
	} else {
		status["rag"] = map[string]interface{}{
			"status": "disabled",
			"note":   "RAG not enabled or failed to initialize",
		}
		status["rag_enabled"] = false
	}

	status["capabilities"] = capabilities
	status["coming_soon"] = []string{
		"document_processing",
		"real_search",
		"full_rag_pipeline",
	}

	return status
}

// getStatusMode returns a descriptive mode string
func (c *Chatbot) getStatusMode() string {
	switch c.preferredProvider {
	case ProviderChatGPT:
		return "chatgpt_only"
	case ProviderLocal:
		return "local_llm_only"
	default:
		return "multi_provider"
	}
}

// refreshProviderStatus checks provider availability (only in auto-detect mode)
func (c *Chatbot) refreshProviderStatus() {
	// Don't refresh in forced modes - they stick to their provider
	if c.preferredProvider == ProviderChatGPT || c.preferredProvider == ProviderLocal {
		return
	}

	c.lastProviderCheck = time.Now()

	// Quick availability check (only for initialized services)
	var localAvailable, chatgptAvailable bool

	if c.llmService != nil {
		localAvailable = c.llmService.IsAvailable()
		c.providerCheckCache[ProviderLocal] = localAvailable
	}

	if c.chatgptService != nil {
		chatgptAvailable = c.chatgptService.IsAvailable()
		c.providerCheckCache[ProviderChatGPT] = chatgptAvailable
	}

	// Only switch if current provider is down and another is available
	if c.currentProvider == ProviderLocal && !localAvailable && chatgptAvailable {
		c.currentProvider = ProviderChatGPT
		log.Printf("Switched to ChatGPT due to local LLM unavailability")
	} else if c.currentProvider == ProviderChatGPT && !chatgptAvailable && localAvailable {
		c.currentProvider = ProviderLocal
		log.Printf("Switched to local LLM due to ChatGPT unavailability")
	}
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
	c.lastProviderCheck = time.Now()

	// Re-initialize provider cache
	if c.providerCheckCache == nil {
		c.providerCheckCache = make(map[LLMProvider]bool)
	}

	// Reset provider availability cache based on what's currently initialized
	if c.llmService != nil {
		c.providerCheckCache[ProviderLocal] = c.llmService.IsAvailable()
	} else {
		c.providerCheckCache[ProviderLocal] = false
	}

	if c.chatgptService != nil {
		c.providerCheckCache[ProviderChatGPT] = c.chatgptService.IsAvailable()
	} else {
		c.providerCheckCache[ProviderChatGPT] = false
	}

	log.Printf("Chatbot state reset. Provider: %s, Initialized services: Local=%v, ChatGPT=%v",
		c.currentProvider,
		c.llmService != nil,
		c.chatgptService != nil)
}

// RefreshProviders attempts to reconnect to all LLM services and update current provider
func (c *Chatbot) RefreshProviders() {
	// Don't refresh in forced provider modes
	if c.preferredProvider == ProviderChatGPT || c.preferredProvider == ProviderLocal {
		log.Printf("RefreshProviders skipped in forced provider mode: %s", c.preferredProvider)
		return
	}

	// Only check initialized services
	var localAvailable, chatgptAvailable bool

	if c.llmService != nil {
		localAvailable = c.llmService.IsAvailable()
	}

	if c.chatgptService != nil {
		chatgptAvailable = c.chatgptService.IsAvailable()
	}

	// Re-evaluate current provider based on availability and preference (auto-detect only)
	if localAvailable {
		c.currentProvider = ProviderLocal
	} else if chatgptAvailable {
		c.currentProvider = ProviderChatGPT
	} else {
		c.currentProvider = ProviderDummy
	}

	log.Printf("Provider refreshed. Current: %s, Local: %v, ChatGPT: %v", c.currentProvider, localAvailable, chatgptAvailable)
}
