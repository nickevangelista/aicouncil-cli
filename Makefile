# Makefile para o ai-council
# Uso: make build | make install | make run PROMPT="sua pergunta"

# Nome do binário
BINARY := ai-council

# Diretório de instalação (deve estar no $PATH)
INSTALL_DIR := $(HOME)/.local/bin

.PHONY: build install run clean tidy

## build: compila para o sistema operacional atual
build:
	go build -o $(BINARY) .
	@echo "✓ Compilado: ./$(BINARY)"

## install: compila e instala em ~/.local/bin (ou /usr/local/bin com sudo)
install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "✓ Instalado em $(INSTALL_DIR)/$(BINARY)"
	@echo "  Certifique-se que $(INSTALL_DIR) está no seu \$$PATH"

## run: roda diretamente com go run (sem compilar)
## Uso: make run PROMPT="Como funciona o GC do Go?"
run:
	go run . ask "$(PROMPT)"

## run-verbose: roda com saída detalhada
run-verbose:
	go run . ask "$(PROMPT)" --verbose

## tidy: baixa e organiza as dependências
tidy:
	go mod tidy

## clean: remove o binário compilado
clean:
	rm -f $(BINARY)

## help: lista os comandos disponíveis
help:
	@grep -E '^## ' Makefile | sed 's/## //' | column -t -s ':'
