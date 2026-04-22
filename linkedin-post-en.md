I built a Go CLI that sends your prompt to multiple AIs simultaneously and makes them vote on each other's responses to elect the best answer.

Here's how it works:

1. You ask a question
2. Gemini, Kiro, and Copilot respond in parallel
3. Each AI receives the 3 anonymized responses (A, B, C) and scores them on accuracy, clarity, completeness, practicality, and conciseness
4. Highest score wins

It's called **ai-council**. The idea came from a real frustration: depending on the question, each AI has different strengths. Instead of switching between tabs, let them decide.

It's configurable — any CLI that accepts a prompt and returns text works as an agent. You can add Claude, Ollama, GPT via sgpt, whatever you want.

Open source: github.com/nickevangelista/aicouncil-cli

[terminal screenshot here]
