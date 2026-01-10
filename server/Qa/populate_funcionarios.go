package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// Estrutura igual à esperada pelo servidor
type Funcionario struct {
	ID    int    `json:"id"`
	Nome  string `json:"nome"`
	Cargo string `json:"cargo"`
}

func main() {
	// URL do endpoint de importação em massa que criamos anteriormente
	url := "http://localhost:8080/api/batch_funcionarios"

	var funcionarios []Funcionario
	cargos := []string{"Operador", "Auxiliar", "Empacotador", "Apoio", "Líder"}

	fmt.Println("Gerando 100 funcionários de teste...")

	// Gera 100 funcionários fictícios
	for i := 1; i <= 100; i++ {
		f := Funcionario{
			ID:    3000 + i, // IDs a partir de 3001 para evitar conflitos com dados reais
			Nome:  fmt.Sprintf("Funcionario Teste %d", i),
			Cargo: cargos[i%len(cargos)], // Alterna os cargos ciclicamente
		}
		funcionarios = append(funcionarios, f)
	}

	// Converte para JSON
	jsonData, err := json.Marshal(funcionarios)
	if err != nil {
		log.Fatal("Erro ao gerar JSON:", err)
	}

	// Envia a requisição POST
	fmt.Println("Enviando requisição para o servidor...")
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatal("Erro na requisição (verifique se o servidor principal está rodando):", err)
	}
	defer resp.Body.Close()

	// Verifica a resposta
	fmt.Printf("Status da resposta: %s\n", resp.Status)
	fmt.Println("Concluído!")
}
