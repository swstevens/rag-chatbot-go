package utils

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

// LoadEnv loads environment variables from a .env file
func LoadEnv(filename string) error {
	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// .env file doesn't exist, which is okay
		log.Printf("No %s file found, using system environment variables only", filename)
		return nil
	}

	// Open the .env file
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error opening %s file: %w", filename, err)
	}
	defer file.Close()

	log.Printf("Loading environment variables from %s", filename)

	// Read line by line
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Parse KEY=VALUE format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			log.Printf("Warning: Invalid format in %s line %d: %s", filename, lineNumber, line)
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		// Remove quotes if present
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
		   (strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		}
		
		// Only set if not already set in system environment
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
			log.Printf("Set %s from .env file", key)
		} else {
			log.Printf("Environment variable %s already set, keeping existing value", key)
		}
	}
	
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading %s file: %w", filename, err)
	}
	
	return nil
}

// LoadEnvWithFallback tries multiple .env file locations
func LoadEnvWithFallback() error {
	// Try multiple locations in order of preference
	locations := []string{
		".env",           // Current directory
		".env.local",     // Local override
		"config/.env",    // Config directory
	}
	
	for _, location := range locations {
		if err := LoadEnv(location); err != nil {
			log.Printf("Could not load %s: %v", location, err)
			continue
		}
		return nil
	}
	
	// No .env file found in any location - that's okay
	log.Printf("No .env files found in standard locations, using system environment only")
	return nil
}
