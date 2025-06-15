package services

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"chatbot/models"
)

// DiscordService handles Discord bot interactions
type DiscordService struct {
	session       *discordgo.Session
	chatbot       *Chatbot
	commandPrefix string
	enabled       bool
	startTime     time.Time
}

// NewDiscordService creates a new Discord service instance
func NewDiscordService(chatbot *Chatbot) *DiscordService {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	commandPrefix := os.Getenv("DISCORD_COMMAND_PREFIX")
	
	if commandPrefix == "" {
		commandPrefix = "!chat "
	}

	service := &DiscordService{
		chatbot:       chatbot,
		commandPrefix: commandPrefix,
		enabled:       false,
		startTime:     time.Now(),
	}

	if token == "" {
		log.Printf("Discord bot disabled: DISCORD_BOT_TOKEN environment variable not set")
		log.Printf("To enable Discord bot:")
		log.Printf("1. Create bot at https://discord.com/developers/applications")
		log.Printf("2. Set DISCORD_BOT_TOKEN environment variable")
		log.Printf("3. Restart the application")
		return service
	}

	// Create Discord session
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Printf("Error creating Discord session: %v", err)
		return service
	}

	service.session = session      

	session.AddHandler(func(s *discordgo.Session, event *discordgo.Ready) {
                log.Printf("✅ Bot is online as: %s", event.User.Username)
                log.Printf("📊 Connected to %d servers", len(event.Guilds))
        })

	// Add message handler
	session.AddHandler(service.messageCreate)
	
	// Set intents
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent

	service.enabled = true
	log.Printf("Discord service initialized with prefix: %s", commandPrefix)
	
	return service
}

// Start begins the Discord bot service
func (d *DiscordService) Start() error {
	if !d.enabled {
		return fmt.Errorf("Discord service not enabled (missing bot token)")
	}

	// Open websocket connection
	err := d.session.Open()
	if err != nil {
		return fmt.Errorf("error opening Discord connection: %w", err)
	}

	log.Printf("Discord bot started successfully! Use '%s<message>' in Discord", d.commandPrefix)
	return nil
}

// Stop closes the Discord bot connection
func (d *DiscordService) Stop() error {
	if d.session != nil {
		return d.session.Close()
	}
	return nil
}

// messageCreate handles incoming Discord messages
func (d *DiscordService) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from bots
	if m.Author.Bot {
		return
	}

	// Check if message starts with command prefix
	if !strings.HasPrefix(m.Content, d.commandPrefix) {
		return
	}

	// Extract message after command prefix
	chatMessage := strings.TrimSpace(m.Content[len(d.commandPrefix):])
	if chatMessage == "" {
		d.sendMessage(s, m.ChannelID, fmt.Sprintf("Please provide a message after `%s`", strings.TrimSpace(d.commandPrefix)))
		return
	}

	// Show typing indicator
	s.ChannelTyping(m.ChannelID)

	// Create session ID based on user and channel
	sessionID := fmt.Sprintf("discord_%s_%s", m.Author.ID, m.ChannelID)

	// Process message through chatbot service
	response := d.chatbot.ProcessMessage(chatMessage, sessionID, []models.ChatMessage{})

	// Send response back to Discord
	d.sendMessage(s, m.ChannelID, response.Message)

	// Log the interaction
	log.Printf("Discord chat: User %s (%s) in channel %s: %s", 
		m.Author.Username, m.Author.ID, m.ChannelID, chatMessage)
}

// sendMessage sends a message to Discord, handling length limits
func (d *DiscordService) sendMessage(s *discordgo.Session, channelID, message string) {
	// Discord has a 2000 character limit
	if len(message) <= 2000 {
		_, err := s.ChannelMessageSend(channelID, message)
		if err != nil {
			log.Printf("Error sending Discord message: %v", err)
		}
		return
	}

	// Split long messages into chunks
	chunks := d.splitMessage(message, 1900) // Leave some margin
	for i, chunk := range chunks {
		if i > 0 {
			chunk = fmt.Sprintf("...continued:\n%s", chunk)
		}
		if i < len(chunks)-1 {
			chunk = chunk + "\n..."
		}
		
		_, err := s.ChannelMessageSend(channelID, chunk)
		if err != nil {
			log.Printf("Error sending Discord message chunk: %v", err)
		}
		
		// Small delay between messages to avoid rate limiting
		time.Sleep(200 * time.Millisecond)
	}
}

// splitMessage splits a message into chunks respecting word boundaries
func (d *DiscordService) splitMessage(message string, maxLength int) []string {
	if len(message) <= maxLength {
		return []string{message}
	}

	var chunks []string
	for len(message) > maxLength {
		// Try to split at a word boundary
		splitIndex := maxLength
		if spaceIndex := strings.LastIndex(message[:maxLength], " "); spaceIndex > maxLength/2 {
			splitIndex = spaceIndex
		}

		chunks = append(chunks, message[:splitIndex])
		message = message[splitIndex:]
		
		// Remove leading space if we split at word boundary
		if strings.HasPrefix(message, " ") {
			message = message[1:]
		}
	}
	
	if len(message) > 0 {
		chunks = append(chunks, message)
	}
	
	return chunks
}

// IsEnabled returns whether the Discord service is enabled
func (d *DiscordService) IsEnabled() bool {
	return d.enabled
}

// GetStatus returns the current status of the Discord service
func (d *DiscordService) GetStatus() map[string]interface{} {
	status := map[string]interface{}{
		"enabled":        d.enabled,
		"command_prefix": d.commandPrefix,
		"uptime":         time.Since(d.startTime).String(),
	}

	if d.enabled && d.session != nil {
		status["status"] = "connected"
		status["user"] = map[string]interface{}{
			"id":       d.session.State.User.ID,
			"username": d.session.State.User.Username,
		}
		status["guilds"] = len(d.session.State.Guilds)
	} else if d.enabled {
		status["status"] = "initialized_not_started"
	} else {
		status["status"] = "disabled"
		status["note"] = "Set DISCORD_BOT_TOKEN environment variable to enable"
	}

	return status
}
