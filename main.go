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

// Server struct - now using MVC controller with chatbot and Discord services
type Server struct {
	router        *mux.Router
	port          string
	controller    *controllers.Controller
	enableDiscord bool
	enableSearch  bool
	llmProvider   services.LLMProvider
}

// NewServer creates a new server instance with MVC structure
func NewServer(port string, enableDiscord bool, llmProvider services.LLMProvider, enableSearch bool) *Server {
	return &Server{
		router:        mux.NewRouter(),
		port:          port,
		controller:    controllers.NewController(llmProvider, enableSearch),
		enableDiscord: enableDiscord,
		enableSearch:  enableSearch,
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

	log.Printf("🧠 LLM Provider: %s", getLLMProviderDescription(s.llmProvider))

	if s.enableSearch {
		log.Printf("🔍 Web Search: Enabled (Brave Search API)")
	} else {
		log.Printf("🔍 Web Search: Disabled (use --search flag to enable)")
	}

	return http.ListenAndServe(s.port, handler)
}

// Stop gracefully stops the server and all services
func (s *Server) Stop() error {
	log.Printf("Stopping services...")
	return s.controller.StopServices()
}

// GetConfig returns the current server configuration
func (s *Server) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"port":          s.port,
		"discord":       s.enableDiscord,
		"search":        s.enableSearch,
		"llm_provider":  string(s.llmProvider),
		"provider_desc": getLLMProviderDescription(s.llmProvider),
	}
}

// IsDiscordEnabled returns whether Discord is enabled
func (s *Server) IsDiscordEnabled() bool {
	return s.enableDiscord
}

// IsSearchEnabled returns whether web search is enabled
func (s *Server) IsSearchEnabled() bool {
	return s.enableSearch
}

// GetLLMProvider returns the current LLM provider
func (s *Server) GetLLMProvider() services.LLMProvider {
	return s.llmProvider
}

// GetPort returns the server port
func (s *Server) GetPort() string {
	return s.port
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
		useChatGPT    = flag.Bool("chatgpt", false, "Use ChatGPT instead of local LLM")
		useLocal      = flag.Bool("local", false, "Force use of local LLM (Ollama)")
		enableSearch  = flag.Bool("search", false, "Enable web search for ChatGPT (requires Brave Search API)")
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

	// Override port from environment if set
	if envPort := os.Getenv("PORT"); envPort != "" {
		*port = envPort
	}

	// Create server with MVC structure
	server := NewServer(*port, *enableDiscord, llmProvider, *enableSearch)

	log.Printf("Phase 3+: Multi-Service Architecture with Multi-Provider LLM + Web Search")
	log.Printf("✅ Models: Request/Response structures")
	log.Printf("✅ Views: Template system")
	log.Printf("✅ Controllers: HTTP handling + Service orchestration")
	log.Printf("✅ Services: Chatbot + Local LLM + ChatGPT + Discord + Web Search (optional)")
	log.Printf("⏳ Next: Document processing and real RAG")

	// Show current configuration
	log.Printf("Configuration:")
	log.Printf("  Port: %s", server.port)
	log.Printf("  Discord: %v", server.enableDiscord)
	log.Printf("  LLM Provider: %s", getLLMProviderDescription(server.llmProvider))
	log.Printf("  Web Search: %v", server.enableSearch)

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
	log.Printf("RAG Chatbot Server - Multi-Service Architecture with Multi-Provider LLM")
	log.Printf("")
	log.Printf("Usage:")
	log.Printf("  go run main.go [flags]")
	log.Printf("")
	log.Printf("Flags:")
	log.Printf("  --port string      Port to run the server on (default \":8080\")")
	log.Printf("  --discord          Enable Discord bot service (default false)")
	log.Printf("  --chatgpt          Use ChatGPT as primary LLM provider (default false)")
	log.Printf("  --local            Force use of local LLM/Ollama (default false)")
	log.Printf("  --search           Enable web search for ChatGPT (default false)")
	log.Printf("  --help             Show this help information")
	log.Printf("")
	log.Printf("LLM Provider Selection:")
	log.Printf("  (no flags)         Auto-detect: Local LLM preferred, ChatGPT fallback")
	log.Printf("  --local            Force local LLM (Ollama), ChatGPT fallback")
	log.Printf("  --chatgpt          Force ChatGPT, local LLM fallback")
	log.Printf("")
	log.Printf("Environment Variables (.env file or system):")
	log.Printf("  PORT                    Override the port (e.g., :3000)")
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
	log.Printf("  DISCORD_COMMAND_PREFIX=!chat ")
	log.Printf("  OPENAI_MODEL=gpt-3.5-turbo")
	log.Printf("  LLM_MODEL=tinyllama")
	log.Printf("")
	log.Printf("Examples:")
	log.Printf("  go run main.go                              # Auto-detect LLM, HTTP only")
	log.Printf("  go run main.go --discord                    # Auto-detect LLM, Discord enabled")
	log.Printf("  go run main.go --chatgpt                    # Force ChatGPT, HTTP only")
	log.Printf("  go run main.go --chatgpt --search           # ChatGPT with web search")
	log.Printf("  go run main.go --chatgpt --discord --search # ChatGPT + Discord + Search")
	log.Printf("  go run main.go --local                      # Force local LLM")
	log.Printf("  go run main.go --local --discord --port :3000  # Local LLM + Discord + custom port")
	log.Printf("")
	log.Printf("Setup Instructions:")
	log.Printf("")
	log.Printf("Discord Bot Setup:")
	log.Printf("  1. Create bot at https://discord.com/developers/applications")
	log.Printf("  2. Get bot token and add to .env: DISCORD_BOT_TOKEN=your_token")
	log.Printf("  3. Invite bot to your server with message permissions")
	log.Printf("  4. Run with --discord flag")
	log.Printf("  5. Use '!chat <message>' in Discord channels")
	log.Printf("")
	log.Printf("ChatGPT Setup:")
	log.Printf("  1. Get API key from https://platform.openai.com/api-keys")
	log.Printf("  2. Add to .env: OPENAI_API_KEY=sk-your_key_here")
	log.Printf("  3. Run with --chatgpt flag")
	log.Printf("  4. Optional: Add --search flag for web search capability")
	log.Printf("")
	log.Printf("Web Search Setup:")
	log.Printf("  1. Get API key from https://brave.com/search/api/")
	log.Printf("  2. Add to .env: BRAVE_SEARCH_API_KEY=your_key_here")
	log.Printf("  3. Run with --chatgpt --search flags")
	log.Printf("  4. ChatGPT will automatically search for current topics")
	log.Printf("")
	log.Printf("Local LLM Setup:")
	log.Printf("  1. Install Ollama: https://ollama.ai")
	log.Printf("  2. Run: ollama serve")
	log.Printf("  3. Pull model: ollama pull tinyllama")
	log.Printf("  4. Run with --local flag (or let auto-detect)")
}
