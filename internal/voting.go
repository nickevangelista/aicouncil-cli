package internal

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Categories são as dimensões de avaliação usadas na votação.
// Cada juiz (agente) dá uma nota de 1 a 10 em cada categoria para cada resposta.
var Categories = []string{
	"precisao",    // acurácia e correção factual
	"clareza",     // facilidade de entendimento
	"completude",  // abrangência e cobertura do tema
	"praticidade", // utilidade e aplicabilidade prática
	"concisao",    // brevidade e objetividade adequadas
}

// CategoryLabels são os labels exibidos na interface (alinhados com Categories)
var CategoryLabels = map[string]string{
	"precisao":    "Precisão",
	"clareza":     "Clareza",
	"completude":  "Completude",
	"praticidade": "Praticidade",
	"concisao":    "Concisão",
}

// Score mapeia cada categoria para uma pontuação (1–10)
type Score map[string]float64

// JudgeResult é o resultado do julgamento de um agente como juiz.
// Cada agente avalia TODAS as respostas, incluindo a sua própria.
type JudgeResult struct {
	Judge  *Agent            // qual agente está julgando
	Scores map[string]Score  // "A" -> {precisao: 8, clareza: 9, ...}
	Err    error             // erro de execução ou de parsing
}

// VoteResult é o resultado final da votação, após agregar todos os juízes.
type VoteResult struct {
	Responses    []*Response              // respostas originais (só as válidas)
	JudgeResults []*JudgeResult           // votos de cada juiz
	Totals       map[string]float64       // total de pontos por letra (A, B, C)
	AvgScores    map[string]Score         // média por categoria por letra
	Winner       *Response               // resposta vencedora
}

// BuildJudgePrompt cria o prompt enviado a cada agente para que ele julgue as respostas.
// As respostas são anonimizadas (A, B, C) para evitar viés de autorreferência.
func BuildJudgePrompt(userPrompt string, responses []*Response) string {
	var sb strings.Builder

	sb.WriteString("Você é um juiz imparcial e especialista. Sua tarefa é avaliar 3 respostas diferentes para a mesma pergunta.\n\n")
	sb.WriteString(fmt.Sprintf("PERGUNTA ORIGINAL:\n%s\n\n", userPrompt))
	sb.WriteString(strings.Repeat("─", 60) + "\n\n")

	for _, r := range responses {
		sb.WriteString(fmt.Sprintf("RESPOSTA %s:\n", r.Letter))
		sb.WriteString(r.Content)
		sb.WriteString("\n\n" + strings.Repeat("─", 60) + "\n\n")
	}

	sb.WriteString("INSTRUÇÕES DE AVALIAÇÃO:\n")
	sb.WriteString("Dê uma nota de 1 a 10 para cada resposta nas categorias abaixo:\n")
	sb.WriteString("  - precisao:    acurácia e correção dos fatos apresentados\n")
	sb.WriteString("  - clareza:     facilidade de entendimento e estrutura\n")
	sb.WriteString("  - completude:  abrangência — cobre todos os aspectos relevantes?\n")
	sb.WriteString("  - praticidade: é acionável e útil na prática?\n")
	sb.WriteString("  - concisao:    brevidade adequada — nem curta demais, nem verbosa\n\n")

	sb.WriteString("FORMATO OBRIGATÓRIO DA RESPOSTA:\n")
	sb.WriteString("Retorne APENAS um objeto JSON válido — sem texto antes, sem texto depois,\n")
	sb.WriteString("sem blocos de código markdown (sem ```), sem explicações. Só o JSON:\n\n")

	// Exemplo de saída esperada para guiar o modelo
	sb.WriteString(`{"A":{"precisao":8,"clareza":9,"completude":7,"praticidade":8,"concisao":7},`)
	sb.WriteString(`"B":{"precisao":7,"clareza":8,"completude":9,"praticidade":7,"concisao":8},`)
	sb.WriteString(`"C":{"precisao":9,"clareza":7,"completude":8,"praticidade":9,"concisao":6}}`)
	sb.WriteString("\n")

	return sb.String()
}

// ParseJudgeResponse extrai as pontuações de uma resposta de julgamento.
// É tolerante a erros: tenta limpar markdown, encontrar JSON embutido em texto, etc.
func ParseJudgeResponse(content string) (map[string]Score, error) {
	content = strings.TrimSpace(content)

	// Remove blocos de código markdown que alguns modelos insistem em usar
	// Ex: ```json { ... } ```
	content = regexp.MustCompile("(?s)```[a-z]*\n?(.*?)\n?```").ReplaceAllString(content, "$1")
	content = strings.TrimSpace(content)

	// Estratégia 1: tenta parsear o conteúdo inteiro como JSON
	var scores map[string]Score
	if err := json.Unmarshal([]byte(content), &scores); err == nil && isValidScoreMap(scores) {
		return scores, nil
	}

	// Estratégia 2: procura qualquer objeto JSON válido no texto
	// Útil quando o modelo adiciona texto antes/depois do JSON
	jsonRegex := regexp.MustCompile(`\{[\s\S]+\}`)
	matches := jsonRegex.FindAllString(content, -1)

	// Ordena por tamanho decrescente — o maior match é provavelmente o mais completo
	sort.Slice(matches, func(i, j int) bool {
		return len(matches[i]) > len(matches[j])
	})

	for _, match := range matches {
		if err := json.Unmarshal([]byte(match), &scores); err == nil && isValidScoreMap(scores) {
			return scores, nil
		}
	}

	return nil, fmt.Errorf("não foi possível extrair pontuações válidas da resposta do juiz")
}

// isValidScoreMap verifica se o mapa de scores tem a estrutura esperada:
// deve ter pelo menos uma chave "A", "B" ou "C" com categorias dentro.
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

// TallyVotes agrega os votos de todos os juízes e determina o vencedor.
// Cada juiz tem peso igual — a média das notas de todos os juízes é usada.
func TallyVotes(responses []*Response, judgeResults []*JudgeResult) *VoteResult {
	result := &VoteResult{
		Responses:    responses,
		JudgeResults: judgeResults,
		Totals:       make(map[string]float64),
		AvgScores:    make(map[string]Score),
	}

	// Inicializa acumuladores para calcular médias depois
	rawSums := make(map[string]Score)   // letra -> {categoria -> soma}
	counts := make(map[string]map[string]int) // letra -> {categoria -> n}

	for _, r := range responses {
		rawSums[r.Letter] = make(Score)
		counts[r.Letter] = make(map[string]int)
		result.AvgScores[r.Letter] = make(Score)
	}

	// Agrega as notas de cada juiz
	validJudges := 0
	for _, jr := range judgeResults {
		if jr == nil || jr.Err != nil || jr.Scores == nil {
			continue // ignora juízes que falharam
		}
		validJudges++

		for letter, categoryScores := range jr.Scores {
			if _, exists := rawSums[letter]; !exists {
				continue // letra desconhecida — ignora
			}
			for cat, score := range categoryScores {
				// Garante que a nota está no intervalo válido (1–10)
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
		// Nenhum juiz funcionou — retorna sem vencedor
		return result
	}

	// Calcula médias e totais finais
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

	// Determina o vencedor: maior total de pontos
	var maxTotal float64
	var winnerLetter string
	for letter, total := range result.Totals {
		if total > maxTotal {
			maxTotal = total
			winnerLetter = letter
		}
	}

	// Mapeia a letra de volta para a Response original
	for _, r := range responses {
		if r.Letter == winnerLetter {
			result.Winner = r
			break
		}
	}

	return result
}
