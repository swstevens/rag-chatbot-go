package models

import "time"

// SearchQuery represents a web search query
type SearchQuery struct {
	Query      string   `json:"query"`
	MaxResults int      `json:"max_results,omitempty"`
	Freshness  string   `json:"freshness,omitempty"` // "pw" for past week, etc.
	SafeSearch bool     `json:"safe_search,omitempty"`
	Filters    Metadata `json:"filters,omitempty"`
}

// SearchRequest represents a request to the search service
type SearchRequest struct {
	BaseRequest
	SearchQuery
}

// SearchResult represents a web search result
type SearchResult struct {
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	Published   string    `json:"published,omitempty"`
	Score       float64   `json:"score,omitempty"`
	Snippet     string    `json:"snippet,omitempty"`
	Domain      string    `json:"domain,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// SearchResponse represents the full search response
type SearchResponse struct {
	BaseResponse
	Query    string         `json:"query"`
	Results  []SearchResult `json:"results"`
	Count    int            `json:"count"`
	Duration string         `json:"duration"`
	Provider string         `json:"provider"` // "brave", "google", etc.
	Cached   bool           `json:"cached,omitempty"`
}

// SearchConfig represents search service configuration
type SearchConfig struct {
	Enabled      bool   `json:"enabled"`
	Provider     string `json:"provider"`
	APIKey       string `json:"api_key,omitempty"`
	BaseURL      string `json:"base_url"`
	MaxResults   int    `json:"max_results"`
	CacheEnabled bool   `json:"cache_enabled"`
	CacheTTL     string `json:"cache_ttl"`
}

// SearchStatus represents search service status
type SearchStatus struct {
	BaseResponse
	Config     SearchConfig `json:"config"`
	Available  bool         `json:"available"`
	LastQuery  time.Time    `json:"last_query,omitempty"`
	QueryCount int          `json:"query_count"`
}

// SearchSuggestion represents a search suggestion
type SearchSuggestion struct {
	Text  string  `json:"text"`
	Score float64 `json:"score"`
	Type  string  `json:"type"` // "query", "topic", "entity"
}

// SearchSuggestionsRequest represents a request for search suggestions
type SearchSuggestionsRequest struct {
	BaseRequest
	Partial string `json:"partial"`
	Limit   int    `json:"limit,omitempty"`
}

// SearchSuggestionsResponse represents search suggestions response
type SearchSuggestionsResponse struct {
	BaseResponse
	Suggestions []SearchSuggestion `json:"suggestions"`
	Partial     string             `json:"partial"`
}
