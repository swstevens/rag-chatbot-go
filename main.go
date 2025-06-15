package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"chatbot/controllers"
	"chatbot/utils"
)

// Server struct - now using MVC controller with chatbot and Discord services
type Server struct {
	router         *mux.Router
	port           string
	controller     *controllers.Controller
	enableDiscord  bool
}

// NewServer creates a new server instance with MVC structure
func NewServer(port string, enableDiscord bool) *Server {
	return &Server{
		router:        mux.NewRouter(),
		port:          port,
		controller:    controllers.NewController(),
		enableDiscord: enableDiscord,
	}
}

// setupRoutes configures all our endpoints using the controller
func (s *Server) setupRoutes() {
	// Static file serving for CSS and other assets
	s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

	// Web interface routes
	s.router.HandleFunc("/", s.controller.IndexHandler).Methods("GET")
	
	// API routes
	s.router.HandleFunc("/hello", s.controller.HelloHandler).Methods("POST")
	s.router.HandleFunc("/chat", s.controller.ChatHandler).Methods("POST")
	s.router.HandleFunc("/health", s.controller.HealthHandler).Methods("GET")
}

// Start begins the HTTP server and all services
func (s *Server) Start() error {
	s.setupRoutes()

	// Start background services based on flags
	if err := s.controller.StartServices(s.enableDiscord); err != nil {
		log.Printf("Warning: Some services failed to start: %v", err)
	}

	// Setup CORS for future frontend integration
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(s.router)

	log.Printf("🚀 Starting RAG Chatbot Server (Phase 3+ - Multi-Service) on port %s", s.port)
	log.Printf("📱 Web interface: http://localhost%s", s.port)
	log.Printf("💬 Chat API: http://localhost%s/chat", s.port)
	log.Printf("🔗 Hello API: http://localhost%s/hello", s.port)
	log.Printf("❤️  Health check: http://localhost%s/health", s.port)
	
	if s.enableDiscord {
		log.Printf("🤖 Discord bot: Enabled (check logs above for status)")
	} else {
		log.Printf("🤖 Discord bot: Disabled (use --discord flag to enable)")
	}

	return http.ListenAndServe(s.port, handler)
}

// Stop gracefully stops the server and all services
func (s *Server) Stop() error {
	log.Printf("Stopping services...")
	return s.controller.StopServices()
}

func main() {
	// Load environment variables from .env file first
	if err := utils.LoadEnvWithFallback(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// Define command-line flags
	var (
		port          = flag.String("port", ":8080", "Port to run the server on (e.g., :8080)")
		enableDiscord = flag.Bool("discord", false, "Enable Discord bot service")
		showHelp      = flag.Bool("help", false, "Show help information")
	)
	flag.Parse()

	// Show help if requested
	if *showHelp {
		showUsage()
		return
	}

	// Override port from environment if set
	if envPort := os.Getenv("PORT"); envPort != "" {
		*port = envPort
	}

	// Create server with MVC structure
	server := NewServer(*port, *enableDiscord)

	log.Printf("Phase 3+: Multi-Service Architecture")
	log.Printf("✅ Models: Request/Response structures")
	log.Printf("✅ Views: Template system")
	log.Printf("✅ Controllers: HTTP handling + Service orchestration")
	log.Printf("✅ Services: Chatbot + LLM + Discord (optional)")
	log.Printf("⏳ Next: Document processing and real RAG")

	// Show current configuration
	log.Printf("Configuration:")
	log.Printf("  Port: %s", *port)
	log.Printf("  Discord: %v", *enableDiscord)
	if *enableDiscord {
		if os.Getenv("DISCORD_BOT_TOKEN") == "" {
			log.Printf("  ⚠️  DISCORD_BOT_TOKEN not set - Discord will be disabled")
		} else {
			// Show partial token for confirmation (security)
			token := os.Getenv("DISCORD_BOT_TOKEN")
			masked := maskToken(token)
			log.Printf("  ✅ DISCORD_BOT_TOKEN loaded: %s", masked)
		}
	}

	// Set up graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-c
	log.Printf("Received shutdown signal...")
	
	// Graceful shutdown
	if err := server.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
	
	log.Printf("Server stopped gracefully")
}

// maskToken masks sensitive token for logging
func maskToken(token string) string {
	if len(token) < 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

// showUsage displays help information
func showUsage() {
	log.Printf("RAG Chatbot Server - Multi-Service Architecture")
	log.Printf("")
	log.Printf("Usage:")
	log.Printf("  go run main.go [flags]")
	log.Printf("")
	log.Printf("Flags:")
	log.Printf("  --port string      Port to run the server on (default \":8080\")")
	log.Printf("  --discord          Enable Discord bot service (default false)")
	log.Printf("  --help             Show this help information")
	log.Printf("")
	log.Printf("Environment Variables (.env file or system):")
	log.Printf("  PORT                    Override the port (e.g., :3000)")
	log.Printf("  DISCORD_BOT_TOKEN       Discord bot token (required for Discord)")
	log.Printf("  DISCORD_COMMAND_PREFIX  Discord command prefix (default \"!chat \")")
	log.Printf("  LLM_BASE_URL           LLM service URL (default \"http://localhost:11434\")")
	log.Printf("  LLM_MODEL              LLM model name (default \"tinyllama\")")
	log.Printf("")
	log.Printf(".env File Setup:")
	log.Printf("  Create a .env file in the project root with:")
	log.Printf("  DISCORD_BOT_TOKEN=your_actual_bot_token_here")
	log.Printf("  DISCORD_COMMAND_PREFIX=!chat ")
	log.Printf("  LLM_MODEL=tinyllama")
	log.Printf("")
	log.Printf("Examples:")
	log.Printf("  go run main.go                           # HTTP server only")
	log.Printf("  go run main.go --discord                 # HTTP server + Discord bot")
	log.Printf("  go run main.go --port :3000              # Custom port")
	log.Printf("  go run main.go --discord --port :3000    # Both custom port and Discord")
	log.Printf("")
	log.Printf("Discord Setup:")
	log.Printf("  1. Create bot at https://discord.com/developers/applications")
	log.Printf("  2. Get bot token and add to .env file: DISCORD_BOT_TOKEN=your_token")
	log.Printf("  3. Invite bot to your server with message permissions")
	log.Printf("  4. Run with --discord flag")
	log.Printf("  5. Use '!chat <message>' in Discord channels")
}
