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
	"chatbot/services"
	"chatbot/utils"
)

// Server struct - now with HTTPS support
type Server struct {
	router        *mux.Router
	port          string
	httpsPort     string
	controller    *controllers.Controller
	enableDiscord bool
	enableSearch  bool
	enableHTTPS   bool
	enableRAG     bool
	certFile      string
	keyFile       string
	llmProvider   services.LLMProvider
}

// NewServer creates a new server instance with HTTPS support
func NewServer(port string, httpsPort string, enableDiscord bool, llmProvider services.LLMProvider, enableSearch bool, enableHTTPS bool, enableRAG bool) *Server {
	// Get SSL certificate paths from environment
	certFile := os.Getenv("SSL_CERT_FILE")
	keyFile := os.Getenv("SSL_KEY_FILE")

	return &Server{
		router:        mux.NewRouter(),
		port:          port,
		httpsPort:     httpsPort,
		controller:    controllers.NewController(llmProvider, enableSearch, enableRAG),
		enableDiscord: enableDiscord,
		enableSearch:  enableSearch,
		enableHTTPS:   enableHTTPS,
		enableRAG:     enableRAG,
		certFile:      certFile,
		keyFile:       keyFile,
		llmProvider:   llmProvider,
	}
}

// setupRoutes configures all our endpoints using the controller
func (s *Server) setupRoutes() {
	// Static file serving for CSS and other assets
	s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

	// Web interface routes
	s.router.HandleFunc("/", s.controller.IndexHandler).Methods("GET")

	// API routes
	s.router.HandleFunc("/chat", s.controller.ChatHandler).Methods("POST")
	s.router.HandleFunc("/health", s.controller.HealthHandler).Methods("GET")
	if s.enableRAG {
		s.router.HandleFunc("/rag", s.controller.RAGHandler).Methods("POST")
	}
}

// Start begins the HTTP and HTTPS servers and all services
func (s *Server) Start() error {
	s.setupRoutes()

	// Start background services based on flags
	if err := s.controller.StartServices(s.enableDiscord); err != nil {
		log.Printf("Warning: Some services failed to start: %v", err)
	}

	// Setup CORS for future frontend integration
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(s.router)

	log.Printf("🚀 Starting RAG Chatbot Server (Phase 3+ - Multi-Service) on port %s", s.port)
	log.Printf("📱 Web interface: http://localhost%s", s.port)
	log.Printf("💬 Chat API: http://localhost%s/chat", s.port)
	log.Printf("❤️  Health check: http://localhost%s/health", s.port)

	if s.enableDiscord {
		log.Printf("🤖 Discord bot: Enabled (check logs above for status)")
	} else {
		log.Printf("🤖 Discord bot: Disabled (use --discord flag to enable)")
	}

	log.Printf("🧠 LLM Provider: %s", getLLMProviderDescription(s.llmProvider))

	if s.enableSearch {
		log.Printf("🔍 Web Search: Enabled (Brave Search API)")
	} else {
		log.Printf("🔍 Web Search: Disabled (use --search flag to enable)")
	}

	// Start HTTPS server if enabled and certificates are available
	if s.enableHTTPS {
		if s.certFile == "" || s.keyFile == "" {
			log.Printf("❌ HTTPS enabled but SSL_CERT_FILE or SSL_KEY_FILE not set")
			log.Printf("   Set these environment variables:")
			log.Printf("   export SSL_CERT_FILE=\"/path/to/cert.pem\"")
			log.Printf("   export SSL_KEY_FILE=\"/path/to/key.pem\"")
			log.Printf("🔒 HTTPS server: DISABLED (missing certificates)")
		} else {
			log.Printf("🔒 HTTPS server starting on port %s", s.httpsPort)
			log.Printf("🔒 HTTPS Web interface: https://localhost%s", s.httpsPort)
			log.Printf("🔒 HTTPS Chat API: https://localhost%s/chat", s.httpsPort)
			log.Printf("🔒 HTTPS Health check: https://localhost%s/health", s.httpsPort)

			// Start HTTPS server in goroutine
			go func() {
				log.Printf("Starting HTTPS server on %s with cert: %s", s.httpsPort, s.certFile)
				if err := http.ListenAndServeTLS(s.httpsPort, s.certFile, s.keyFile, handler); err != nil {
					log.Printf("HTTPS server failed: %v", err)
				}
			}()
		}
	} else {
		log.Printf("🔒 HTTPS server: Disabled (use --https flag to enable)")
	}

	// Start HTTP server (always runs)
	log.Printf("Starting HTTP server on %s", s.port)
	return http.ListenAndServe(s.port, handler)
}

// Stop gracefully stops the server and all services
func (s *Server) Stop() error {
	log.Printf("Stopping services...")
	return s.controller.StopServices()
}

// GetConfig returns the current server configuration
func (s *Server) GetConfig() map[string]interface{} {
	config := map[string]interface{}{
		"port":          s.port,
		"https_port":    s.httpsPort,
		"discord":       s.enableDiscord,
		"search":        s.enableSearch,
		"https":         s.enableHTTPS,
		"llm_provider":  string(s.llmProvider),
		"provider_desc": getLLMProviderDescription(s.llmProvider),
	}

	if s.enableHTTPS {
		config["ssl_cert_file"] = s.certFile
		config["ssl_key_file"] = s.keyFile
		config["ssl_configured"] = (s.certFile != "" && s.keyFile != "")
	}

	return config
}

// IsDiscordEnabled returns whether Discord is enabled
func (s *Server) IsDiscordEnabled() bool {
	return s.enableDiscord
}

// IsSearchEnabled returns whether web search is enabled
func (s *Server) IsSearchEnabled() bool {
	return s.enableSearch
}

// IsHTTPSEnabled returns whether HTTPS is enabled
func (s *Server) IsHTTPSEnabled() bool {
	return s.enableHTTPS
}

// GetLLMProvider returns the current LLM provider
func (s *Server) GetLLMProvider() services.LLMProvider {
	return s.llmProvider
}

// GetPort returns the HTTP server port
func (s *Server) GetPort() string {
	return s.port
}

// GetHTTPSPort returns the HTTPS server port
func (s *Server) GetHTTPSPort() string {
	return s.httpsPort
}

func main() {
	// Load environment variables from .env file first
	if err := utils.LoadEnvWithFallback(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// Define command-line flags
	var (
		port          = flag.String("port", ":8080", "Port to run the HTTP server on (e.g., :8080)")
		httpsPort     = flag.String("https-port", ":8443", "Port to run the HTTPS server on (e.g., :8443)")
		enableDiscord = flag.Bool("discord", false, "Enable Discord bot service")
		useChatGPT    = flag.Bool("chatgpt", false, "Use ChatGPT instead of local LLM")
		useLocal      = flag.Bool("local", false, "Force use of local LLM (Ollama)")
		enableSearch  = flag.Bool("search", false, "Enable web search for ChatGPT (requires Brave Search API)")
		enableHTTPS   = flag.Bool("https", false, "Enable HTTPS server (requires SSL_CERT_FILE and SSL_KEY_FILE)")
		enableRAG     = flag.Bool("rag", false, "Enable RAG (Retrieval-Augmented Generation) with document indexing")
		showHelp      = flag.Bool("help", false, "Show help information")
	)
	flag.Parse()

	// Show help if requested
	if *showHelp {
		showUsage()
		return
	}

	// Determine LLM provider based on flags
	var llmProvider services.LLMProvider
	if *useChatGPT && *useLocal {
		log.Fatal("Cannot use both --chatgpt and --local flags at the same time")
	} else if *useChatGPT {
		llmProvider = services.ProviderChatGPT
	} else if *useLocal {
		llmProvider = services.ProviderLocal
	} else {
		llmProvider = "" // Auto-detect
	}

	// Override ports from environment if set
	if envPort := os.Getenv("PORT"); envPort != "" {
		*port = envPort
	}
	if envHTTPSPort := os.Getenv("HTTPS_PORT"); envHTTPSPort != "" {
		*httpsPort = envHTTPSPort
	}

	// Create server with HTTPS support
	server := NewServer(*port, *httpsPort, *enableDiscord, llmProvider, *enableSearch, *enableHTTPS, *enableRAG)

	log.Printf("Phase 3+: Multi-Service Architecture with Multi-Provider LLM + Web Search + HTTPS")
	log.Printf("✅ Models: Request/Response structures")
	log.Printf("✅ Views: Template system")
	log.Printf("✅ Controllers: HTTP handling + Service orchestration")
	log.Printf("✅ Services: Chatbot + Local LLM + ChatGPT + Discord + Web Search (optional)")
	log.Printf("✅ Security: HTTPS support with SSL certificates")
	log.Printf("⏳ Next: Document processing and real RAG")

	// Show current configuration
	log.Printf("Configuration:")
	log.Printf("  HTTP Port: %s", server.port)
	log.Printf("  HTTPS Port: %s", server.httpsPort)
	log.Printf("  Discord: %v", server.enableDiscord)
	log.Printf("  LLM Provider: %s", getLLMProviderDescription(server.llmProvider))
	log.Printf("  Web Search: %v", server.enableSearch)
	log.Printf("  HTTPS: %v", server.enableHTTPS)

	if server.enableDiscord {
		if os.Getenv("DISCORD_BOT_TOKEN") == "" {
			log.Printf("  ⚠️  DISCORD_BOT_TOKEN not set - Discord will be disabled")
		} else {
			// Show partial token for confirmation (security)
			token := os.Getenv("DISCORD_BOT_TOKEN")
			masked := maskToken(token)
			log.Printf("  ✅ DISCORD_BOT_TOKEN loaded: %s", masked)
		}
	}

	if server.llmProvider == services.ProviderChatGPT || (*useChatGPT && server.llmProvider == "") {
		if os.Getenv("OPENAI_API_KEY") == "" {
			log.Printf("  ⚠️  OPENAI_API_KEY not set - ChatGPT will not be available")
		} else {
			apiKey := os.Getenv("OPENAI_API_KEY")
			masked := maskToken(apiKey)
			log.Printf("  ✅ OPENAI_API_KEY loaded: %s", masked)
		}
	}

	if server.enableSearch {
		if os.Getenv("BRAVE_SEARCH_API_KEY") == "" {
			log.Printf("  ⚠️  BRAVE_SEARCH_API_KEY not set - Web search will not be available")
		} else {
			apiKey := os.Getenv("BRAVE_SEARCH_API_KEY")
			masked := maskToken(apiKey)
			log.Printf("  ✅ BRAVE_SEARCH_API_KEY loaded: %s", masked)
		}
	}

	if server.enableHTTPS {
		if server.certFile == "" || server.keyFile == "" {
			log.Printf("  ⚠️  SSL certificates not configured:")
			log.Printf("      Set SSL_CERT_FILE and SSL_KEY_FILE environment variables")
			log.Printf("      Example: export SSL_CERT_FILE=/etc/letsencrypt/live/yourdomain.com/fullchain.pem")
			log.Printf("      Example: export SSL_KEY_FILE=/etc/letsencrypt/live/yourdomain.com/privkey.pem")
		} else {
			log.Printf("  ✅ SSL_CERT_FILE: %s", server.certFile)
			log.Printf("  ✅ SSL_KEY_FILE: %s", server.keyFile)
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

// getLLMProviderDescription returns a human-readable description of the LLM provider
func getLLMProviderDescription(provider services.LLMProvider) string {
	switch provider {
	case services.ProviderChatGPT:
		return "ChatGPT (forced)"
	case services.ProviderLocal:
		return "Local LLM/Ollama (forced)"
	default:
		return "Auto-detect (Local LLM preferred, ChatGPT fallback)"
	}
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
	log.Printf("RAG Chatbot Server - Multi-Service Architecture with Multi-Provider LLM + HTTPS")
	log.Printf("")
	log.Printf("Usage:")
	log.Printf("  go run main.go [flags]")
	log.Printf("")
	log.Printf("Flags:")
	log.Printf("  --port string      Port to run the HTTP server on (default \":8080\")")
	log.Printf("  --https-port string Port to run the HTTPS server on (default \":8443\")")
	log.Printf("  --discord          Enable Discord bot service (default false)")
	log.Printf("  --chatgpt          Use ChatGPT as primary LLM provider (default false)")
	log.Printf("  --local            Force use of local LLM/Ollama (default false)")
	log.Printf("  --search           Enable web search for ChatGPT (default false)")
	log.Printf("  --https            Enable HTTPS server (default false)")
	log.Printf("  --rag              Enable RAG with document indexing (default false)")
	log.Printf("  --help             Show this help information")
	log.Printf("")
	log.Printf("LLM Provider Selection:")
	log.Printf("  (no flags)         Auto-detect: Local LLM preferred, ChatGPT fallback")
	log.Printf("  --local            Force local LLM (Ollama), ChatGPT fallback")
	log.Printf("  --chatgpt          Force ChatGPT, local LLM fallback")
	log.Printf("")
	log.Printf("Environment Variables (.env file or system):")
	log.Printf("  PORT                    Override the HTTP port (e.g., :3000)")
	log.Printf("  HTTPS_PORT              Override the HTTPS port (e.g., :8443)")
	log.Printf("  SSL_CERT_FILE           Path to SSL certificate file (required for HTTPS)")
	log.Printf("  SSL_KEY_FILE            Path to SSL private key file (required for HTTPS)")
	log.Printf("  DISCORD_BOT_TOKEN       Discord bot token (required for Discord)")
	log.Printf("  DISCORD_COMMAND_PREFIX  Discord command prefix (default \"!chat \")")
	log.Printf("  OPENAI_API_KEY          OpenAI API key (required for ChatGPT)")
	log.Printf("  OPENAI_MODEL            OpenAI model (default \"gpt-3.5-turbo\")")
	log.Printf("  OPENAI_BASE_URL         OpenAI API URL (default \"https://api.openai.com/v1\")")
	log.Printf("  BRAVE_SEARCH_API_KEY    Brave Search API key (required for web search)")
	log.Printf("  LLM_BASE_URL           Local LLM URL (default \"http://localhost:11434\")")
	log.Printf("  LLM_MODEL              Local LLM model (default \"tinyllama\")")
	log.Printf("")
	log.Printf(".env File Setup:")
	log.Printf("  Create a .env file in the project root with:")
	log.Printf("  DISCORD_BOT_TOKEN=your_discord_bot_token_here")
	log.Printf("  OPENAI_API_KEY=sk-your_openai_api_key_here")
	log.Printf("  BRAVE_SEARCH_API_KEY=your_brave_search_api_key_here")
	log.Printf("  SSL_CERT_FILE=/path/to/your/cert.pem")
	log.Printf("  SSL_KEY_FILE=/path/to/your/key.pem")
	log.Printf("  DISCORD_COMMAND_PREFIX=!chat ")
	log.Printf("  OPENAI_MODEL=gpt-3.5-turbo")
	log.Printf("  LLM_MODEL=tinyllama")
	log.Printf("")
	log.Printf("HTTPS Setup:")
	log.Printf("  1. Get SSL certificates (Let's Encrypt recommended):")
	log.Printf("     sudo apt install certbot")
	log.Printf("     sudo certbot certonly --standalone -d yourdomain.com")
	log.Printf("  2. Set environment variables:")
	log.Printf("     export SSL_CERT_FILE=/etc/letsencrypt/live/yourdomain.com/fullchain.pem")
	log.Printf("     export SSL_KEY_FILE=/etc/letsencrypt/live/yourdomain.com/privkey.pem")
	log.Printf("  3. Run with --https flag")
	log.Printf("")
	log.Printf("Examples:")
	log.Printf("  go run main.go                              # HTTP only, auto-detect LLM")
	log.Printf("  go run main.go --https                      # HTTP + HTTPS, auto-detect LLM")
	log.Printf("  go run main.go --discord                    # HTTP + Discord, auto-detect LLM")
	log.Printf("  go run main.go --chatgpt --https            # HTTP + HTTPS + ChatGPT")
	log.Printf("  go run main.go --chatgpt --search --https   # ChatGPT + web search + HTTPS")
	log.Printf("  go run main.go --discord --https --search   # All features enabled")
	log.Printf("  go run main.go --local --https              # Force local LLM + HTTPS")
	log.Printf("  go run main.go --https --port :3000 --https-port :3443  # Custom ports")
	log.Printf("")
	log.Printf("Quick SSL Setup (for swstevens.duckdns.org):")
	log.Printf("  sudo certbot certonly --standalone -d swstevens.duckdns.org")
	log.Printf("  export SSL_CERT_FILE=/etc/letsencrypt/live/swstevens.duckdns.org/fullchain.pem")
	log.Printf("  export SSL_KEY_FILE=/etc/letsencrypt/live/swstevens.duckdns.org/privkey.pem")
	log.Printf("  go run main.go --https")
	log.Printf("")
	log.Printf("URLs after HTTPS setup:")
	log.Printf("  HTTP:  http://swstevens.duckdns.org:8080/chat")
	log.Printf("  HTTPS: https://swstevens.duckdns.org:8443/chat")
}
