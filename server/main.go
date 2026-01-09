package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	_ "modernc.org/sqlite" // O driver Pure Go (o underline é importante)
)

// 1. MODELO (Equivalente à sua Classe Java)
// As tags `json:"..."` ensinam como transformar em JSON
type Tarefa struct {
	ID        int    `json:"id"`
	Descricao string `json:"descricao"`
	Feita     bool   `json:"feita"`
}

func main() {
	// 2. CONEXÃO COM BANCO (Cria o arquivo se não existir)
	db, err := sql.Open("sqlite", "./dados.db")
	if err != nil {
		log.Fatal(err) // Se falhar aqui, o programa para (tipo um System.exit)
	}
	defer db.Close() // Fecha a conexão quando a main terminar

	// Cria a tabela se não existir
	sqlStmt := `create table if not exists tarefas (id integer not null primary key, descricao text, feita bool);`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	// 3. DEFINIÇÃO DAS ROTAS (Endpoints)
	// Em Java seria o @GetMapping
	http.HandleFunc("/tarefas", func(w http.ResponseWriter, r *http.Request) {
		// Define que a resposta é JSON
		w.Header().Set("Content-Type", "application/json")

		// Consulta no banco
		rows, err := db.Query("SELECT id, descricao, feita FROM tarefas")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var lista []Tarefa

		// Loop para ler linha a linha (ResultSet)
		for rows.Next() {
			var t Tarefa
			// Mapeia as colunas para a struct
			if err := rows.Scan(&t.ID, &t.Descricao, &t.Feita); err != nil {
				log.Printf("Erro ao ler linha: %v", err)
				continue
			}
			lista = append(lista, t)
		}

		// Verifica se houve erro durante a iteração das linhas
		if err = rows.Err(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Converte a lista para JSON e escreve na resposta
		json.NewEncoder(w).Encode(lista)
	})

	// 4. SOBE O SERVIDOR
	log.Println("Servidor rodando na porta :8080...")
	// Roda e trava aqui esperando requisições
	log.Fatal(http.ListenAndServe(":8080", nil))
}