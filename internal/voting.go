package internal

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Categories are the evaluation dimensions used in voting.
// Each judge (agent) gives a score from 1 to 10 in each category for each response.
var Categories = []string{
	"accuracy",      // factual correctness
	"clarity",       // ease of understanding
	"completeness",  // coverage and breadth of the topic
	"practicality",  // usefulness and real-world applicability
	"conciseness",   // appropriate brevity — not too short, not too verbose
}

// CategoryLabels are the labels shown in the UI (aligned with Categories)
var CategoryLabels = map[string]string{
	"accuracy":     "Accuracy",
	"clarity":      "Clarity",
	"completeness": "Completeness",
	"practicality": "Practicality",
	"conciseness":  "Conciseness",
}

// Score maps each category to a score (1–10)
type Score map[string]float64

// JudgeResult is the result of an agent acting as a judge.
// Each agent evaluates ALL responses, including its own.
type JudgeResult struct {
	Judge  *Agent            // which agent is judging
	Scores map[string]Score  // "A" -> {accuracy: 8, clarity: 9, ...}
	Err    error             // execution or parsing error
}

// VoteResult is the final voting result after aggregating all judges.
type VoteResult struct {
	Responses    []*Response              // original responses (valid ones only)
	JudgeResults []*JudgeResult           // each judge's votes
	Totals       map[string]float64       // total points per letter (A, B, C)
	AvgScores    map[string]Score         // average per category per letter
	Winner       *Response               // winning response
}

// BuildJudgePrompt creates the prompt sent to each agent to judge the responses.
// Responses are anonymized (A, B, C) to avoid self-reference bias.
func BuildJudgePrompt(userPrompt string, responses []*Response) string {
	var sb strings.Builder

	sb.WriteString("You are an impartial expert judge. Your task is to evaluate 3 different responses to the same question.\n\n")
	sb.WriteString(fmt.Sprintf("ORIGINAL QUESTION:\n%s\n\n", userPrompt))
	sb.WriteString(strings.Repeat("─", 60) + "\n\n")

	for _, r := range responses {
		sb.WriteString(fmt.Sprintf("RESPONSE %s:\n", r.Letter))
		sb.WriteString(r.Content)
		sb.WriteString("\n\n" + strings.Repeat("─", 60) + "\n\n")
	}

	sb.WriteString("EVALUATION INSTRUCTIONS:\n")
	sb.WriteString("Give a score from 1 to 10 for each response in the categories below:\n")
	sb.WriteString("  - accuracy:     factual correctness of the information presented\n")
	sb.WriteString("  - clarity:      ease of understanding and structure\n")
	sb.WriteString("  - completeness: coverage — does it address all relevant aspects?\n")
	sb.WriteString("  - practicality: is it actionable and useful in practice?\n")
	sb.WriteString("  - conciseness:  appropriate brevity — not too short, not too verbose\n\n")

	sb.WriteString("REQUIRED RESPONSE FORMAT:\n")
	sb.WriteString("Return ONLY a valid JSON object — no text before, no text after,\n")
	sb.WriteString("no markdown code blocks (no ```), no explanations. Just the JSON:\n\n")

	// Example of expected output to guide the model
	sb.WriteString(`{"A":{"accuracy":8,"clarity":9,"completeness":7,"practicality":8,"conciseness":7},`)
	sb.WriteString(`"B":{"accuracy":7,"clarity":8,"completeness":9,"practicality":7,"conciseness":8},`)
	sb.WriteString(`"C":{"accuracy":9,"clarity":7,"completeness":8,"practicality":9,"conciseness":6}}`)
	sb.WriteString("\n")

	return sb.String()
}

// ParseJudgeResponse extracts scores from a judge response.
// It's error-tolerant: tries to strip markdown, find embedded JSON in text, etc.
func ParseJudgeResponse(content string) (map[string]Score, error) {
	content = strings.TrimSpace(content)

	// Remove markdown code blocks that some models insist on using
	// Ex: ```json { ... } ```
	content = regexp.MustCompile("(?s)```[a-z]*\n?(.*?)\n?```").ReplaceAllString(content, "$1")
	content = strings.TrimSpace(content)

	// Strategy 1: try to parse the entire content as JSON
	var scores map[string]Score
	if err := json.Unmarshal([]byte(content), &scores); err == nil && isValidScoreMap(scores) {
		return scores, nil
	}

	// Strategy 2: look for any valid JSON object in the text
	// Useful when the model adds text before/after the JSON
	jsonRegex := regexp.MustCompile(`\{[\s\S]+\}`)
	matches := jsonRegex.FindAllString(content, -1)

	// Sort by descending length — the largest match is likely the most complete
	sort.Slice(matches, func(i, j int) bool {
		return len(matches[i]) > len(matches[j])
	})

	for _, match := range matches {
		if err := json.Unmarshal([]byte(match), &scores); err == nil && isValidScoreMap(scores) {
			return scores, nil
		}
	}

	return nil, fmt.Errorf("could not extract valid scores from judge response")
}

// isValidScoreMap checks if the score map has the expected structure:
// must have at least one key "A", "B", or "C" with categories inside.
func isValidScoreMap(scores map[string]Score) bool {
	if len(scores) == 0 {
		return false
	}
	for _, letter := range []string{"A", "B", "C"} {
		if s, ok := scores[letter]; ok && len(s) > 0 {
			return true
		}
	}
	return false
}

// TallyVotes aggregates votes from all judges and determines the winner.
// Each judge has equal weight — the average of all judges' scores is used.
func TallyVotes(responses []*Response, judgeResults []*JudgeResult) *VoteResult {
	result := &VoteResult{
		Responses:    responses,
		JudgeResults: judgeResults,
		Totals:       make(map[string]float64),
		AvgScores:    make(map[string]Score),
	}

	// Initialize accumulators to calculate averages later
	rawSums := make(map[string]Score)          // letter -> {category -> sum}
	counts := make(map[string]map[string]int)  // letter -> {category -> n}

	for _, r := range responses {
		rawSums[r.Letter] = make(Score)
		counts[r.Letter] = make(map[string]int)
		result.AvgScores[r.Letter] = make(Score)
	}

	// Aggregate scores from each judge
	validJudges := 0
	for _, jr := range judgeResults {
		if jr == nil || jr.Err != nil || jr.Scores == nil {
			continue // skip judges that failed
		}
		validJudges++

		for letter, categoryScores := range jr.Scores {
			if _, exists := rawSums[letter]; !exists {
				continue // unknown letter — skip
			}
			for cat, score := range categoryScores {
				// Ensure score is within valid range (1–10)
				if score < 1 {
					score = 1
				}
				if score > 10 {
					score = 10
				}
				rawSums[letter][cat] += score
				counts[letter][cat]++
			}
		}
	}

	if validJudges == 0 {
		// No judge worked — return without a winner
		return result
	}

	// Calculate averages and final totals
	for _, r := range responses {
		var total float64
		for _, cat := range Categories {
			n := counts[r.Letter][cat]
			if n == 0 {
				continue
			}
			avg := rawSums[r.Letter][cat] / float64(n)
			result.AvgScores[r.Letter][cat] = avg
			total += avg
		}
		result.Totals[r.Letter] = total
	}

	// Determine the winner: highest total points
	var maxTotal float64
	var winnerLetter string
	for letter, total := range result.Totals {
		if total > maxTotal {
			maxTotal = total
			winnerLetter = letter
		}
	}

	// Map the letter back to the original Response
	for _, r := range responses {
		if r.Letter == winnerLetter {
			result.Winner = r
			break
		}
	}

	return result
}
