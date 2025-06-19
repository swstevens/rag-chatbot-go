package services

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"chatbot/models"

	"github.com/philippgille/chromem-go"
)

// RAGService handles document storage and retrieval using chromem-go
type RAGService struct {
	db               *chromem.DB
	collection       *chromem.Collection
	initialized      bool
	dataPath         string
	collectionName   string
	discordMessages  map[string][]*models.DiscordMessage
	messagesMutex    sync.RWMutex
	embeddingEnabled bool
}

// NewRAGService creates a new RAG service instance
func NewRAGService(dataPath, collectionName string, embeddingEnabled bool) *RAGService {
	return &RAGService{
		dataPath:         dataPath,
		collectionName:   collectionName,
		discordMessages:  make(map[string][]*models.DiscordMessage),
		embeddingEnabled: embeddingEnabled,
		initialized:      false,
	}
}

// Initialize sets up the chromem database and collection
func (r *RAGService) Initialize() error {
	// Create chromem database
	db := chromem.NewDB()

	var collection *chromem.Collection
	var err error

	if r.embeddingEnabled {
		// Use OpenAI embeddings if enabled
		openaiAPIKey := os.Getenv("OPENAI_API_KEY")
		if openaiAPIKey == "" {
			log.Printf("OpenAI API key not found, using default embeddings")
			collection, err = db.GetOrCreateCollection(r.collectionName, nil, nil)
		} else {
			// Create embedding function for OpenAI
			embeddingFunc := chromem.NewEmbeddingFuncOpenAI(openaiAPIKey, chromem.EmbeddingModelOpenAI3Small)
			collection, err = db.GetOrCreateCollection(r.collectionName, nil, embeddingFunc)
		}
	} else {
		// Use default embeddings (local)
		collection, err = db.GetOrCreateCollection(r.collectionName, nil, nil)
	}

	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	r.db = db
	r.collection = collection
	r.initialized = true

	log.Printf("RAG service initialized with collection: %s, embedding enabled: %v", r.collectionName, r.embeddingEnabled)
	return nil
}

// IndexDocuments processes and indexes documents from the data folder
func (r *RAGService) IndexDocuments() error {
	if !r.initialized {
		return fmt.Errorf("RAG service not initialized")
	}

	if r.dataPath == "" {
		return fmt.Errorf("data path not set")
	}

	// Check if data path exists
	if _, err := os.Stat(r.dataPath); os.IsNotExist(err) {
		log.Printf("Data path %s does not exist, creating it", r.dataPath)
		if err := os.MkdirAll(r.dataPath, 0755); err != nil {
			return fmt.Errorf("failed to create data path: %w", err)
		}

		// Create example file
		examplePath := filepath.Join(r.dataPath, "example.txt")
		exampleContent := "This is an example document for the RAG system. Add your documents to the data folder to make them searchable."
		if err := os.WriteFile(examplePath, []byte(exampleContent), 0644); err != nil {
			log.Printf("Failed to create example file: %v", err)
		} else {
			log.Printf("Created example file at %s", examplePath)
		}
	}

	var documents []models.RAGDocument

	// Walk through data directory
	err := filepath.WalkDir(r.dataPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and hidden files
		if d.IsDir() || strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		// Process supported file types
		ext := strings.ToLower(filepath.Ext(path))
		if !r.isSupportedFileType(ext) {
			log.Printf("Skipping unsupported file type: %s", path)
			return nil
		}

		content, err := r.extractTextFromFile(path)
		if err != nil {
			log.Printf("Failed to extract text from %s: %v", path, err)
			return nil
		}

		// Create document chunks
		chunks := r.chunkText(content, 500) // 500 character chunks
		for i, chunk := range chunks {
			doc := models.RAGDocument{
				ID:      fmt.Sprintf("%s_chunk_%d", strings.TrimSuffix(d.Name(), ext), i),
				Content: chunk,
				Source:  path,
				Metadata: map[string]interface{}{
					"file_name":    d.Name(),
					"file_path":    path,
					"file_type":    ext,
					"chunk_index":  i,
					"total_chunks": len(chunks),
					"indexed_at":   time.Now().UTC().Format(time.RFC3339),
				},
			}
			documents = append(documents, doc)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk data directory: %w", err)
	}

	// Add documents to collection
	if len(documents) == 0 {
		log.Printf("No documents found to index in %s", r.dataPath)
		return nil
	}

	for _, doc := range documents {
		// Convert metadata to map[string]string for chromem-go
		metadata := make(map[string]string)
		for k, v := range doc.Metadata {
			metadata[k] = fmt.Sprintf("%v", v)
		}

		err := r.collection.AddDocument(context.Background(), chromem.Document{
			ID:       doc.ID,
			Content:  doc.Content,
			Metadata: metadata,
		})
		if err != nil {
			log.Printf("Failed to add document %s: %v", doc.ID, err)
			continue
		}
	}

	log.Printf("Indexed %d document chunks from %s", len(documents), r.dataPath)
	return nil
}

// Query searches for relevant documents and context
func (r *RAGService) Query(query string, channelID string, limit int) (*models.RAGResponse, error) {
	if !r.initialized {
		return nil, fmt.Errorf("RAG service not initialized")
	}

	if limit <= 0 {
		limit = 5
	}

	// Get Discord message context if available
	var msgContext []string
	if channelID != "" {
		msgContext = r.getDiscordContext(channelID, 10)
	}

	// Search documents
	results, err := r.collection.Query(context.Background(), query, limit, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query collection: %w", err)
	}

	// Convert results to our format
	var documents []models.RAGDocument
	for _, result := range results {
		// Convert metadata back to map[string]interface{}
		metadata := make(map[string]interface{})
		for k, v := range result.Metadata {
			metadata[k] = v
		}

		doc := models.RAGDocument{
			ID:       result.ID,
			Content:  result.Content,
			Source:   r.getSourceFromMetadata(metadata),
			Metadata: metadata,
			Score:    result.Similarity, // chromem-go uses Similarity field
		}
		documents = append(documents, doc)
	}

	return &models.RAGResponse{
		Documents: documents,
		Query:     query,
		Context:   msgContext,
		Timestamp: time.Now(),
		Total:     len(documents),
		BaseResponse: models.BaseResponse{
			Status:    models.StatusSuccess,
			Timestamp: time.Now(),
		},
	}, nil
}

// AddDiscordMessage stores Discord message for context
func (r *RAGService) AddDiscordMessage(channelID string, message *models.DiscordMessage) {
	r.messagesMutex.Lock()
	defer r.messagesMutex.Unlock()

	// Initialize channel if not exists
	if r.discordMessages[channelID] == nil {
		r.discordMessages[channelID] = make([]*models.DiscordMessage, 0)
	}

	// Add message to front
	r.discordMessages[channelID] = append([]*models.DiscordMessage{message}, r.discordMessages[channelID]...)

	// Keep only last 10 messages
	if len(r.discordMessages[channelID]) > 10 {
		r.discordMessages[channelID] = r.discordMessages[channelID][:10]
	}
}

// getDiscordContext retrieves recent Discord messages for context
func (r *RAGService) getDiscordContext(channelID string, limit int) []string {
	r.messagesMutex.RLock()
	defer r.messagesMutex.RUnlock()

	messages, exists := r.discordMessages[channelID]
	if !exists || len(messages) == 0 {
		return []string{}
	}

	var msgContext []string
	maxMessages := limit
	if len(messages) < maxMessages {
		maxMessages = len(messages)
	}

	for i := 0; i < maxMessages; i++ {
		msg := messages[i]
		if !msg.IsBot && len(strings.TrimSpace(msg.Content)) > 10 {
			msgContext = append(msgContext, fmt.Sprintf("%s: %s", msg.Author, msg.Content))
		}
	}

	// Reverse to get chronological order
	for i, j := 0, len(msgContext)-1; i < j; i, j = i+1, j-1 {
		msgContext[i], msgContext[j] = msgContext[j], msgContext[i]
	}

	return msgContext
}

// isSupportedFileType checks if file type is supported
func (r *RAGService) isSupportedFileType(ext string) bool {
	supportedTypes := map[string]bool{
		".txt":  true,
		".md":   true,
		".json": true,
		".csv":  true,
		".log":  true,
		".yml":  true,
		".yaml": true,
	}
	return supportedTypes[ext]
}

// extractTextFromFile extracts text content from file
func (r *RAGService) extractTextFromFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// chunkText splits text into smaller chunks
func (r *RAGService) chunkText(text string, maxChunkSize int) []string {
	if len(text) <= maxChunkSize {
		return []string{text}
	}

	var chunks []string
	sentences := r.splitIntoSentences(text)

	var currentChunk strings.Builder
	for _, sentence := range sentences {
		if currentChunk.Len()+len(sentence) > maxChunkSize && currentChunk.Len() > 0 {
			chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
			currentChunk.Reset()
		}
		currentChunk.WriteString(sentence)
		currentChunk.WriteString(" ")
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
	}

	return chunks
}

// splitIntoSentences splits text into sentences
func (r *RAGService) splitIntoSentences(text string) []string {
	sentenceRegex := regexp.MustCompile(`[.!?]+\s+`)
	sentences := sentenceRegex.Split(text, -1)

	var result []string
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) > 0 {
			result = append(result, sentence)
		}
	}
	return result
}

// getSourceFromMetadata extracts source path from metadata
func (r *RAGService) getSourceFromMetadata(metadata map[string]interface{}) string {
	if source, ok := metadata["file_path"].(string); ok {
		return source
	}
	if name, ok := metadata["file_name"].(string); ok {
		return name
	}
	return "unknown"
}

// GetStatus returns the status of the RAG service
func (r *RAGService) GetStatus() map[string]interface{} {
	status := map[string]interface{}{
		"initialized":       r.initialized,
		"collection_name":   r.collectionName,
		"data_path":         r.dataPath,
		"embedding_enabled": r.embeddingEnabled,
	}

	if r.initialized && r.collection != nil {
		status["status"] = "active"
		// Note: chromem-go doesn't provide document count directly
		status["note"] = "Collection active"
	} else {
		status["status"] = "inactive"
		status["error"] = "Not initialized"
	}

	// Add Discord context stats
	r.messagesMutex.RLock()
	channelCount := len(r.discordMessages)
	totalMessages := 0
	for _, messages := range r.discordMessages {
		totalMessages += len(messages)
	}
	r.messagesMutex.RUnlock()

	status["discord_context"] = map[string]interface{}{
		"channels_tracked": channelCount,
		"total_messages":   totalMessages,
	}

	return status
}

// IsEnabled returns whether the RAG service is enabled and initialized
func (r *RAGService) IsEnabled() bool {
	return r.initialized
}

// Close closes the RAG service (chromem-go handles cleanup automatically)
func (r *RAGService) Close() error {
	r.initialized = false
	return nil
}
