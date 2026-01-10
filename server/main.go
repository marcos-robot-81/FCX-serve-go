package main

import (
	"html/template"
	"log"
	"net/http"

	"fcx-box/database"
	"fcx-box/handlers"
)

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
	}
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.html"))

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

	// Servir arquivos estáticos (CSS)
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("./css"))))

	// 5. Sobe o Servidor
	log.Println("Servidor rodando na porta :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
