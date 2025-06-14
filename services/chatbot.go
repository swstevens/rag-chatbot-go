package services

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"chatbot/models"
)

// Chatbot handles chat processing and response generation
type Chatbot struct {
	initialized bool
	startTime   time.Time
}

// NewChatbot creates a new chatbot instance
func NewChatbot() *Chatbot {
	return &Chatbot{
		initialized: true,
		startTime:   time.Now(),
	}
}

// ProcessMessage processes a user message and returns a response
func (c *Chatbot) ProcessMessage(message string, sessionID string, history []models.ChatMessage) models.ChatResponse {
	// Clean the input message
	message = strings.TrimSpace(message)

	// Generate dummy response
	response := c.generateDummyResponse(message, len(history))

	// Create response with dummy context
	context := c.generateDummyContext(message)
	sources := c.generateDummySources()

	return models.ChatResponse{
		Message:   response,
		SessionID: sessionID,
		Context:   context,
		Sources:   sources,
		Status:    "success",
		Timestamp: time.Now(),
	}
}

// generateDummyResponse creates a dummy response based on the user message
func (c *Chatbot) generateDummyResponse(message string, historyLength int) string {
	message = strings.ToLower(message)

	// Response patterns based on message content
	if strings.Contains(message, "hello") || strings.Contains(message, "hi") || strings.Contains(message, "hey") {
		greetings := []string{
			"Hello! I'm a dummy chatbot response. Soon I'll be powered by real RAG technology!",
			"Hi there! I'm currently using mock responses, but I'm learning to search documents.",
			"Hey! I'm in Phase 3 development - dummy responses for now, but real AI coming soon!",
		}
		return greetings[rand.Intn(len(greetings))]
	}

	if strings.Contains(message, "how") && (strings.Contains(message, "are") || strings.Contains(message, "you")) {
		return "I'm doing great! I'm a chatbot in development. I can't search documents yet, but I'm working on it!"
	}

	if strings.Contains(message, "what") && strings.Contains(message, "can") {
		return "Right now I can only give you dummy responses! But soon I'll be able to search through documents and give you real answers based on your content."
	}

	if strings.Contains(message, "document") || strings.Contains(message, "file") || strings.Contains(message, "search") {
		return "I see you're asking about documents! That's exactly what I'm being built for. Once my document processing is complete, I'll be able to search through your files and provide relevant information."
	}

	if strings.Contains(message, "rag") || strings.Contains(message, "retrieval") {
		return "You mentioned RAG! That's my specialty (or will be). RAG stands for Retrieval-Augmented Generation - I'll search documents and use that context to give better answers."
	}

	if strings.Contains(message, "help") {
		return "I'd love to help! Right now I'm just giving dummy responses, but I'm being developed to become a RAG chatbot that can search through documents and provide intelligent answers."
	}

	// Progressive responses based on conversation length
	if historyLength == 0 {
		return fmt.Sprintf("Thanks for your message: \"%s\". I'm a dummy chatbot right now, but I'm learning! This is message #%d in our conversation.", message, historyLength+1)
	} else if historyLength < 3 {
		return fmt.Sprintf("I see this is message #%d! You said: \"%s\". I'm still giving dummy responses, but each message helps me understand the conversation flow.", historyLength+1, message)
	} else {
		return fmt.Sprintf("We're really chatting now! (Message #%d) About your message \"%s\" - I'm practicing conversation handling. Soon I'll have real document search capabilities!", historyLength+1, message)
	}
}

// generateDummyContext creates dummy context snippets
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

// generateDummySources creates dummy source file names
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
	return map[string]interface{}{
		"status":      "active",
		"mode":        "dummy_responses",
		"initialized": c.initialized,
		"uptime":      time.Since(c.startTime).String(),
		"phase":       "3",
		"capabilities": []string{
			"dummy_responses",
			"conversation_tracking",
			"context_simulation",
		},
		"coming_soon": []string{
			"document_processing",
			"real_search",
			"llm_integration",
		},
	}
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
