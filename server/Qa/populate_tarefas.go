package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Estrutura para ler a resposta da API de funcionários
type Funcionario struct {
	ID    int    `json:"id"`
	Nome  string `json:"nome"`
	Cargo string `json:"cargo"`
}

func main() {
	baseURL := "http://localhost:8080"

	// Cliente HTTP configurado para não seguir redirecionamentos (ganho de performance)
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// 1. Obter lista de funcionários existentes
	fmt.Println("Obtendo lista de funcionários...")
	resp, err := http.Get(baseURL + "/funcionarios")
	if err != nil {
		log.Fatal("Erro ao conectar com o servidor:", err)
	}
	defer resp.Body.Close()

	var funcionarios []Funcionario
	if err := json.NewDecoder(resp.Body).Decode(&funcionarios); err != nil {
		log.Fatal("Erro ao decodificar JSON de funcionários:", err)
	}

	fmt.Printf("Encontrados %d funcionários.\n", len(funcionarios))

	// Data para a escala (hoje)
	dataEscala := time.Now().Format("2006-01-02")
	fmt.Printf("Gerando escala e tarefas para o dia: %s\n", dataEscala)

	// Seed aleatório
	rand.Seed(time.Now().UnixNano())

	// Agrupa funcionários por cargo para facilitar a distribuição
	pool := make(map[string][]Funcionario)
	for _, f := range funcionarios {
		pool[f.Cargo] = append(pool[f.Cargo], f)
	}

	countEscalados := 0

	// Função auxiliar para atribuir turno a um grupo de funcionários
	assignShift := func(cargo string, count int, startHour int) {
		available := pool[cargo]
		if len(available) == 0 {
			return
		}

		take := count
		if len(available) < take {
			take = len(available)
		}

		selected := available[:take]
		pool[cargo] = available[take:] // Remove do pool para não escalar duas vezes

		for _, f := range selected {
			// Define horário de trabalho (8 horas de duração + 1h intervalo)
			endHour := startHour + 8

			// Prepara formulário para adicionar à escala
			form := url.Values{}
			form.Set("data", dataEscala)
			form.Set("funcionario_id", strconv.Itoa(f.ID))
			form.Set("cargo", f.Cargo)

			var horasTrabalhadas []int

			for h := startHour; h <= endHour; h++ {
				// Simula intervalo (4 horas após o início)
				if h == startHour+4 {
					continue
				}
				// Limite de 24h
				if h > 24 {
					break
				}
				form.Set(fmt.Sprintf("hora_%d", h), "on")
				horasTrabalhadas = append(horasTrabalhadas, h)
			}

			// Envia POST para adicionar à escala
			respEscala, err := client.PostForm(baseURL+"/action/adicionar_escala", form)
			if err != nil {
				log.Printf("Erro ao adicionar escala para %s: %v", f.Nome, err)
				continue
			}
			respEscala.Body.Close()
			countEscalados++

			// Adiciona tarefas aleatórias para cada hora trabalhada
			for _, h := range horasTrabalhadas {
				// Preenche coluna 1, 2 ou 3 aleatoriamente
				colunas := []string{"1", "2", "3"}

				for _, col := range colunas {
					// 60% de chance de ter tarefa nessa coluna
					if rand.Float32() > 0.4 {
						valor := gerarTarefaAleatoria(f.Cargo)

						params := url.Values{}
						params.Set("data", dataEscala)
						params.Set("hora", strconv.Itoa(h))
						params.Set("id", strconv.Itoa(f.ID))
						params.Set("coluna", col)
						params.Set("valor", valor)
						params.Set("cargo", f.Cargo)

						// GET request para atualizar tarefa
						getUrl := fmt.Sprintf("%s/action/atualizar_tarefa?%s", baseURL, params.Encode())
						respTarefa, err := client.Get(getUrl)
						if err != nil {
							log.Printf("Erro ao atualizar tarefa: %v", err)
						} else {
							respTarefa.Body.Close()
						}
					}
				}
			}
			fmt.Printf(".") // Feedback visual
		}
	}

	// --- APLICAÇÃO DAS REGRAS ---

	// Líderes: 6, 8, 13, 14 (1 em cada)
	for _, h := range []int{6, 8, 13, 14} {
		assignShift("Líder", 1, h)
	}

	// Auxiliar: 7(1), 8(1), 11(2), 14(2)
	assignShift("Auxiliar", 1, 7)
	assignShift("Auxiliar", 1, 8)
	assignShift("Auxiliar", 2, 11)
	assignShift("Auxiliar", 2, 14)

	// Apoio: 6:30(6), 8, 9, 13, 14, 16 (1 em cada)
	for _, h := range []int{6, 8, 9, 13, 14, 16} {
		assignShift("Apoio", 1, h)
	}

	// Operadores: 10 em cada horário: 7, 8, 9, 11, 14, 15, 16
	for _, h := range []int{7, 8, 9, 11, 14, 15, 16} {
		assignShift("Operador", 10, h)
	}

	// Empacotadores: 3 a 5 em cada horário: 7, 8, 9, 11, 14, 15, 16
	for _, h := range []int{7, 8, 9, 11, 14, 15, 16} {
		assignShift("Empacotador", rand.Intn(3)+3, h)
	}

	fmt.Printf("\nConcluído! %d funcionários escalados com tarefas.\n", countEscalados)
}

func gerarTarefaAleatoria(cargo string) string {
	if cargo == "Operador" {
		// Retorna número do caixa (1 a 15)
		return strconv.Itoa(rand.Intn(15) + 1)
	}

	tarefas := []string{
		"Reposição", "Limpeza", "Organização", "Frente de Loja",
		"Estoque", "Validade", "Preço", "Atendimento", "Inventário",
	}
	return tarefas[rand.Intn(len(tarefas))]
}
