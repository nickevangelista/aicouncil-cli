package internal

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Agent representa um assistente de IA acessível via linha de comando.
// Cada agente tem seu próprio binário e argumentos configuráveis.
type Agent struct {
	// Nome amigável, ex: "Gemini", "Kiro", "Copilot"
	Name string `json:"name"`

	// Binário a executar, ex: "gemini", "kiro", "gh"
	Command string `json:"command"`

	// Argumentos passados ao binário.
	// Use {prompt} como placeholder — ele será substituído pelo texto real.
	// Exemplo: ["copilot", "suggest", "-t", "general", "{prompt}"]
	Args []string `json:"args"`

	// Timeout em segundos. Padrão: 90s se deixado como 0.
	TimeoutSeconds int `json:"timeout_seconds"`

	// Se true, envia o prompt via stdin ao invés de substituir em Args.
	// Útil para CLIs que lêem da entrada padrão.
	UseStdin bool `json:"use_stdin"`
}

// Response é a resposta de um agente a um prompt.
type Response struct {
	Agent   *Agent // qual agente gerou essa resposta
	Content string // texto da resposta (vazio se houve erro)
	Err     error  // erro, se houver (nil = sucesso)
	Letter  string // "A", "B" ou "C" — atribuído pelo conselho
}

// Ask envia um prompt ao agente e retorna a resposta.
// Roda o binário como um subprocesso e captura o stdout.
func (a *Agent) Ask(prompt string) Response {
	// Define o timeout (mínimo 10s, padrão 90s)
	timeoutSec := a.TimeoutSeconds
	if timeoutSec <= 0 {
		timeoutSec = 90
	}
	timeout := time.Duration(timeoutSec) * time.Second

	// Cria um contexto com timeout — garante que o processo é encerrado se demorar demais
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Substitui o placeholder {prompt} em cada argumento
	args := make([]string, len(a.Args))
	for i, arg := range a.Args {
		args[i] = strings.ReplaceAll(arg, "{prompt}", prompt)
	}

	// Cria o comando
	cmd := exec.CommandContext(ctx, a.Command, args...)

	// Captura stdout e stderr separadamente
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Se UseStdin=true, envia o prompt pela entrada padrão
	// Isso é útil quando o {prompt} não está nos args ou quando o prompt é muito longo
	if a.UseStdin {
		cmd.Stdin = strings.NewReader(prompt)
	}

	// Executa o processo e espera terminar
	if err := cmd.Run(); err != nil {
		// Timeout é o erro mais comum — dá uma mensagem específica
		if ctx.Err() == context.DeadlineExceeded {
			return Response{
				Agent: a,
				Err:   fmt.Errorf("timeout após %v — tente aumentar timeout_seconds no config.json", timeout),
			}
		}
		// Outros erros: inclui o stderr para facilitar debug
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return Response{
			Agent: a,
			Err:   fmt.Errorf("falha ao executar %q: %s", a.Command, errMsg),
		}
	}

	content := strings.TrimSpace(stdout.String())
	if content == "" {
		return Response{
			Agent: a,
			Err:   fmt.Errorf("%s retornou resposta vazia — verifique se o comando está correto", a.Name),
		}
	}

	return Response{
		Agent:   a,
		Content: content,
	}
}
