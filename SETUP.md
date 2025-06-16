# RAG Chatbot Server - Complete Setup Guide

This guide covers the complete setup process for the RAG Chatbot Server, including DuckDNS configuration, router port forwarding, SSL certificates, and optimization for Raspberry Pi.

## Table of Contents

1. [System Requirements](#system-requirements)
2. [Basic Installation](#basic-installation)
3. [Environment Configuration](#environment-configuration)
4. [LLM Provider Setup](#llm-provider-setup)
5. [DuckDNS Domain Setup](#duckdns-domain-setup)
6. [Router Port Forwarding](#router-port-forwarding)
7. [HTTPS Setup with Let's Encrypt](#https-setup-with-lets-encrypt)
8. [Discord Bot Setup](#discord-bot-setup)
9. [Web Search Setup](#web-search-setup)
10. [Production Deployment](#production-deployment)
11. [Monitoring and Maintenance](#monitoring-and-maintenance)
12. [Troubleshooting](#troubleshooting)

## System Requirements

### Minimum Requirements
- **OS**: Linux (Raspberry Pi OS, Ubuntu, Debian)
- **Architecture**: ARM64 (Raspberry Pi 4) or x86_64
- **RAM**: 256MB (ChatGPT mode) or 1.5GB (Local LLM mode)
- **Storage**: 1GB free space
- **Network**: Internet connection for external APIs

### Recommended for Raspberry Pi 4 (2GB RAM)
- **Use ChatGPT mode** to minimize local resource usage
- **Or use tinyllama model** for local LLM operation
- **Enable swap** if using local LLM with limited RAM

## Basic Installation

### 1. Install Go

```bash
# Download and install Go 1.21+ for ARM64 (Raspberry Pi)
wget https://go.dev/dl/go1.21.0.linux-arm64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-arm64.tar.gz

# Add to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify installation
go version
```

### 2. Clone and Build Project

```bash
# Clone the repository
git clone <your-repo-url>
cd rag-chatbot-server

# Build optimized binary for production
go build -ldflags="-s -w" -o chatbot main.go

# Test the build
./chatbot --help
```

## Environment Configuration

### 1. Create .env File

```bash
# Create environment configuration file
cp .env.example .env
nano .env
```

### 2. Basic .env Configuration

```bash
# Server Configuration
PORT=:8080
HTTPS_PORT=:8443

# Domain Configuration (replace with your DuckDNS domain)
DOMAIN=yourdomain.duckdns.org

# SSL Certificates (will be configured later)
SSL_CERT_FILE=/opt/chatbot/certs/fullchain.pem
SSL_KEY_FILE=/opt/chatbot/certs/privkey.pem

# Discord Bot (optional - configure in Discord Bot Setup section)
DISCORD_BOT_TOKEN=
DISCORD_COMMAND_PREFIX=!chat 

# OpenAI/ChatGPT (optional - for ChatGPT mode)
OPENAI_API_KEY=
OPENAI_MODEL=gpt-3.5-turbo
OPENAI_BASE_URL=https://api.openai.com/v1

# Web Search (optional - for enhanced ChatGPT responses)
BRAVE_SEARCH_API_KEY=

# Local LLM/Ollama (for local mode)
LLM_BASE_URL=http://localhost:11434
LLM_MODEL=tinyllama
```

## LLM Provider Setup

Choose one or both LLM providers based on your needs and resources.

### Option A: Local LLM with Ollama (Higher RAM Usage)

```bash
# Install Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# Start Ollama service
ollama serve

# In another terminal, pull lightweight model
ollama pull tinyllama

# Test the model
ollama run tinyllama "Hello, how are you?"

# For production, create systemd service
sudo tee /etc/systemd/system/ollama.service << 'EOF'
[Unit]
Description=Ollama Service
After=network.target

[Service]
Type=simple
User=ollama
Group=ollama
ExecStart=/usr/local/bin/ollama serve
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF

# Enable and start service
sudo systemctl enable ollama
sudo systemctl start ollama
```

### Option B: ChatGPT/OpenAI (Lower RAM Usage, Recommended for Pi)

```bash
# Get API key from https://platform.openai.com/api-keys
# Add to .env file:
echo "OPENAI_API_KEY=sk-your_key_here" >> .env

# Test API access
curl -H "Authorization: Bearer sk-your_key_here" \
  https://api.openai.com/v1/models | head -20
```

## DuckDNS Domain Setup

DuckDNS provides free dynamic DNS for your Raspberry Pi.

### 1. Create DuckDNS Account

1. Go to [DuckDNS.org](https://www.duckdns.org/)
2. Sign in with your preferred account (Google, GitHub, etc.)
3. Create a new domain (e.g., `yourdomain.duckdns.org`)
4. Note your token from the account page

### 2. Configure Dynamic DNS Updates

```bash
# Create DuckDNS update script
mkdir -p ~/duckdns
cd ~/duckdns

# Create update script (replace TOKEN and DOMAIN)
tee duck.sh << 'EOF'
#!/bin/bash
TOKEN="your-duckdns-token-here"
DOMAIN="yourdomain"
echo url="https://www.duckdns.org/update?domains=$DOMAIN&token=$TOKEN&ip=" | curl -k -o ~/duckdns/duck.log -K -
EOF

chmod +x duck.sh

# Test the update
./duck.sh
cat duck.log  # Should show "OK"

# Add to crontab for automatic updates every 5 minutes
crontab -e
# Add this line:
# */5 * * * * ~/duckdns/duck.sh >/dev/null 2>&1
```

### 3. Verify DNS Resolution

```bash
# Check if your domain resolves to your public IP
nslookup yourdomain.duckdns.org

# Check your public IP
curl ifconfig.me
```

## Router Port Forwarding

Configure your router to forward traffic to your Raspberry Pi.

### 1. Find Your Raspberry Pi's Local IP

```bash
# Get your Pi's local IP address
ip addr show | grep inet | grep -v 127.0.0.1
# Look for something like 192.168.1.100

# Or use hostname
hostname -I
```

### 2. Access Router Configuration

1. Open your router's web interface (usually `192.168.1.1` or `192.168.0.1`)
2. Log in with admin credentials
3. Find "Port Forwarding" or "Virtual Server" section

### 3. Configure Port Forwarding Rules

Create these port forwarding rules:

| Service | External Port | Internal IP | Internal Port | Protocol |
|---------|---------------|-------------|---------------|----------|
| HTTP | 80 | 192.168.1.100 | 8080 | TCP |
| HTTPS | 443 | 192.168.1.100 | 8443 | TCP |
| Alt HTTP | 8080 | 192.168.1.100 | 8080 | TCP |
| Alt HTTPS | 8443 | 192.168.1.100 | 8443 | TCP |

**Replace `192.168.1.100` with your Pi's actual local IP address.**

### 4. Test Port Forwarding

```bash
# Start your chatbot server
./chatbot --port :8080

# Test from external device (use your phone's mobile data)
# Replace yourdomain.duckdns.org with your actual domain
curl http://yourdomain.duckdns.org:8080/health

# Should return server health information
```

## HTTPS Setup with Let's Encrypt

### Method 1: DNS Challenge (Recommended - Works Behind NAT)

This method doesn't require port 80 to be accessible and works even if your ISP blocks port 80.

```bash
# Install certbot
sudo apt update
sudo apt install -y certbot

# Stop any services using port 80
sudo systemctl stop nginx apache2 2>/dev/null || true

# Run certbot with DNS challenge
sudo certbot certonly \
  --manual \
  --preferred-challenges dns \
  --email your-email@example.com \
  --agree-tos \
  --no-eff-email \
  -d yourdomain.duckdns.org

# Follow the prompts to create DNS TXT record in DuckDNS
# 1. Certbot will show you a TXT record to create
# 2. Go to DuckDNS.org and add the TXT record
# 3. Wait 2-3 minutes for DNS propagation
# 4. Press Enter in certbot to continue
```

#### Adding TXT Record to DuckDNS:

1. Go to [DuckDNS.org](https://www.duckdns.org/)
2. Click on your domain
3. In the "txt record" field, paste the value from certbot
4. Click "update ip" to save
5. Wait 2-3 minutes, then continue with certbot

### Method 2: Standalone Challenge (Requires Port 80 Open)

```bash
# Only use this if port 80 is properly forwarded
sudo certbot certonly \
  --standalone \
  --email your-email@example.com \
  --agree-tos \
  --no-eff-email \
  -d yourdomain.duckdns.org
```

### 3. Copy Certificates to Application Directory

```bash
# Create certificate directory
sudo mkdir -p /opt/chatbot/certs

# Copy certificates with proper permissions
sudo cp /etc/letsencrypt/live/yourdomain.duckdns.org/fullchain.pem /opt/chatbot/certs/
sudo cp /etc/letsencrypt/live/yourdomain.duckdns.org/privkey.pem /opt/chatbot/certs/

# Set proper ownership and permissions
sudo chown $USER:$USER /opt/chatbot/certs/*.pem
sudo chmod 644 /opt/chatbot/certs/fullchain.pem
sudo chmod 600 /opt/chatbot/certs/privkey.pem

# Update .env file
echo "SSL_CERT_FILE=/opt/chatbot/certs/fullchain.pem" >> .env
echo "SSL_KEY_FILE=/opt/chatbot/certs/privkey.pem" >> .env
```

### 4. Set Up Auto-Renewal

```bash
# Create renewal script
sudo tee /opt/chatbot/update-certs.sh << 'EOF'
#!/bin/bash
# Update certificates after renewal
cp /etc/letsencrypt/live/yourdomain.duckdns.org/fullchain.pem /opt/chatbot/certs/
cp /etc/letsencrypt/live/yourdomain.duckdns.org/privkey.pem /opt/chatbot/certs/
chown $USER:$USER /opt/chatbot/certs/*.pem
chmod 644 /opt/chatbot/certs/fullchain.pem
chmod 600 /opt/chatbot/certs/privkey.pem
systemctl restart chatbot 2>/dev/null || pkill -f chatbot
echo "Certificates updated: $(date)"
EOF

sudo chmod +x /opt/chatbot/update-certs.sh

# Add to crontab for auto-renewal
(sudo crontab -l 2>/dev/null; echo "0 12 * * * /usr/bin/certbot renew --quiet && /opt/chatbot/update-certs.sh") | sudo crontab -
```

### 5. Test HTTPS

```bash
# Start server with HTTPS
./chatbot --https

# Test HTTPS endpoint
curl https://yourdomain.duckdns.org:8443/health

# Test SSL certificate
openssl s_client -connect yourdomain.duckdns.org:8443 -servername yourdomain.duckdns.org
```

## Discord Bot Setup

This section provides complete step-by-step instructions for creating a Discord bot, getting the API token, and configuring it with your chatbot server.

### 1. Create Discord Developer Account and Application

#### Step 1.1: Access Discord Developer Portal
1. Open your web browser and go to [https://discord.com/developers/applications](https://discord.com/developers/applications)
2. Log in with your Discord account (create one if you don't have it)
3. You'll see the Discord Developer Portal dashboard

#### Step 1.2: Create New Application
1. Click the **"New Application"** button (blue button in top right)
2. Enter a name for your application (e.g., "RAG Chatbot Server" or "My AI Assistant")
3. Read and accept Discord's Developer Terms of Service
4. Click **"Create"**

#### Step 1.3: Configure Application Details (Optional)
1. In the **"General Information"** tab, you can:
   - Add a description for your bot
   - Upload an avatar/icon for your bot
   - Add tags to categorize your bot
2. Click **"Save Changes"** if you made any modifications

### 2. Create and Configure the Bot

#### Step 2.1: Create Bot User
1. In the left sidebar, click on **"Bot"**
2. Click the **"Add Bot"** button
3. Confirm by clicking **"Yes, do it!"**
4. Your bot is now created!

#### Step 2.2: Get Your Bot Token (IMPORTANT)
1. In the **"Bot"** section, find the **"Token"** area
2. Click **"Reset Token"** (or "Copy" if this is the first time)
3. Confirm the action if prompted
4. **IMMEDIATELY COPY THE TOKEN** - this is your `DISCORD_BOT_TOKEN`
5. ⚠️ **NEVER share this token publicly** - treat it like a password

```bash
# Add it to your .env file immediately:
echo "DISCORD_BOT_TOKEN=your_actual_token_here" >> .env
```

#### Step 2.3: Configure Bot Settings
1. In the **"Bot"** section, configure these settings:
   - **Public Bot**: Turn OFF (unless you want others to invite your bot)
   - **Requires OAuth2 Code Grant**: Keep OFF
   - **Presence Intent**: Turn ON
   - **Server Members Intent**: Turn ON (if you want member information)
   - **Message Content Intent**: Turn ON (required to read message content)

2. Under **"Bot Permissions"**, the bot needs these permissions:
   - Send Messages
   - Read Message History
   - View Channels
   - Use External Emojis (optional)

### 3. Generate Bot Invite Link

#### Step 3.1: Use OAuth2 URL Generator
1. In the left sidebar, click **"OAuth2"** → **"URL Generator"**
2. In **"Scopes"**, select:
   - ☑️ **bot**
3. In **"Bot Permissions"**, select:
   - ☑️ **View Channels**
   - ☑️ **Send Messages**
   - ☑️ **Read Message History**
   - ☑️ **Add Reactions** (optional)
   - ☑️ **Use External Emojis** (optional)

#### Step 3.2: Copy and Use Invite URL
1. Copy the **"Generated URL"** at the bottom
2. Open this URL in a new browser tab
3. Select the Discord server where you want to add the bot
4. Make sure you have **"Manage Server"** permission on that server
5. Click **"Authorize"**
6. Complete any CAPTCHA if prompted

### 4. Configure Environment Variables

#### Step 4.1: Add Bot Token to .env File
```bash
# Navigate to your project directory
cd /path/to/your/rag-chatbot-server

# Add the Discord bot token (replace with your actual token)
echo "DISCORD_BOT_TOKEN=your_actual_bot_token_here" >> .env

# Optional: Customize the command prefix (default is "!chat ")
echo "DISCORD_COMMAND_PREFIX=!chat " >> .env
```

#### Step 4.2: Verify Environment Configuration
```bash
# Check that your .env file contains the Discord token
grep DISCORD_BOT_TOKEN .env

# The output should show your token (partially hidden for security)
```

### 5. Test Discord Bot Integration

#### Step 5.1: Start Server with Discord Enabled
```bash
# Start the chatbot server with Discord integration
./chatbot --discord

# You should see logs indicating Discord connection:
# ✅ Bot is online as: YourBotName
# 📊 Connected to X servers
# Discord bot started successfully! Use '!chat <message>' in Discord
```

#### Step 5.2: Test Bot in Discord Server
1. Go to your Discord server where you invited the bot
2. Find a channel where the bot has permissions
3. Type a test command:
   ```
   !chat Hello, how are you?
   ```
4. The bot should respond with a message

#### Step 5.3: Test Different Commands
```bash
# Test basic greeting
!chat hi

# Test LLM-related query
!chat what model are you using?

# Test longer conversation
!chat tell me about artificial intelligence

# Test with different message lengths to verify the bot handles them properly
```

### 6. Discord Bot Troubleshooting

#### Common Issues and Solutions

**Bot doesn't respond to commands:**
```bash
# Check if bot token is correct
grep DISCORD_BOT_TOKEN .env

# Check server logs for Discord connection
tail -f chatbot.log | grep -i discord

# Verify bot is online in Discord (should show green status)
```

**Permission errors:**
1. Go back to Discord Developer Portal
2. Check **OAuth2** → **URL Generator**
3. Regenerate invite URL with proper permissions
4. Re-invite bot to server

**Bot appears offline:**
```bash
# Check if server is running with Discord enabled
ps aux | grep chatbot

# Restart server with Discord flag
./chatbot --discord

# Check for authentication errors in logs
```

**"Missing Access" or "Unknown Message" errors:**
1. Ensure bot has **"Message Content Intent"** enabled in Developer Portal
2. Verify bot has **"View Channels"** and **"Send Messages"** permissions
3. Check that bot role is positioned correctly in server role hierarchy

#### Verification Steps
```bash
# 1. Verify bot token format (should start with "MTI..." or similar)
echo $DISCORD_BOT_TOKEN | head -c 20

# 2. Test bot token validity
curl -H "Authorization: Bot $DISCORD_BOT_TOKEN" \
     https://discord.com/api/v10/users/@me

# 3. Check server response
./chatbot --discord --help | grep -i discord
```

### 7. Advanced Discord Configuration

#### Step 7.1: Custom Command Prefix
```bash
# Change the command prefix from "!chat " to something else
echo "DISCORD_COMMAND_PREFIX=!ai " >> .env

# Now use: !ai hello instead of !chat hello
```

#### Step 7.2: Multiple Server Management
- The same bot token can be used across multiple Discord servers
- Invite the bot to additional servers using the same OAuth2 URL
- Each server interaction is logged separately

#### Step 7.3: Bot Status and Activity
The bot will show as online when the server is running. You can see:
- **Green dot**: Bot is online and responding
- **Gray dot**: Bot is offline or server is down

### 8. Security Best Practices

#### Protect Your Bot Token
```bash
# Never commit .env file to version control
echo ".env" >> .gitignore

# Set proper file permissions
chmod 600 .env

# Verify token is not in any public files
grep -r "DISCORD_BOT_TOKEN" . --exclude=".env" --exclude="SETUP.md"
```

#### Token Regeneration
If your token is compromised:
1. Go to Discord Developer Portal → Your Application → Bot
2. Click **"Reset Token"**
3. Update your `.env` file with the new token
4. Restart your chatbot server

#### Monitor Bot Activity
```bash
# Monitor Discord-related logs
tail -f chatbot.log | grep -i discord

# Check bot interaction frequency
grep "Discord chat:" chatbot.log | wc -l
```

### 9. Production Discord Bot Deployment

#### Step 9.1: Verify Bot Configuration
```bash
# Create a Discord bot verification script
tee discord-check.sh << 'EOF'
#!/bin/bash
echo "=== Discord Bot Configuration Check ==="

# Check if token is set
if [ -z "$DISCORD_BOT_TOKEN" ]; then
    echo "❌ DISCORD_BOT_TOKEN not set in environment"
else
    echo "✅ DISCORD_BOT_TOKEN is configured"
fi

# Check token format
if [[ $DISCORD_BOT_TOKEN =~ ^[A-Za-z0-9._-]+$ ]]; then
    echo "✅ Token format appears valid"
else
    echo "❌ Token format appears invalid"
fi

# Test bot API access
if curl -s -H "Authorization: Bot $DISCORD_BOT_TOKEN" \
   https://discord.com/api/v10/users/@me > /dev/null; then
    echo "✅ Bot token is valid and working"
else
    echo "❌ Bot token authentication failed"
fi

echo "=== End Check ==="
EOF

chmod +x discord-check.sh
source .env && ./discord-check.sh
```

#### Step 9.2: Production Startup
```bash
# Start with Discord in production
nohup ./chatbot --discord --chatgpt --https > chatbot.log 2>&1 &

# Verify Discord connection in logs
sleep 5
grep -i "Bot is online" chatbot.log
```

🎉 **Your Discord bot is now fully configured and ready to chat with users in your Discord server!**

## Web Search Setup

Enable web search for ChatGPT to provide current information.

### 1. Get Brave Search API Key

1. Go to [Brave Search API](https://brave.com/search/api/)
2. Sign up for an account
3. Create a new API key
4. Note the monthly request limits

### 2. Configure Environment

```bash
# Add API key to .env file
echo "BRAVE_SEARCH_API_KEY=your_brave_api_key_here" >> .env
```

### 3. Test Web Search

```bash
# Start server with ChatGPT and search enabled
./chatbot --chatgpt --search

# Test with a current events question
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "What is the current weather?"}'
```

## Production Deployment

### 1. Create Systemd Service

```bash
# Create service file
sudo tee /etc/systemd/system/chatbot.service << 'EOF'
[Unit]
Description=RAG Chatbot Server
After=network.target

[Service]
Type=simple
User=pi
Group=pi
WorkingDirectory=/home/pi/rag-chatbot-server
ExecStart=/home/pi/rag-chatbot-server/chatbot --chatgpt --discord --https
Restart=always
RestartSec=3
StandardOutput=file:/var/log/chatbot.log
StandardError=file:/var/log/chatbot.log

[Install]
WantedBy=multi-user.target
EOF

# Enable and start service
sudo systemctl enable chatbot
sudo systemctl start chatbot

# Check status
sudo systemctl status chatbot
```

### 2. Alternative: Manual Background Deployment

```bash
# Create startup script
tee start-chatbot.sh << 'EOF'
#!/bin/bash
cd /home/pi/rag-chatbot-server
nohup ./chatbot --chatgpt --discord --https > chatbot.log 2>&1 &
echo "Chatbot started in background"
echo "PID: $(pgrep -f chatbot)"
echo "Logs: tail -f chatbot.log"
EOF

chmod +x start-chatbot.sh

# Start the server
./start-chatbot.sh

# Check if running
ps aux | grep chatbot
```

### 3. Firewall Configuration

```bash
# Configure UFW firewall
sudo ufw enable
sudo ufw allow 8080/tcp
sudo ufw allow 8443/tcp
sudo ufw allow 22/tcp  # SSH
sudo ufw status
```

## Monitoring and Maintenance

### 1. Log Management

```bash
# View real-time logs
tail -f chatbot.log

# Or for systemd service
sudo journalctl -u chatbot -f

# Rotate logs to prevent disk filling
sudo tee /etc/logrotate.d/chatbot << 'EOF'
/var/log/chatbot.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 644 pi pi
    postrotate
        systemctl restart chatbot
    endscript
}
EOF
```

### 2. Resource Monitoring

```bash
# Install monitoring tools
sudo apt install -y htop iotop

# Monitor system resources
htop

# Check memory usage
free -h

# Check disk usage
df -h

# Monitor network usage
sudo iotop
```

### 3. Health Checking

```bash
# Create health check script
tee health-check.sh << 'EOF'
#!/bin/bash
echo "=== Chatbot Health Check ==="
echo "Time: $(date)"
echo ""

# Check if process is running
if pgrep -f chatbot > /dev/null; then
    echo "✅ Process: Running (PID: $(pgrep -f chatbot))"
else
    echo "❌ Process: Not running"
fi

# Check HTTP endpoint
if curl -s http://localhost:8080/health > /dev/null; then
    echo "✅ HTTP: Responding"
else
    echo "❌ HTTP: Not responding"
fi

# Check HTTPS endpoint
if curl -s -k https://localhost:8443/health > /dev/null; then
    echo "✅ HTTPS: Responding"
else
    echo "❌ HTTPS: Not responding"
fi

# Check memory usage
echo ""
echo "Memory usage:"
ps aux | grep chatbot | grep -v grep
echo ""
echo "System memory:"
free -h
EOF

chmod +x health-check.sh

# Run health check
./health-check.sh

# Schedule regular health checks
(crontab -l 2>/dev/null; echo "*/15 * * * * /home/pi/rag-chatbot-server/health-check.sh >> /var/log/health-check.log") | crontab -
```

## Troubleshooting

### Common Issues and Solutions

#### 1. Certificate Permission Errors

```bash
# Error: permission denied accessing certificates
# Solution: Copy certificates to user directory
sudo cp /etc/letsencrypt/live/yourdomain.duckdns.org/* /opt/chatbot/certs/
sudo chown $USER:$USER /opt/chatbot/certs/*
```

#### 2. Port Already in Use

```bash
# Error: bind: address already in use
# Check what's using the port
sudo netstat -tlnp | grep :8080

# Kill the process using the port
sudo pkill -f "process-name"
```

#### 3. Discord Bot Not Responding

```bash
# Check bot token
echo $DISCORD_BOT_TOKEN

# Verify bot permissions in Discord server
# Check logs for connection errors
tail -f chatbot.log | grep -i discord
```

#### 4. Out of Memory (Raspberry Pi)

```bash
# Switch to ChatGPT mode to save RAM
./chatbot --chatgpt --discord

# Or add swap space
sudo fallocate -l 1G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

#### 5. Let's Encrypt DNS Challenge Issues

```bash
# Verify DNS TXT record
dig TXT _acme-challenge.yourdomain.duckdns.org

# Check DuckDNS TXT record field is populated
# Wait 5-10 minutes for DNS propagation before continuing certbot
```

#### 6. DuckDNS Not Updating

```bash
# Check DuckDNS update script
~/duckdns/duck.sh
cat ~/duckdns/duck.log

# Verify token and domain name in script
# Check internet connectivity
ping www.duckdns.org
```

#### 7. Router Port Forwarding Not Working

```bash
# Test internal connectivity first
curl http://localhost:8080/health

# Check if router forwards correctly
# Use external tool like canyouseeme.org
# Verify router settings and Pi's local IP
```

### Log Analysis

```bash
# Common log patterns to check:

# Successful startup
grep "Starting.*server" chatbot.log

# Certificate issues
grep -i "certificate\|ssl\|tls" chatbot.log

# Discord connection
grep -i "discord" chatbot.log

# LLM provider issues
grep -i "llm\|chatgpt\|ollama" chatbot.log

# API errors
grep -i "error\|failed" chatbot.log
```

### Performance Optimization for Raspberry Pi

```bash
# 1. Use ChatGPT instead of local LLM
./chatbot --chatgpt --discord

# 2. Reduce log verbosity in production
# 3. Monitor resource usage regularly
# 4. Consider using lighter models if using local LLM
export LLM_MODEL=tinyllama:latest

# 5. Enable swap if using local LLM
sudo swapon -s  # Check current swap
```

## Support and Maintenance

### Getting Help

1. Check this setup guide and troubleshooting section
2. Review application logs: `tail -f chatbot.log`
3. Test individual components (Discord, LLM, HTTPS) separately
4. Verify all environment variables are set correctly
5. Check network connectivity and port forwarding

---

**🚀 Your RAG Chatbot Server should now be fully operational with HTTPS, Discord integration, and optimized for Raspberry Pi deployment!**