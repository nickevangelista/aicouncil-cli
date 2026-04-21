package internal

import (
	"fmt"
	"sync"
)

// Council é o conselho democrático: orquestra agentes, coleta respostas e conduz a votação.
type Council struct {
	Config *Config
}

// NewCouncil cria um conselho a partir de um arquivo de configuração.
func NewCouncil(configPath string) (*Council, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}
	return &Council{Config: cfg}, nil
}

// DeliberationResult contém tudo que aconteceu durante uma deliberação.
type DeliberationResult struct {
	Prompt     string         // pergunta original do usuário
	Responses  []*Response    // todas as respostas (incluindo as com erro)
	VoteResult *VoteResult    // resultado da votação (nil se --no-vote)
	Winner     *Response      // resposta vencedora
}

// Deliberate é o método principal: executa as 3 fases do conselho.
//
// Fase 1 — Coleta: envia o prompt para todos os agentes em paralelo.
// Fase 2 — Votação: cada agente julga todas as respostas (também em paralelo).
// Fase 3 — Apuração: agrega os votos e determina o vencedor.
func (c *Council) Deliberate(prompt string, verbose bool) (*DeliberationResult, error) {
	agents := c.Config.Agents
	if len(agents) == 0 {
		return nil, fmt.Errorf("nenhum agente configurado — verifique o config.json")
	}

	// ══════════════════════════════════════════
	// FASE 1: Coleta de respostas (em paralelo)
	// ══════════════════════════════════════════
	PrintPhase(1, "Consultando agentes", verbose)

	// Cria o slice de respostas com o tamanho exato para acesso concorrente seguro
	// (cada goroutine escreve em seu próprio índice → não precisa de mutex)
	responses := make([]*Response, len(agents))
	var wg sync.WaitGroup

	for i, agent := range agents {
		wg.Add(1)
		// Captura i e agent no loop — necessário em Go < 1.22 (boa prática geral)
		go func(idx int, a *Agent) {
			defer wg.Done()

			PrintAgentWorking(a.Name, verbose)
			resp := a.Ask(prompt)
			responses[idx] = &resp

			if resp.Err != nil {
				PrintAgentError(a.Name, resp.Err, verbose)
			} else {
				PrintAgentDone(a.Name, verbose)
			}
		}(i, agent)
	}

	wg.Wait() // aguarda todos os agentes terminarem

	// Atribui letras (A, B, C...) e separa as respostas válidas
	letters := []string{"A", "B", "C", "D", "E"}
	var validResponses []*Response

	for i, r := range responses {
		if r == nil {
			continue
		}
		if i < len(letters) {
			r.Letter = letters[i]
		}
		if r.Err == nil && r.Content != "" {
			validResponses = append(validResponses, r)
		}
	}

	if len(validResponses) == 0 {
		return nil, fmt.Errorf("todos os agentes falharam — verifique se os CLIs estão instalados e configurados")
	}

	// Com apenas 1 agente funcionando não há o que votar
	if len(validResponses) == 1 {
		PrintSingleWinner(validResponses[0].Agent.Name, verbose)
		return &DeliberationResult{
			Prompt:    prompt,
			Responses: responses,
			Winner:    validResponses[0],
		}, nil
	}

	// ══════════════════════════════════════════
	// FASE 2: Votação cruzada (em paralelo)
	// ══════════════════════════════════════════
	PrintPhase(2, fmt.Sprintf("Votação cruzada (%d juízes)", len(agents)), verbose)

	// Cria o prompt de julgamento (igual para todos os juízes)
	judgePrompt := BuildJudgePrompt(prompt, validResponses)

	judgeResults := make([]*JudgeResult, len(agents))

	for i, agent := range agents {
		wg.Add(1)
		go func(idx int, a *Agent) {
			defer wg.Done()

			PrintAgentWorking(a.Name+" (juiz)", verbose)

			// Reutiliza o método Ask — o judgePrompt é tratado como qualquer outro prompt
			resp := a.Ask(judgePrompt)

			jr := &JudgeResult{Judge: a}

			if resp.Err != nil {
				jr.Err = resp.Err
				PrintAgentError(a.Name+" (juiz)", resp.Err, verbose)
			} else {
				// Tenta extrair as pontuações do JSON retornado
				scores, err := ParseJudgeResponse(resp.Content)
				if err != nil {
					jr.Err = fmt.Errorf("parsing falhou para %s: %w", a.Name, err)
					PrintAgentError(a.Name+" (juiz)", jr.Err, verbose)
				} else {
					jr.Scores = scores
					PrintAgentDone(a.Name+" (juiz)", verbose)
				}
			}

			judgeResults[idx] = jr
		}(i, agent)
	}

	wg.Wait()

	// ══════════════════════════════════════════
	// FASE 3: Apuração dos votos
	// ══════════════════════════════════════════
	PrintPhase(3, "Apuração", verbose)

	voteResult := TallyVotes(validResponses, judgeResults)

	return &DeliberationResult{
		Prompt:     prompt,
		Responses:  responses,
		VoteResult: voteResult,
		Winner:     voteResult.Winner,
	}, nil
}
