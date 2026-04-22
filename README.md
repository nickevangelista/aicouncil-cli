# AI Council 🏛️

> AI democracy: sends your prompt to multiple assistants, runs cross-voting, and returns the best response.

<img width="1440" height="795" alt="image" src="https://github.com/user-attachments/assets/9585c7cf-0f15-484f-ac8f-aa844ae8d2f7" />



## How it works

```
You → ai-council → [Gemini, Kiro, Copilot] (parallel)
                 → each AI judges all 3 responses (parallel)
                 → tally by category
                 → 🏆 best response
```

**Phase 1 — Collection:** The prompt is sent to all agents simultaneously (goroutines).

**Phase 2 — Cross-voting:** Each agent receives the 3 anonymized responses (A, B, C) and scores them 1–10 across 5 categories: accuracy, clarity, completeness, practicality, and conciseness.

**Phase 3 — Tally:** Scores are aggregated, the winner is elected by highest total.

## Installation

### 1. Prerequisites

Install the CLIs the council will use:

```bash
# Go (required to compile)
# https://go.dev/dl/

# Node.js (required to install CLIs via npm)
# https://nodejs.org/

# Gemini CLI
npm install -g @google/gemini-cli
gemini  # authenticate with your Google account

# GitHub Copilot CLI
# macOS/Linux/WSL:
curl -fsSL https://gh.io/copilot-install | bash
# Windows:
winget install GitHub.Copilot
copilot login  # authenticate with your GitHub account

# Kiro CLI
# Follow the instructions at https://kiro.dev
```

> **WSL:** if you don't have `make`, run `sudo apt install build-essential`
> **macOS:** `make` comes pre-installed with Xcode Command Line Tools (`xcode-select --install`)

### 2. Build ai-council

```bash
git clone https://github.com/nickevangelista/aicouncil-cli
cd aicouncil-cli

# Download dependencies
go mod tidy

# Compile and install globally as "ai-council"
make install
```

## Usage

```bash
# Basic question — shows scoreboard + winner
ai-council ask "How to do idiomatic error handling in Go?"

# With detailed progress
ai-council ask "Explain Dijkstra's algorithm" --verbose

# No voting — see all responses
ai-council ask "What's the difference between mutex and channel?" --no-vote

# Quiet mode — only prints the response (great for pipes)
ai-council ask "Refactor this code to be more readable" --quiet

# Pipe to clipboard
ai-council ask "Write a README for a REST API project" --quiet | pbcopy   # macOS
ai-council ask "Write a README for a REST API project" --quiet | clip.exe  # WSL

# Using a custom config
ai-council ask "Question" --config /path/to/my-config.json
```

## Configuration (`config.json`)

The `config.json` lives in the same folder where you run the command.

```json
{
  "agents": [
    {
      "name": "Gemini",
      "command": "gemini",
      "args": ["-p", "{prompt}"],
      "timeout_seconds": 90,
      "use_stdin": false
    },
    {
      "name": "MyCLI",
      "command": "my-cli",
      "args": ["ask"],
      "timeout_seconds": 60,
      "use_stdin": true
    }
  ]
}
```

| Field | Type | Description |
|---|---|---|
| `name` | string | Friendly agent name |
| `command` | string | Binary to execute (must be in `$PATH`) |
| `args` | []string | Arguments — use `{prompt}` as a placeholder |
| `timeout_seconds` | int | Timeout per call (default: 90) |
| `use_stdin` | bool | Send the prompt via stdin instead of args |

**Without `config.json`:** uses Gemini + Kiro + Copilot with default commands.

## Flags

| Flag | Shorthand | Description |
|---|---|---|
| `--config` | `-c` | Path to the config file |
| `--verbose` | `-v` | Show detailed progress for each phase |
| `--no-vote` | `-n` | Display all responses without voting |
| `--quiet` | `-q` | Only prints the winning response (good for pipes) |

## Project structure

```
ai-council/
├── main.go                 # Entry point, CLI with cobra
├── go.mod                  # Go module
├── config.json             # Agent configuration
└── internal/
    ├── config.go           # Configuration loading
    ├── agent.go            # Agent struct + subprocess execution
    ├── council.go          # Orchestration of the 3 phases
    ├── voting.go           # Judge prompt, JSON parsing, vote tally
    └── display.go          # Terminal UI (ANSI colors)
```

## Adding new agents

Any CLI that accepts a prompt and returns text works. Examples:

```json
{
  "name": "Claude",
  "command": "claude",
  "args": ["-p", "{prompt}"],
  "timeout_seconds": 120
}
```

```json
{
  "name": "Ollama Llama",
  "command": "ollama",
  "args": ["run", "llama3", "{prompt}"],
  "timeout_seconds": 180
}
```

```json
{
  "name": "GPT-4",
  "command": "sgpt",
  "args": ["{prompt}"],
  "timeout_seconds": 60
}
```

## Dependencies

- `github.com/spf13/cobra` — CLI framework (widely used in Go)
- Go standard library for everything else

## License

MIT
