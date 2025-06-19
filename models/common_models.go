package models

import "time"

// Response status constants
const (
	StatusSuccess = "success"
	StatusError   = "error"
)

// BaseRequest represents common request fields
type BaseRequest struct {
	SessionID string    `json:"session_id,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// BaseResponse represents common response fields
type BaseResponse struct {
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Metadata represents generic metadata
type Metadata map[string]interface{}
