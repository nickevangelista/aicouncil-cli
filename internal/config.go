package internal

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config é a configuração completa do conselho, lida do config.json
type Config struct {
	Agents []*Agent `json:"agents"`
}

// LoadConfig carrega a configuração de um arquivo JSON.
// Se o arquivo não existir, retorna a configuração padrão com os 3 agentes pré-configurados.
func LoadConfig(path string) (*Config, error) {
	// Verifica se o arquivo existe
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Sem config.json → usa os valores padrão
		return defaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config.json inválido: %w", err)
	}

	if len(cfg.Agents) == 0 {
		return nil, fmt.Errorf("config.json não tem nenhum agente definido")
	}

	return &cfg, nil
}

// defaultConfig retorna a configuração padrão com Gemini, Kiro e Copilot.
// Ajuste os campos Command/Args conforme o seu ambiente.
func defaultConfig() *Config {
	return &Config{
		Agents: []*Agent{
			{
				Name:    "Gemini",
				Command: "gemini",
				// gemini-cli aceita o prompt como argumento com a flag -p
				// Instale: npm install -g @google/gemini-cli
				Args:           []string{"-p", "{prompt}"},
				TimeoutSeconds: 90,
				UseStdin:       false,
			},
			{
				Name:    "Kiro",
				Command: "kiro",
				// Kiro CLI da AWS — ajuste conforme a versão instalada
				// Instale: https://kiro.dev
				Args:           []string{"ask", "{prompt}"},
				TimeoutSeconds: 90,
				UseStdin:       false,
			},
			{
				Name:    "Copilot",
				Command: "gh",
				// GitHub Copilot via gh CLI
				// Instale: gh extension install github/gh-copilot
				Args:           []string{"copilot", "suggest", "-t", "general", "{prompt}"},
				TimeoutSeconds: 90,
				UseStdin:       false,
			},
		},
	}
}
