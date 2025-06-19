package models

import "time"

// DiscordMessage represents a Discord message for context
type DiscordMessage struct {
	ID        string    `json:"id"`
	ChannelID string    `json:"channel_id"`
	Content   string    `json:"content"`
	Author    string    `json:"author"`
	Timestamp time.Time `json:"timestamp"`
	IsBot     bool      `json:"is_bot"`
}

// DiscordConfig represents Discord service configuration
type DiscordConfig struct {
	Token         string `json:"token"`
	CommandPrefix string `json:"command_prefix"`
	Enabled       bool   `json:"enabled"`
}

// DiscordStatus represents Discord service status
type DiscordStatus struct {
	BaseResponse
	Enabled       bool           `json:"enabled"`
	CommandPrefix string         `json:"command_prefix"`
	Uptime        string         `json:"uptime"`
	User          *DiscordUser   `json:"user,omitempty"`
	Guilds        int            `json:"guilds,omitempty"`
	Channels      map[string]int `json:"channels,omitempty"`
}

// DiscordUser represents a Discord user
type DiscordUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Bot      bool   `json:"bot"`
}

// DiscordChannelInfo represents Discord channel information
type DiscordChannelInfo struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	MessageCount int       `json:"message_count"`
	LastActivity time.Time `json:"last_activity"`
	RAGEnabled   bool      `json:"rag_enabled"`
}
