package models

// HelloRequest represents the hello endpoint request (extracted from main.go)
type HelloRequest struct {
	Name    string `json:"name" form:"name"`
	Message string `json:"message,omitempty" form:"message"`
}

// HelloResponse represents the hello endpoint response (extracted from main.go)
type HelloResponse struct {
	Response string `json:"response"`
	Status   string `json:"status"`
}
