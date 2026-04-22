package internal

import (
	"fmt"
	"sync"
)

// Council is the democratic council: orchestrates agents, collects responses, and conducts voting.
type Council struct {
	Config *Config
}

// NewCouncil creates a council from a config file.
func NewCouncil(configPath string) (*Council, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}
	return &Council{Config: cfg}, nil
}

// DeliberationResult holds everything that happened during a deliberation.
type DeliberationResult struct {
	Prompt     string      // original user question
	Responses  []*Response // all responses (including ones with errors)
	VoteResult *VoteResult // voting result (nil if --no-vote)
	Winner     *Response   // winning response
}

// Deliberate is the main method: runs the 3 phases of the council.
//
// Phase 1 — Collection: sends the prompt to all agents in parallel.
// Phase 2 — Voting: each agent judges all responses (also in parallel).
// Phase 3 — Tally: aggregates votes and determines the winner.
func (c *Council) Deliberate(prompt string, verbose bool) (*DeliberationResult, error) {
	agents := c.Config.Agents
	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents configured — check your config.json")
	}

	// ══════════════════════════════════════════
	// PHASE 1: Collect responses (in parallel)
	// ══════════════════════════════════════════
	PrintPhase(1, "Consulting agents", verbose)

	// Create the responses slice with exact size for safe concurrent access
	// (each goroutine writes to its own index → no mutex needed)
	responses := make([]*Response, len(agents))
	var wg sync.WaitGroup

	for i, agent := range agents {
		wg.Add(1)
		// Capture i and agent in the loop — required in Go < 1.22 (good practice in general)
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

	wg.Wait() // wait for all agents to finish

	// Assign letters (A, B, C...) and separate valid responses
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
		return nil, fmt.Errorf("all agents failed — check if the CLIs are installed and configured")
	}

	// With only 1 agent working there's nothing to vote on
	if len(validResponses) == 1 {
		PrintSingleWinner(validResponses[0].Agent.Name, verbose)
		return &DeliberationResult{
			Prompt:    prompt,
			Responses: responses,
			Winner:    validResponses[0],
		}, nil
	}

	// ══════════════════════════════════════════
	// PHASE 2: Cross-voting (in parallel)
	// ══════════════════════════════════════════
	PrintPhase(2, fmt.Sprintf("Cross-voting (%d judges)", len(agents)), verbose)

	// Build the judge prompt (same for all judges)
	judgePrompt := BuildJudgePrompt(prompt, validResponses)

	judgeResults := make([]*JudgeResult, len(agents))

	for i, agent := range agents {
		wg.Add(1)
		go func(idx int, a *Agent) {
			defer wg.Done()

			PrintAgentWorking(a.Name+" (judge)", verbose)

			// Reuse the Ask method — the judgePrompt is treated like any other prompt
			resp := a.Ask(judgePrompt)

			jr := &JudgeResult{Judge: a}

			if resp.Err != nil {
				jr.Err = resp.Err
				PrintAgentError(a.Name+" (judge)", resp.Err, verbose)
			} else {
				// Try to extract scores from the returned JSON
				scores, err := ParseJudgeResponse(resp.Content)
				if err != nil {
					jr.Err = fmt.Errorf("parsing failed for %s: %w", a.Name, err)
					PrintAgentError(a.Name+" (judge)", jr.Err, verbose)
				} else {
					jr.Scores = scores
					PrintAgentDone(a.Name+" (judge)", verbose)
				}
			}

			judgeResults[idx] = jr
		}(i, agent)
	}

	wg.Wait()

	// ══════════════════════════════════════════
	// PHASE 3: Vote tally
	// ══════════════════════════════════════════
	PrintPhase(3, "Tallying votes", verbose)

	voteResult := TallyVotes(validResponses, judgeResults)

	return &DeliberationResult{
		Prompt:     prompt,
		Responses:  responses,
		VoteResult: voteResult,
		Winner:     voteResult.Winner,
	}, nil
}
