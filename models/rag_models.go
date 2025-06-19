package models

import "time"

// RAGDocument represents a document chunk stored in the vector database
type RAGDocument struct {
	ID       string   `json:"id"`
	Content  string   `json:"content"`
	Source   string   `json:"source"`
	Metadata Metadata `json:"metadata"`
	Score    float64  `json:"score,omitempty"` // Similarity score
}

// RAGQuery represents a query to the RAG system
type RAGQuery struct {
	Query     string   `json:"query"`
	Context   []string `json:"context,omitempty"`
	ChannelID string   `json:"channel_id,omitempty"`
	Limit     int      `json:"limit,omitempty"`
	Threshold float64  `json:"threshold,omitempty"` // Minimum similarity score
}

// RAGRequest represents a request to the RAG system
type RAGRequest struct {
	BaseRequest
	Query     string  `json:"query"`
	ChannelID string  `json:"channel_id,omitempty"`
	Limit     int     `json:"limit,omitempty"`
	Threshold float64 `json:"threshold,omitempty"`
}

// RAGResponse represents the response from RAG system
type RAGResponse struct {
	BaseResponse
	Documents []RAGDocument `json:"documents"`
	Query     string        `json:"query"`
	Context   []string      `json:"context"`
	Total     int           `json:"total"` // Total documents found
}

// RAGConfig represents RAG service configuration
type RAGConfig struct {
	Enabled          bool   `json:"enabled"`
	DataPath         string `json:"data_path"`
	CollectionName   string `json:"collection_name"`
	EmbeddingEnabled bool   `json:"embedding_enabled"`
	ChunkSize        int    `json:"chunk_size"`
	ChunkOverlap     int    `json:"chunk_overlap"`
}

// RAGStatus represents RAG service status
type RAGStatus struct {
	BaseResponse
	Config         RAGConfig         `json:"config"`
	Initialized    bool              `json:"initialized"`
	DocumentCount  int               `json:"document_count,omitempty"`
	DiscordContext *DiscordRAGStatus `json:"discord_context,omitempty"`
	SupportedTypes []string          `json:"supported_file_types"`
	IndexedAt      time.Time         `json:"indexed_at,omitempty"`
}

// DiscordRAGStatus represents Discord context status for RAG
type DiscordRAGStatus struct {
	ChannelsTracked int `json:"channels_tracked"`
	TotalMessages   int `json:"total_messages"`
	MaxHistory      int `json:"max_history"`
}

// DocumentIndexRequest represents a request to index documents
type DocumentIndexRequest struct {
	BaseRequest
	Path      string   `json:"path,omitempty"`
	Files     []string `json:"files,omitempty"`
	Recursive bool     `json:"recursive"`
	Force     bool     `json:"force"` // Force reindexing
}

// DocumentIndexResponse represents the response from document indexing
type DocumentIndexResponse struct {
	BaseResponse
	IndexedCount int      `json:"indexed_count"`
	SkippedCount int      `json:"skipped_count"`
	ErrorCount   int      `json:"error_count"`
	Files        []string `json:"files,omitempty"`
	Duration     string   `json:"duration"`
}
