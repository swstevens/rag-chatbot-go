package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

// Request/Response models for our API
type HelloRequest struct {
	Name    string `json:"name"`
	Message string `json:"message,omitempty"`
}

type HelloResponse struct {
	Response string `json:"response"`
	Status   string `json:"status"`
}

// Server struct - foundation for future MVC controller
type Server struct {
	router *mux.Router
	port   string
}

// NewServer creates a new server instance
func NewServer(port string) *Server {
	return &Server{
		router: mux.NewRouter(),
		port:   port,
	}
}

// setupRoutes configures all our endpoints
func (s *Server) setupRoutes() {
	// GET / - Index endpoint that returns HTML
	s.router.HandleFunc("/", s.indexHandler).Methods("GET")
	
	// POST /hello - Hello endpoint that processes input
	s.router.HandleFunc("/hello", s.helloHandler).Methods("POST")
	
	// GET /health - Health check (useful for future deployment)
	s.router.HandleFunc("/health", s.healthHandler).Methods("GET")
}

// indexHandler serves the main page
func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	
	html := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Web Request Handler - Phase 1</title>
		<style>
			body { font-family: Arial, sans-serif; margin: 40px; }
			.container { max-width: 600px; }
			h1 { color: #333; }
			.endpoint { background: #f5f5f5; padding: 15px; margin: 10px 0; border-radius: 5px; }
		</style>
	</head>
	<body>
		<div class="container">
			<h1>Phase 1: Web Request Handler</h1>
			<p>Foundation for future MVC RAG chatbot</p>
			
			<h2>Available Endpoints:</h2>
			<div class="endpoint">
				<strong>GET /</strong> - This index page
			</div>
			<div class="endpoint">
				<strong>POST /hello</strong> - Hello endpoint (JSON)
			</div>
			<div class="endpoint">
				<strong>GET /health</strong> - Health check
			</div>
			
			<h2>Test the Hello Endpoint:</h2>
			<p>Send a POST request to <code>/hello</code> with JSON:</p>
			<pre>{"name": "Your Name", "message": "Hello World"}</pre>
		</div>
	</body>
	</html>`
	
	fmt.Fprint(w, html)
}

// helloHandler processes hello requests
func (s *Server) helloHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Parse request body
	var req HelloRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := HelloResponse{
			Response: "Invalid JSON format",
			Status:   "error",
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	
	// Validate required fields
	if strings.TrimSpace(req.Name) == "" {
		w.WriteHeader(http.StatusBadRequest)
		response := HelloResponse{
			Response: "Name field is required",
			Status:   "error",
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	
	// Create response
	greeting := fmt.Sprintf("Hello, %s!", req.Name)
	if req.Message != "" {
		greeting += fmt.Sprintf(" Your message: %s", req.Message)
	}
	
	response := HelloResponse{
		Response: greeting,
		Status:   "success",
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// healthHandler provides health check endpoint
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	health := map[string]interface{}{
		"status":    "healthy",
		"service":   "web-request-handler",
		"version":   "1.0.0",
		"timestamp": fmt.Sprintf("%d", r.Context().Value("timestamp")),
	}
	
	json.NewEncoder(w).Encode(health)
}

// Start configures and starts the HTTP server
func (s *Server) Start() error {
	// Setup routes
	s.setupRoutes()
	
	// Configure CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	
	// Wrap router with CORS handler
	handler := c.Handler(s.router)
	
	// Ensure port has colon prefix
	if !strings.HasPrefix(s.port, ":") {
		s.port = ":" + s.port
	}
	
	log.Printf("Server starting on port %s", s.port)
	log.Printf("Visit http://localhost%s to see the index page", s.port)
	
	return http.ListenAndServe(s.port, handler)
}

func main() {
	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = ":8080"
	}
	
	// Create and start server
	server := NewServer(port)
	log.Printf("Phase 1: Web Request Handler")
	log.Printf("This is the foundation for our future MVC RAG chatbot")
	
	if err := server.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}