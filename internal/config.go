package internal

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config is the full council configuration, read from config.json
type Config struct {
	Agents []*Agent `json:"agents"`
}

// LoadConfig loads the configuration from a JSON file.
// If the file doesn't exist, returns the default config with the 3 pre-configured agents.
func LoadConfig(path string) (*Config, error) {
	// Check if the file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// No config.json → use defaults
		return defaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config.json: %w", err)
	}

	if len(cfg.Agents) == 0 {
		return nil, fmt.Errorf("config.json has no agents defined")
	}

	return &cfg, nil
}

// defaultConfig returns the default configuration with Gemini, Kiro, and Copilot.
// Adjust the Command/Args fields to match your environment.
func defaultConfig() *Config {
	return &Config{
		Agents: []*Agent{
			{
				Name:    "Gemini",
				Command: "gemini",
				// gemini-cli accepts the prompt as an argument with the -p flag
				// Install: npm install -g @google/gemini-cli
				Args:           []string{"-p", "{prompt}"},
				TimeoutSeconds: 90,
				UseStdin:       false,
			},
			{
				Name:           "Kiro",
				Command:        "kiro-cli",
				Args:           []string{"chat", "{prompt}"},
				TimeoutSeconds: 90,
				UseStdin:       false,
			},
			{
				Name:           "Copilot",
				Command:        "copilot",
				Args:           []string{"-p", "{prompt}"},
				TimeoutSeconds: 90,
				UseStdin:       false,
			},
		},
	}
}
