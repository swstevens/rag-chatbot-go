package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"chatbot/controllers"
)

// Server struct - now using MVC controller
type Server struct {
	router     *mux.Router
	port       string
	controller *controllers.Controller
}

// NewServer creates a new server instance with MVC structure
func NewServer(port string) *Server {
	return &Server{
		router:     mux.NewRouter(),
		port:       port,
		controller: controllers.NewController(),
	}
}

// setupRoutes configures all our endpoints using the controller
func (s *Server) setupRoutes() {
	// Static file serving for CSS and other assets
	s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

	// All routes handled by the controller
	s.router.HandleFunc("/", s.controller.IndexHandler).Methods("GET")
	s.router.HandleFunc("/hello", s.controller.HelloHandler).Methods("POST")
	s.router.HandleFunc("/health", s.controller.HealthHandler).Methods("GET")
}

// Start begins the HTTP server
func (s *Server) Start() error {
	s.setupRoutes()

	// Setup CORS for future frontend integration
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(s.router)

	log.Printf("🚀 Starting RAG Chatbot Server (MVC Separated) on port %s", s.port)
	log.Printf("📱 Web interface: http://localhost%s", s.port)
	log.Printf("🔗 API endpoint: http://localhost%s/hello", s.port)
	log.Printf("❤️  Health check: http://localhost%s/health", s.port)

	return http.ListenAndServe(s.port, handler)
}

func main() {
	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = ":8080"
	}

	// Create and start server with MVC structure
	server := NewServer(port)

	log.Printf("MVC Separation Complete:")
	log.Printf("✅ Models: Data structures extracted")
	log.Printf("✅ Controllers: Business logic extracted")
	log.Printf("✅ Views: Static files (HTML/CSS)")

	if err := server.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
