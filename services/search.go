package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// SearchResult represents a web search result
type SearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Published   string `json:"published,omitempty"`
}

// SearchResponse represents the full search response
type SearchResponse struct {
	Query   string         `json:"query"`
	Results []SearchResult `json:"results"`
	Count   int            `json:"count"`
}

// BraveSearchResponse represents the API response from Brave Search
type BraveSearchResponse struct {
	Web struct {
		Results []struct {
			Title       string `json:"title"`
			URL         string `json:"url"`
			Description string `json:"description"`
			Published   string `json:"published"`
		} `json:"results"`
	} `json:"web"`
}

// SearchService handles web search operations
type SearchService struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	enabled    bool
}

// NewSearchService creates a new search service instance
func NewSearchService() *SearchService {
	apiKey := os.Getenv("BRAVE_SEARCH_API_KEY")
	baseURL := "https://api.search.brave.com/res/v1/web/search"

	return &SearchService{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		enabled: apiKey != "",
	}
}

// IsEnabled checks if the search service is properly configured
func (s *SearchService) IsEnabled() bool {
	return s.enabled && s.apiKey != ""
}

// Search performs a web search using Brave Search API
func (s *SearchService) Search(query string, maxResults int) (*SearchResponse, error) {
	if !s.IsEnabled() {
		return nil, fmt.Errorf("search service not enabled - missing BRAVE_SEARCH_API_KEY")
	}

	// Clean and prepare the query
	cleanQuery := strings.TrimSpace(query)
	if cleanQuery == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	// Limit max results to reasonable number
	if maxResults <= 0 || maxResults > 10 {
		maxResults = 5
	}

	// Build request URL
	params := url.Values{}
	params.Add("q", cleanQuery)
	params.Add("count", fmt.Sprintf("%d", maxResults))
	params.Add("offset", "0")
	params.Add("freshness", "pw") // Past week for more current results
	params.Add("text_decorations", "false")

	requestURL := fmt.Sprintf("%s?%s", s.baseURL, params.Encode())

	// Create HTTP request
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create search request: %w", err)
	}

	// Set headers
	req.Header.Set("X-Subscription-Token", s.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "RAG-Chatbot/1.0")

	// Make request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search API error %d: %s", resp.StatusCode, string(body))
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read search response: %w", err)
	}

	// Parse response
	var braveResp BraveSearchResponse
	if err := json.Unmarshal(body, &braveResp); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	// Convert to our format
	searchResp := &SearchResponse{
		Query:   cleanQuery,
		Results: make([]SearchResult, 0, len(braveResp.Web.Results)),
		Count:   len(braveResp.Web.Results),
	}

	for _, result := range braveResp.Web.Results {
		searchResp.Results = append(searchResp.Results, SearchResult{
			Title:       result.Title,
			URL:         result.URL,
			Description: result.Description,
			Published:   result.Published,
		})
	}

	return searchResp, nil
}

// SearchForContext performs a search and formats results as context for LLM
func (s *SearchService) SearchForContext(query string, maxResults int) ([]string, error) {
	searchResp, err := s.Search(query, maxResults)
	if err != nil {
		return nil, err
	}

	if len(searchResp.Results) == 0 {
		return []string{}, nil
	}

	context := make([]string, 0, len(searchResp.Results))
	for i, result := range searchResp.Results {
		contextEntry := fmt.Sprintf("[Search Result %d] %s - %s (Source: %s)",
			i+1, result.Title, result.Description, result.URL)
		context = append(context, contextEntry)
	}

	return context, nil
}

// QuickSearch performs a search and returns a summary string
func (s *SearchService) QuickSearch(query string) (string, error) {
	searchResp, err := s.Search(query, 3)
	if err != nil {
		return "", err
	}

	if len(searchResp.Results) == 0 {
		return fmt.Sprintf("No search results found for: %s", query), nil
	}

	summary := fmt.Sprintf("Found %d results for '%s':\n", len(searchResp.Results), query)
	for i, result := range searchResp.Results {
		summary += fmt.Sprintf("%d. %s - %s\n", i+1, result.Title, result.Description)
	}

	return summary, nil
}

// ShouldSearch determines if a query would benefit from web search
func (s *SearchService) ShouldSearch(message string) bool {
	if !s.IsEnabled() {
		return false
	}

	message = strings.ToLower(message)

	// Keywords that suggest current/recent information needs
	searchKeywords := []string{
		"latest", "recent", "current", "today", "news", "update", "2024", "2025",
		"what's happening", "breaking", "trending", "now", "currently",
		"this week", "this month", "this year", "status", "price",
		"weather", "stock", "rate", "election", "event",
	}

	for _, keyword := range searchKeywords {
		if strings.Contains(message, keyword) {
			return true
		}
	}

	// Question patterns that often need current info
	questionPatterns := []string{
		"what is the", "how much", "where is", "when did", "who won",
		"what happened", "how to", "why is", "is there",
	}

	for _, pattern := range questionPatterns {
		if strings.Contains(message, pattern) {
			return true
		}
	}

	return false
}

// GetStatus returns the status of the search service
func (s *SearchService) GetStatus() map[string]interface{} {
	status := map[string]interface{}{
		"base_url": s.baseURL,
		"timeout":  s.httpClient.Timeout.String(),
	}

	if s.IsEnabled() {
		status["status"] = "enabled"
		// Mask API key for security
		if len(s.apiKey) > 8 {
			status["api_key"] = s.apiKey[:4] + "..." + s.apiKey[len(s.apiKey)-4:]
		} else {
			status["api_key"] = "***"
		}
	} else {
		status["status"] = "disabled"
		status["error"] = "BRAVE_SEARCH_API_KEY not set"
	}

	return status
}

// TestSearch performs a test search to verify the service is working
func (s *SearchService) TestSearch() error {
	if !s.IsEnabled() {
		return fmt.Errorf("search service not enabled")
	}

	_, err := s.Search("test query", 1)
	return err
}
