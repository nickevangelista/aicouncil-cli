# AI Council 🏛️

> Democracia entre IAs: envia seu prompt para múltiplos assistentes, faz votação cruzada e retorna a melhor resposta.

## Como funciona

```
Você → ai-council → [Gemini, Kiro, Copilot] (paralelo)
                 → cada IA julga as 3 respostas (paralelo)  
                 → apuração por categoria
                 → 🏆 melhor resposta
```

**Fase 1 — Coleta:** O prompt é enviado para todos os agentes ao mesmo tempo (goroutines).

**Fase 2 — Votação cruzada:** Cada agente recebe as 3 respostas anonimizadas (A, B, C) e dá notas de 1–10 em 5 categorias: precisão, clareza, completude, praticidade e concisão.

**Fase 3 — Apuração:** As notas são agregadas, o vencedor é eleito por maior pontuação total.

## Instalação

### 1. Pré-requisitos

Instale os CLIs que o conselho vai usar:

```bash
# Gemini CLI
npm install -g @google/gemini-cli
gemini  # autentica com sua conta Google

# GitHub Copilot
gh extension install github/gh-copilot
gh auth login

# Kiro (AWS)
# Siga as instruções em https://kiro.dev
```

### 2. Compilar o ai-council

```bash
git clone https://github.com/nickevangelista/aicouncil-cli
cd ai-council

# Baixa as dependências
go mod tidy

# Compila e instala globalmente como "ai-council"
make install
```

## Uso

```bash
# Pergunta básica — mostra placar + vencedor
ai-council ask "Como fazer tratamento de erros idiomático em Go?"

# Com progresso detalhado
ai-council ask "Explica o algoritmo de Dijkstra" --verbose

# Sem votação — vê todas as respostas
ai-council ask "Qual a diferença entre mutex e channel?" --no-vote

# Modo silencioso — só imprime a resposta (perfeito para pipes)
ai-council ask "Refatora esse código para ser mais legível" --quiet

# Pipe para a área de transferência
ai-council ask "Escreve um README para um projeto de API REST" --quiet | pbcopy

# Usando config customizado
ai-council ask "Pergunta" --config /caminho/meu-config.json
```

## Configuração (`config.json`)

O `config.json` fica na mesma pasta onde você roda o comando.

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
      "name": "MeuCLI",
      "command": "meu-cli",
      "args": ["perguntar"],
      "timeout_seconds": 60,
      "use_stdin": true
    }
  ]
}
```

| Campo | Tipo | Descrição |
|---|---|---|
| `name` | string | Nome amigável do agente |
| `command` | string | Binário a executar (deve estar no `$PATH`) |
| `args` | []string | Argumentos — use `{prompt}` como placeholder |
| `timeout_seconds` | int | Timeout por chamada (padrão: 90) |
| `use_stdin` | bool | Envia o prompt via stdin ao invés de args |

**Sem `config.json`:** usa Gemini + Kiro + Copilot com os comandos padrão.

## Flags

| Flag | Atalho | Descrição |
|---|---|---|
| `--config` | `-c` | Caminho para o arquivo de configuração |
| `--verbose` | `-v` | Mostra progresso detalhado de cada fase |
| `--no-vote` | `-n` | Exibe todas as respostas sem votação |
| `--quiet` | `-q` | Só imprime a resposta vencedora (bom para pipes) |

## Estrutura do projeto

```
ai-council/
├── main.go                 # Ponto de entrada, CLI com cobra
├── go.mod                  # Módulo Go
├── config.json             # Configuração dos agentes
└── internal/
    ├── config.go           # Carregamento de configuração
    ├── agent.go            # Struct Agent + execução do subprocess
    ├── council.go          # Orquestração das 3 fases
    ├── voting.go           # Prompt de julgamento, parsing JSON, apuração
    └── display.go          # Interface no terminal (cores ANSI)
```

## Adicionando novos agentes

Qualquer CLI que aceite um prompt e devolva texto funciona. Exemplos:

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

## Dependências

- `github.com/spf13/cobra` — framework de CLI (amplamente usado em Go)
- Biblioteca padrão do Go para todo o resto

## Licença

MIT
