package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"fcx-box/database"
	"fcx-box/handlers"
)

//go:embed templates css midias
var content embed.FS

func main() {
	// 1. Inicializa Banco de Dados
	db := database.Conectar()
	defer db.Close() // Fecha a conexão quando a main terminar

	// 2. Carrega Templates
	// Adiciona funções auxiliares para usar no HTML (seq para loop, add para soma)
	funcMap := template.FuncMap{
		"seq": func(start, end int) []int {
			var s []int
			for i := start; i <= end; i++ {
				s = append(s, i)
			}
			return s
		},
		"add": func(a, b int) int { return a + b },
		"get": func(list []string, idx int) string {
			if idx >= 0 && idx < len(list) {
				return list[idx]
			}
			return ""
		},
		"firstName": func(s string) string {
			if idx := strings.Index(s, " "); idx != -1 {
				return s[:idx]
			}
			return s
		},
		"truncate": func(s string, n int) string {
			runes := []rune(s)
			if len(runes) > n {
				return string(runes[:n])
			}
			return s
		},
	}
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseFS(content, "templates/*.html"))

	// 3. Inicializa Handlers
	app := &handlers.App{
		DB:   db,
		Tmpl: tmpl,
	}

	// 4. Rotas
	http.HandleFunc("/", app.PageIndex)
	http.HandleFunc("/page/home", app.PageHome)
	http.HandleFunc("/page/menu", app.PageMenu)
	http.HandleFunc("/funcionarios", app.ListarFuncionarios)
	http.HandleFunc("/produtos", app.ProdutosHandler)
	http.HandleFunc("/page/novo_funcionario", app.PageNovoFuncionario)
	http.HandleFunc("/action/salvar_funcionario", app.ActionSalvarFuncionario)
	http.HandleFunc("/page/nova_retirada", app.PageNovaRetirada)
	http.HandleFunc("/action/salvar_retirada", app.ActionSalvarRetirada)
	http.HandleFunc("/page/deletar_funcionario", app.PageDeletarFuncionario)
	http.HandleFunc("/action/deletar_funcionario", app.ActionDeletarFuncionario)
	http.HandleFunc("/page/lista_funcionarios", app.PageListarFuncionarios)
	http.HandleFunc("/page/historico_funcionario", app.PageHistoricoFuncionario)
	http.HandleFunc("/page/editar_funcionario", app.PageEditarFuncionario)
	http.HandleFunc("/action/atualizar_funcionario", app.ActionAtualizarFuncionario)
	http.HandleFunc("/api/batch_funcionarios", app.BatchAddFuncionarios)
	http.HandleFunc("/page/cria_escala", app.PageCriaEscala)
	http.HandleFunc("/action/adicionar_escala", app.ActionAdicionarEscala)
	http.HandleFunc("/action/remover_escala", app.ActionRemoverEscala)
	http.HandleFunc("/action/atualizar_status", app.ActionAtualizarStatus)
	http.HandleFunc("/action/atualizar_tarefa", app.ActionAtualizarTarefa)
	http.HandleFunc("/api/historico_anterior", app.APIGetHistoricoAnterior)
	http.HandleFunc("/page/imprimir_escala", app.PageImprimirEscala)

	// Servir arquivos estáticos (CSS)
	http.Handle("/css/", http.FileServer(http.FS(content)))
	http.Handle("/midias/", http.FileServer(http.FS(content)))

	// 5. Sobe o Servidor
	log.Println("Servidor rodando na porta :8080...")

	// Configuração de servidor mais robusta com timeouts para uso em rede
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      nil, // Usa o DefaultServeMux definido acima
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}
