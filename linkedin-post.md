Fiz um CLI em Go que manda seu prompt pra múltiplas IAs ao mesmo tempo e faz elas votarem entre si pra eleger a melhor resposta.

Funciona assim:

1. Você manda uma pergunta
2. Gemini, Kiro e Copilot respondem em paralelo
3. Cada IA recebe as 3 respostas anonimizadas (A, B, C) e dá notas em precisão, clareza, completude, praticidade e concisão
4. A com maior pontuação ganha

O projeto se chama **ai-council**. A ideia veio de uma frustração real: dependendo da pergunta, cada IA tem um ponto forte diferente. Em vez de ficar alternando entre abas, deixa elas decidirem.

É configurável — qualquer CLI que aceite um prompt e devolva texto funciona como agente. Dá pra adicionar Claude, Ollama, GPT via sgpt, o que quiser.

Código aberto: github.com/nickevangelista/aicouncil-cli

[print do terminal aqui]
