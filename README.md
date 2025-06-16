# RAG Chatbot Server

A lightweight, multi-service chatbot server built in Go for resource-constrained environments like Raspberry Pi. Features multi-provider LLM support, Discord integration, web search capabilities, and HTTPS support.

## Why Go?

This project is built in Go instead of Python to create a **lightweight server architecture** optimized for resource-constrained devices like the **Raspberry Pi 4 (2GB RAM)**. Go provides:

- **Low memory footprint** - Critical for 2GB RAM environments
- **Single binary deployment** - No dependency management nightmares
- **Excellent concurrency** - Handle multiple Discord/HTTP requests efficiently
- **Fast startup times** - Important for embedded deployments
- **Cross-compilation** - Build on any machine, deploy to ARM devices

## Features

- **Multi-Provider LLM Support**
  - Local LLM via Ollama (tinyllama default for low RAM)
  - OpenAI ChatGPT with optional web search
  - Automatic provider detection and fallback
  - Smart dummy responses when LLMs unavailable

- **Multiple Interfaces**
  - HTTP REST API for web integration
  - Discord bot for community chat
  - Web interface for direct interaction
  - HTTPS support for secure connections

- **Resource Optimization**
  - Lightweight Go architecture for Raspberry Pi
  - Configurable LLM models (tinyllama for 2GB RAM)
  - Efficient memory usage and fast startup
  - Single binary deployment

- **Optional Enhancements**
  - Web search integration (Brave Search API)
  - SSL/TLS encryption with Let's Encrypt
  - Graceful shutdown handling
  - Comprehensive health monitoring

## Quick Start

### Prerequisites

- Go 1.19+ installed
- Ollama running (for local LLM) or OpenAI API key
- Optional: Discord bot token, Brave Search API key, SSL certificates

### Build and Run

```bash
# Clone the repository
git clone <your-repo-url>
cd rag-chatbot-server

# Build the binary (optimized for ARM64/Raspberry Pi)
go build -ldflags="-s -w" -o chatbot main.go

# Run with default settings (HTTP only, auto-detect LLM)
./chatbot

# Or run directly with Go
go run main.go
```

### Production Deployment (Raspberry Pi)

```bash
# Build optimized binary
go build -ldflags="-s -w" -o chatbot main.go

# Run in background with nohup (survives SSH disconnects)
nohup ./chatbot --https --discord --search > chatbot.log 2>&1 &

# Check if running
ps aux | grep chatbot

# View logs
tail -f chatbot.log

# Stop the server
pkill chatbot
```

## Command Line Options

```bash
# Basic usage
./chatbot [flags]

# Available flags:
--port string      Port for HTTP server (default ":8080")
--https-port string Port for HTTPS server (default ":8443") 
--discord          Enable Discord bot service
--chatgpt          Use ChatGPT as primary LLM provider
--local            Force use of local LLM/Ollama
--search           Enable web search for ChatGPT
--https            Enable HTTPS server
--help             Show detailed help information
```

## Configuration Examples

### Lightweight (Raspberry Pi 2GB RAM)
```bash
# Local LLM only - minimal resource usage
./chatbot --local --discord

# Or ChatGPT only - saves local RAM
./chatbot --chatgpt --discord
```

### Full Featured
```bash
# All features enabled
./chatbot --discord --chatgpt --search --https
```

### Development
```bash
# Quick testing
go run main.go --local
```

## API Endpoints

Once running, you can access:

- **Web Interface**: `http://localhost:8080/`
- **Chat API**: `http://localhost:8080/chat`
- **Health Check**: `http://localhost:8080/health`
- **HTTPS** (if enabled): `https://localhost:8443/`

### Example API Usage

```bash
# Simple chat request
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello, how are you?"}'

# With session tracking
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "What is Go programming?",
    "session_id": "user_123"
  }'
```

## Resource Requirements

### Minimum (Local LLM)
- **RAM**: 1.5GB (tinyllama model)
- **CPU**: ARM64 (Raspberry Pi 4)
- **Storage**: 500MB for model + 100MB for application

### Recommended (ChatGPT Mode)
- **RAM**: 256MB (Go binary only)
- **CPU**: Any ARM64/x86_64
- **Network**: Internet connection for API calls
- **Storage**: 50MB for application only

## Architecture

**Phase 3+: Multi-Service Architecture**

- ✅ **Models**: Request/Response structures
- ✅ **Views**: Template system  
- ✅ **Controllers**: HTTP handling + Service orchestration
- ✅ **Services**: Chatbot + Local LLM + ChatGPT + Discord + Web Search
- ✅ **Security**: HTTPS support with SSL certificates
- ⏳ **Next**: Document processing and real RAG

## Setup and Configuration

For detailed setup instructions including:
- Environment configuration
- Discord bot setup
- DuckDNS domain configuration
- Router port forwarding
- HTTPS certificates with Let's Encrypt
- Raspberry Pi optimization

See **[SETUP.md](SETUP.md)** for complete installation guide.

## Project Structure

```
├── main.go              # Server setup and configuration
├── controllers/         # HTTP request handlers
├── services/           # Business logic (LLM, Discord, Search)
├── models/             # Data structures
├── utils/              # Utility functions
├── views/              # HTML templates
├── static/             # CSS and assets
├── .env                # Environment configuration
├── README.md           # This file
└── SETUP.md            # Detailed setup instructions
```

## Quick Troubleshooting

**Server won't start:**
```bash
# Check if port is already in use
sudo netstat -tlnp | grep :8080

# Check logs
tail -f chatbot.log
```

**Out of memory on Raspberry Pi:**
```bash
# Use ChatGPT instead of local LLM to save RAM
./chatbot --chatgpt

# Or use lighter local model
export LLM_MODEL=tinyllama
./chatbot --local
```

**Discord bot not responding:**
- Verify `DISCORD_BOT_TOKEN` is set in `.env`
- Check bot has proper permissions in Discord server
- Ensure bot is online (check Discord developer portal)

For complete troubleshooting and setup instructions, see **[SETUP.md](SETUP.md)**.

## License

[Your chosen license]

## Contributing

1. Fork the repository
2. Create a feature branch
3. Test on Raspberry Pi if possible
4. Submit a pull request

---

**Built with ❤️ in Go for efficient deployment on resource-constrained devices like Raspberry Pi.**