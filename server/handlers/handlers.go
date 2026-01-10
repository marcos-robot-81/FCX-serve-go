package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"fcx-box/models" // <--- VERIFIQUE SEU go.mod
)

// App guarda as dependências que os handlers precisam
type App struct {
	DB   *sql.DB
	Tmpl *template.Template
}

// --- API JSON ---

func (app *App) ListarFuncionarios(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	rows, err := app.DB.Query("SELECT id, nome, cargo FROM funcionarios")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var lista []models.Funcionario
	for rows.Next() {
		var f models.Funcionario
		rows.Scan(&f.ID, &f.Nome, &f.Cargo)
		lista = append(lista, f)
	}
	json.NewEncoder(w).Encode(lista)
}

func (app *App) ProdutosHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodPost {
		var p models.Produto
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		agora := time.Now()
		p.Data = agora.Format("2006-01-02")
		p.Hora = agora.Format("15:04:05")

		res, _ := app.DB.Exec("INSERT INTO produtos (data, hora, tipo, quantidade, funcionario_id) VALUES (?, ?, ?, ?, ?)", p.Data, p.Hora, p.Tipo, p.Quantidade, p.FuncionarioID)
		id, _ := res.LastInsertId()
		p.ID = int(id)

		// Tabela Diária
		nomeTabelaDia := fmt.Sprintf("retiradas_%s", agora.Format("20060102"))
		app.DB.Exec(fmt.Sprintf(`create table if not exists %s (id integer not null primary key, data text, hora text, tipo text, quantidade integer, funcionario_id integer);`, nomeTabelaDia))
		app.DB.Exec(fmt.Sprintf("INSERT INTO %s (data, hora, tipo, quantidade, funcionario_id) VALUES (?, ?, ?, ?, ?)", nomeTabelaDia), p.Data, p.Hora, p.Tipo, p.Quantidade, p.FuncionarioID)

		json.NewEncoder(w).Encode(p)
		return
	}

	// GET
	rows, _ := app.DB.Query("SELECT id, data, hora, tipo, quantidade, funcionario_id FROM produtos")
	defer rows.Close()
	var lista []models.Produto
	for rows.Next() {
		var p models.Produto
		rows.Scan(&p.ID, &p.Data, &p.Hora, &p.Tipo, &p.Quantidade, &p.FuncionarioID)
		lista = append(lista, p)
	}
	json.NewEncoder(w).Encode(lista)
}

// --- PÁGINAS HTML ---

func (app *App) PageIndex(w http.ResponseWriter, r *http.Request) {
	app.Tmpl.ExecuteTemplate(w, "index.html", nil)
}

func (app *App) PageHome(w http.ResponseWriter, r *http.Request) {
	app.Tmpl.ExecuteTemplate(w, "home.html", nil)
}

func (app *App) PageMenu(w http.ResponseWriter, r *http.Request) {
	app.Tmpl.ExecuteTemplate(w, "menu.html", nil)
}

func (app *App) PageNovoFuncionario(w http.ResponseWriter, r *http.Request) {
	app.Tmpl.ExecuteTemplate(w, "add_funcionario.html", nil)
}

func (app *App) ActionSalvarFuncionario(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		idStr := r.FormValue("id")
		nome := r.FormValue("nome")
		cargo := r.FormValue("cargo")

		if idStr != "" {
			id, _ := strconv.Atoi(idStr)
			// Verifica se o ID já existe
			var exists int
			err := app.DB.QueryRow("SELECT 1 FROM funcionarios WHERE id = ?", id).Scan(&exists)
			if err == nil {
				// ID já existe: exibe alerta e volta para a página anterior
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				fmt.Fprintf(w, "<script>alert('O ID %d já está ocupado!'); window.history.back();</script>", id)
				return
			}
			// Insere com ID manual
			app.DB.Exec("INSERT INTO funcionarios (id, nome, cargo) VALUES (?, ?, ?)", id, nome, cargo)
		} else {
			// Insere sem ID (o banco gera automaticamente)
			app.DB.Exec("INSERT INTO funcionarios (nome, cargo) VALUES (?, ?)", nome, cargo)
		}

		http.Redirect(w, r, "/page/menu", http.StatusSeeOther)
	}
}

func (app *App) PageNovaRetirada(w http.ResponseWriter, r *http.Request) {
	rows, _ := app.DB.Query("SELECT id, nome, cargo FROM funcionarios")
	defer rows.Close()
	var lista []models.Funcionario
	for rows.Next() {
		var f models.Funcionario
		rows.Scan(&f.ID, &f.Nome, &f.Cargo)
		lista = append(lista, f)
	}
	app.Tmpl.ExecuteTemplate(w, "retirada.html", lista)
}

func (app *App) ActionSalvarRetirada(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		funcID, _ := strconv.Atoi(r.FormValue("funcionario_id"))
		tipo := r.FormValue("tipo")
		qtd, _ := strconv.Atoi(r.FormValue("quantidade"))

		agora := time.Now()
		data := agora.Format("2006-01-02")
		hora := agora.Format("15:04:05")

		if _, err := app.DB.Exec("INSERT INTO produtos (data, hora, tipo, quantidade, funcionario_id) VALUES (?, ?, ?, ?, ?)", data, hora, tipo, qtd, funcID); err != nil {
			log.Printf("Erro ao salvar produto: %v", err)
			http.Error(w, "Erro ao salvar produto", http.StatusInternalServerError)
			return
		}

		nomeTabelaDia := fmt.Sprintf("retiradas_%s", agora.Format("20060102"))
		app.DB.Exec(fmt.Sprintf(`create table if not exists %s (id integer not null primary key, data text, hora text, tipo text, quantidade integer, funcionario_id integer);`, nomeTabelaDia))
		if _, err := app.DB.Exec(fmt.Sprintf("INSERT INTO %s (data, hora, tipo, quantidade, funcionario_id) VALUES (?, ?, ?, ?, ?)", nomeTabelaDia), data, hora, tipo, qtd, funcID); err != nil {
			log.Printf("Erro ao salvar na tabela do dia: %v", err)
		}

		http.Redirect(w, r, "/page/menu", http.StatusSeeOther)
	}
}

func (app *App) PageDeletarFuncionario(w http.ResponseWriter, r *http.Request) {
	qID := r.URL.Query().Get("q_id")
	qNome := r.URL.Query().Get("q_nome")
	qCargo := r.URL.Query().Get("q_cargo")

	// Paginação
	pageStr := r.URL.Query().Get("page")
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	limit := 10
	offset := (page - 1) * limit

	sqlQuery := "SELECT id, nome, cargo FROM funcionarios WHERE 1=1"
	var args []interface{}

	if qID != "" {
		sqlQuery += " AND CAST(id AS TEXT) LIKE ?"
		args = append(args, "%"+qID+"%")
	}
	if qNome != "" {
		sqlQuery += " AND nome LIKE ?"
		args = append(args, "%"+qNome+"%")
	}
	if qCargo != "" {
		sqlQuery += " AND cargo LIKE ?"
		args = append(args, "%"+qCargo+"%")
	}

	sqlQuery += " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := app.DB.Query(sqlQuery, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var lista []models.Funcionario
	for rows.Next() {
		var f models.Funcionario
		rows.Scan(&f.ID, &f.Nome, &f.Cargo)
		lista = append(lista, f)
	}

	data := ListaFuncData{
		Funcionarios: lista,
		Page:         page,
		PrevPage:     page - 1,
		NextPage:     page + 1,
		QueryID:      qID,
		QueryNome:    qNome,
		QueryCargo:   qCargo,
	}
	app.Tmpl.ExecuteTemplate(w, "delete_funcionario.html", data)
}

func (app *App) ActionDeletarFuncionario(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		id, err := strconv.Atoi(r.FormValue("id"))
		if err != nil {
			http.Error(w, "ID inválido", http.StatusBadRequest)
			return
		}

		// Inicia transação para garantir integridade (apaga tudo ou nada)
		tx, err := app.DB.Begin()
		if err != nil {
			http.Error(w, "Erro interno no banco", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback() // Garante que a transação seja cancelada se houver erro ou panic

		// 1. Primeiro deleta o histórico (produtos) desse funcionário
		if _, err := tx.Exec("DELETE FROM produtos WHERE funcionario_id = ?", id); err != nil {
			log.Printf("Erro ao deletar produtos do funcionário %d: %v", id, err)
			http.Error(w, "Erro ao limpar histórico do funcionário", http.StatusInternalServerError)
			return
		}

		// 2. Depois deleta o funcionário
		if _, err := tx.Exec("DELETE FROM funcionarios WHERE id = ?", id); err != nil {
			log.Printf("Erro ao deletar funcionário %d: %v", id, err)
			http.Error(w, "Erro ao deletar funcionário", http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			log.Printf("Erro ao confirmar exclusão: %v", err)
			http.Error(w, "Erro ao finalizar exclusão", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/page/deletar_funcionario", http.StatusSeeOther)
	}
}

// --- EDIÇÃO DE FUNCIONÁRIO ---

func (app *App) PageEditarFuncionario(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, _ := strconv.Atoi(idStr)

	var f models.Funcionario
	err := app.DB.QueryRow("SELECT id, nome, cargo FROM funcionarios WHERE id = ?", id).Scan(&f.ID, &f.Nome, &f.Cargo)
	if err != nil {
		http.Error(w, "Funcionário não encontrado", http.StatusNotFound)
		return
	}

	app.Tmpl.ExecuteTemplate(w, "editar_funcionario.html", f)
}

func (app *App) ActionAtualizarFuncionario(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		oldID, _ := strconv.Atoi(r.FormValue("old_id"))
		newID, _ := strconv.Atoi(r.FormValue("id"))
		nome := r.FormValue("nome")
		cargo := r.FormValue("cargo")

		// Se o ID mudou, verifica se o novo ID já existe
		if oldID != newID {
			var exists int
			err := app.DB.QueryRow("SELECT 1 FROM funcionarios WHERE id = ?", newID).Scan(&exists)
			if err == nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				fmt.Fprintf(w, "<script>alert('O ID %d já está ocupado!'); window.history.back();</script>", newID)
				return
			}
			// Atualiza produtos para o novo ID (Requisito: "todos os produtos com o id do funcionario tambem devem ser mudados")
			app.DB.Exec("UPDATE produtos SET funcionario_id = ? WHERE funcionario_id = ?", newID, oldID)
		}

		app.DB.Exec("UPDATE funcionarios SET id = ?, nome = ?, cargo = ? WHERE id = ?", newID, nome, cargo, oldID)
		http.Redirect(w, r, "/page/lista_funcionarios", http.StatusSeeOther)
	}
}

// --- IMPORTAÇÃO EM MASSA (BATCH) ---

func (app *App) BatchAddFuncionarios(w http.ResponseWriter, r *http.Request) {
	// Configuração de CORS para permitir requisições externas (ex: testes via fetch/browser)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		log.Printf("BatchAddFuncionarios: Método recebido %s (esperado POST). Verifique se houve redirecionamento.", r.Method)
		http.Error(w, "Método não permitido. Use POST.", http.StatusMethodNotAllowed)
		return
	}

	var funcionarios []models.Funcionario
	if err := json.NewDecoder(r.Body).Decode(&funcionarios); err != nil {
		http.Error(w, "Erro ao ler JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Inicia uma transação para inserir tudo de uma vez (muito mais rápido para 200+ registros)
	tx, err := app.DB.Begin()
	if err != nil {
		http.Error(w, "Erro interno ao iniciar transação", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	for _, f := range funcionarios {
		if f.ID != 0 {
			// Se o JSON trouxer ID, tenta inserir com aquele ID
			if _, err := tx.Exec("INSERT INTO funcionarios (id, nome, cargo) VALUES (?, ?, ?)", f.ID, f.Nome, f.Cargo); err != nil {
				http.Error(w, fmt.Sprintf("Erro ao inserir ID %d (%s): %v", f.ID, f.Nome, err), http.StatusInternalServerError)
				return
			}
		} else {
			// Se não trouxer ID, deixa o banco gerar
			if _, err := tx.Exec("INSERT INTO funcionarios (nome, cargo) VALUES (?, ?)", f.Nome, f.Cargo); err != nil {
				http.Error(w, fmt.Sprintf("Erro ao inserir %s: %v", f.Nome, err), http.StatusInternalServerError)
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Erro ao salvar dados no banco", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Sucesso! %d funcionários importados.", len(funcionarios))
}

// --- API AUXILIAR PARA O FRONTEND ---

func (app *App) APIGetHistoricoAnterior(w http.ResponseWriter, r *http.Request) {
	dataStr := r.URL.Query().Get("data")
	idStr := r.URL.Query().Get("id")
	id, _ := strconv.Atoi(idStr)

	// Calcula data anterior
	t, err := time.Parse("2006-01-02", dataStr)
	if err != nil {
		http.Error(w, "Data inválida", http.StatusBadRequest)
		return
	}
	prevDate := t.AddDate(0, 0, -1).Format("2006-01-02")

	var jsonContent string
	err = app.DB.QueryRow("SELECT json_content FROM escalas WHERE data = ?", prevDate).Scan(&jsonContent)

	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		// Sem escala no dia anterior
		fmt.Fprintf(w, `{"data": "%s", "mensagem": "Sem registro."}`, prevDate)
		return
	}

	var dia models.DiaDeTrabalho
	json.Unmarshal([]byte(jsonContent), &dia)

	// Coleta atribuições únicas do dia anterior
	atribuicoes := make(map[string]bool)
	for h := 1; h <= 24; h++ {
		q := getQuadro(&dia, h)
		for _, p := range q.Pessoas {
			if p.FuncionarioID == id {
				vals := []string{p.Caixa1, p.Caixa2, p.Caixa3, p.Tarefa1, p.Tarefa2, p.Tarefa3}
				for _, v := range vals {
					if v != "" {
						atribuicoes[v] = true
					}
				}
			}
		}
	}

	var resumo string
	for k := range atribuicoes {
		if resumo != "" {
			resumo += ", "
		}
		resumo += k
	}

	fmt.Fprintf(w, `{"data": "%s", "mensagem": "%s"}`, prevDate, resumo)
}

// --- ESCALA (DIA DE TRABALHO) ---

type EscalaViewData struct {
	Data          string
	Funcionarios  []models.Funcionario
	Tabelas       map[string][]models.Quadro
	SelectedCargo string
}

func (app *App) PageCriaEscala(w http.ResponseWriter, r *http.Request) {
	cargoFilter := r.URL.Query().Get("cargo")
	data := r.URL.Query().Get("data")
	if data == "" {
		data = time.Now().Format("2006-01-02")
	}

	// 1. Carrega lista de funcionários para o formulário
	var rows *sql.Rows
	var err error

	if cargoFilter != "" {
		// Se tem filtro, busca só daquele cargo
		rows, err = app.DB.Query("SELECT id, nome, cargo FROM funcionarios WHERE cargo = ? ORDER BY nome", cargoFilter)
	} else {
		// Se não tem, busca todos
		rows, err = app.DB.Query("SELECT id, nome, cargo FROM funcionarios ORDER BY nome")
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var funcs []models.Funcionario
	for rows.Next() {
		var f models.Funcionario
		rows.Scan(&f.ID, &f.Nome, &f.Cargo)
		funcs = append(funcs, f)
	}
	rows.Close()

	// 2. Carrega a escala do banco (JSON)
	var jsonContent string
	var dia models.DiaDeTrabalho
	err = app.DB.QueryRow("SELECT json_content FROM escalas WHERE data = ?", data).Scan(&jsonContent)
	if err == nil {
		json.Unmarshal([]byte(jsonContent), &dia)
	}
	dia.Data = data

	// --- LÓGICA DO HISTÓRICO ANTERIOR (PLACEHOLDER) ---
	tCurrent, _ := time.Parse("2006-01-02", data)
	prevDate := tCurrent.AddDate(0, 0, -1).Format("2006-01-02")
	var prevJson string
	var prevDia models.DiaDeTrabalho
	historicoMap := make(map[int][]string)

	// Tenta carregar o dia anterior
	if err := app.DB.QueryRow("SELECT json_content FROM escalas WHERE data = ?", prevDate).Scan(&prevJson); err == nil {
		json.Unmarshal([]byte(prevJson), &prevDia)

		// Mapa temporário para garantir unicidade das tarefas: ID -> Tarefa -> bool
		tempMap := make(map[int][]string)
		seen := make(map[int]map[string]bool)

		for h := 1; h <= 24; h++ {
			q := getQuadro(&prevDia, h)
			for _, p := range q.Pessoas {
				if seen[p.FuncionarioID] == nil {
					seen[p.FuncionarioID] = make(map[string]bool)
				}

				vals := []string{p.Caixa1, p.Caixa2, p.Caixa3, p.Tarefa1, p.Tarefa2, p.Tarefa3}
				for _, v := range vals {
					if v != "" && !seen[p.FuncionarioID][v] {
						seen[p.FuncionarioID][v] = true
						tempMap[p.FuncionarioID] = append(tempMap[p.FuncionarioID], v)
					}
				}
			}
		}
		for id, tasks := range tempMap {
			historicoMap[id] = tasks
		}
	}

	// 3. Organiza os dados para as 5 tabelas (Cargo -> 24 Horas)
	cargos := []string{"Operador", "Auxiliar", "Empacotador", "Apoio", "Líder"}
	tabelas := make(map[string][]models.Quadro)

	for _, cargo := range cargos {
		var horas []models.Quadro
		for h := 1; h <= 24; h++ {
			quadroOriginal := getQuadro(&dia, h)
			// Filtra apenas pessoas deste cargo neste horário
			var pessoasDoCargo []models.EscalaPessoa
			for _, p := range quadroOriginal.Pessoas {
				if p.Cargo == cargo {
					// Injeta o histórico no campo temporário
					if hist, ok := historicoMap[p.FuncionarioID]; ok {
						p.HistoricoAnterior = hist
					}
					pessoasDoCargo = append(pessoasDoCargo, p)
				}
			}
			horas = append(horas, models.Quadro{Pessoas: pessoasDoCargo})
		}
		tabelas[cargo] = horas
	}

	viewData := EscalaViewData{
		Data:          data,
		Funcionarios:  funcs,
		Tabelas:       tabelas,
		SelectedCargo: cargoFilter,
	}

	app.Tmpl.ExecuteTemplate(w, "cria_escala.html", viewData)
}

func (app *App) PageImprimirEscala(w http.ResponseWriter, r *http.Request) {
	data := r.URL.Query().Get("data")
	cargosSelected := r.URL.Query()["cargos"] // Pega múltiplos valores do checkbox

	if data == "" {
		http.Error(w, "Data é obrigatória", http.StatusBadRequest)
		return
	}

	// Carrega a escala
	var jsonContent string
	var dia models.DiaDeTrabalho
	err := app.DB.QueryRow("SELECT json_content FROM escalas WHERE data = ?", data).Scan(&jsonContent)
	if err == nil {
		json.Unmarshal([]byte(jsonContent), &dia)
	}
	dia.Data = data

	// Se nenhum cargo foi selecionado, seleciona todos por padrão
	if len(cargosSelected) == 0 {
		cargosSelected = []string{"Operador", "Auxiliar", "Empacotador", "Apoio", "Líder"}
	}

	tabelas := make(map[string][]models.Quadro)
	var cargosOrdenados []string
	ordemPadrao := []string{"Operador", "Auxiliar", "Empacotador", "Apoio", "Líder"}

	selMap := make(map[string]bool)
	for _, c := range cargosSelected {
		selMap[c] = true
	}

	for _, cargo := range ordemPadrao {
		if selMap[cargo] {
			cargosOrdenados = append(cargosOrdenados, cargo)
			var horas []models.Quadro
			for h := 1; h <= 24; h++ {
				quadroOriginal := getQuadro(&dia, h)
				var pessoasDoCargo []models.EscalaPessoa
				for _, p := range quadroOriginal.Pessoas {
					if p.Cargo == cargo {
						pessoasDoCargo = append(pessoasDoCargo, p)
					}
				}
				horas = append(horas, models.Quadro{Pessoas: pessoasDoCargo})
			}
			tabelas[cargo] = horas
		}
	}

	// Reutiliza a estrutura de dados passando também a ordem dos cargos
	dataView := struct {
		Data        string
		Tabelas     map[string][]models.Quadro
		CargosOrdem []string
	}{
		Data:        data,
		Tabelas:     tabelas,
		CargosOrdem: cargosOrdenados,
	}

	app.Tmpl.ExecuteTemplate(w, "imprimir_escala.html", dataView)
}

func (app *App) ActionAdicionarEscala(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		data := r.FormValue("data")
		funcID, _ := strconv.Atoi(r.FormValue("funcionario_id"))
		cargo := r.FormValue("cargo")

		// Busca nome do funcionário
		var nomeFunc string
		app.DB.QueryRow("SELECT nome FROM funcionarios WHERE id = ?", funcID).Scan(&nomeFunc)

		// Carrega ou cria escala
		var jsonContent string
		var dia models.DiaDeTrabalho
		err := app.DB.QueryRow("SELECT json_content FROM escalas WHERE data = ?", data).Scan(&jsonContent)
		if err == nil {
			json.Unmarshal([]byte(jsonContent), &dia)
		}
		dia.Data = data

		// Adiciona nas horas selecionadas
		for h := 1; h <= 24; h++ {
			if r.FormValue(fmt.Sprintf("hora_%d", h)) == "on" {
				quadro := getQuadro(&dia, h)
				novaPessoa := models.EscalaPessoa{
					FuncionarioID:     funcID,
					NomeDoFuncionario: nomeFunc,
					Cargo:             cargo,
					Data:              data,
				}
				quadro.Pessoas = append(quadro.Pessoas, novaPessoa)
				setQuadro(&dia, h, quadro)
			}
		}

		// Salva no banco
		novoJson, _ := json.Marshal(dia)
		app.DB.Exec("INSERT OR REPLACE INTO escalas (data, json_content) VALUES (?, ?)", data, string(novoJson))

		http.Redirect(w, r, "/page/cria_escala?data="+data, http.StatusSeeOther)
	}
}

func (app *App) ActionRemoverEscala(w http.ResponseWriter, r *http.Request) {
	data := r.URL.Query().Get("data")
	hora, _ := strconv.Atoi(r.URL.Query().Get("hora"))
	funcID, _ := strconv.Atoi(r.URL.Query().Get("id"))

	var jsonContent string
	var dia models.DiaDeTrabalho
	if err := app.DB.QueryRow("SELECT json_content FROM escalas WHERE data = ?", data).Scan(&jsonContent); err != nil {
		http.Redirect(w, r, "/page/cria_escala?data="+data, http.StatusSeeOther)
		return
	}
	json.Unmarshal([]byte(jsonContent), &dia)

	quadro := getQuadro(&dia, hora)
	var novasPessoas []models.EscalaPessoa
	for _, p := range quadro.Pessoas {
		if p.FuncionarioID != funcID {
			novasPessoas = append(novasPessoas, p)
		}
	}
	quadro.Pessoas = novasPessoas
	setQuadro(&dia, hora, quadro)

	novoJson, _ := json.Marshal(dia)
	app.DB.Exec("UPDATE escalas SET json_content = ? WHERE data = ?", string(novoJson), data)

	http.Redirect(w, r, "/page/cria_escala?data="+data, http.StatusSeeOther)
}

func (app *App) ActionAtualizarStatus(w http.ResponseWriter, r *http.Request) {
	data := r.URL.Query().Get("data")
	hora, _ := strconv.Atoi(r.URL.Query().Get("hora"))
	funcID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	status := r.URL.Query().Get("status")
	cargoFilter := r.URL.Query().Get("cargo") // Para manter o filtro após reload

	var jsonContent string
	var dia models.DiaDeTrabalho
	if err := app.DB.QueryRow("SELECT json_content FROM escalas WHERE data = ?", data).Scan(&jsonContent); err != nil {
		http.Redirect(w, r, "/page/cria_escala?data="+data, http.StatusSeeOther)
		return
	}
	json.Unmarshal([]byte(jsonContent), &dia)

	quadro := getQuadro(&dia, hora)
	for i, p := range quadro.Pessoas {
		if p.FuncionarioID == funcID {
			quadro.Pessoas[i].Status = status
			break
		}
	}
	setQuadro(&dia, hora, quadro)

	novoJson, _ := json.Marshal(dia)
	app.DB.Exec("UPDATE escalas SET json_content = ? WHERE data = ?", string(novoJson), data)

	redirectURL := fmt.Sprintf("/page/cria_escala?data=%s&cargo=%s", data, cargoFilter)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (app *App) ActionAtualizarTarefa(w http.ResponseWriter, r *http.Request) {
	data := r.URL.Query().Get("data")
	hora, _ := strconv.Atoi(r.URL.Query().Get("hora"))
	funcID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	coluna := r.URL.Query().Get("coluna") // "1", "2", "3"
	valor := r.URL.Query().Get("valor")
	cargoFilter := r.URL.Query().Get("cargo")

	var jsonContent string
	var dia models.DiaDeTrabalho
	if err := app.DB.QueryRow("SELECT json_content FROM escalas WHERE data = ?", data).Scan(&jsonContent); err != nil {
		http.Redirect(w, r, "/page/cria_escala?data="+data, http.StatusSeeOther)
		return
	}
	json.Unmarshal([]byte(jsonContent), &dia)

	quadro := getQuadro(&dia, hora)
	for i, p := range quadro.Pessoas {
		if p.FuncionarioID == funcID {
			if p.Cargo == "Operador" {
				switch coluna {
				case "1":
					quadro.Pessoas[i].Caixa1 = valor
				case "2":
					quadro.Pessoas[i].Caixa2 = valor
				case "3":
					quadro.Pessoas[i].Caixa3 = valor
				}
			} else {
				switch coluna {
				case "1":
					quadro.Pessoas[i].Tarefa1 = valor
				case "2":
					quadro.Pessoas[i].Tarefa2 = valor
				case "3":
					quadro.Pessoas[i].Tarefa3 = valor
				}
			}
			break
		}
	}
	setQuadro(&dia, hora, quadro)

	novoJson, _ := json.Marshal(dia)
	app.DB.Exec("UPDATE escalas SET json_content = ? WHERE data = ?", string(novoJson), data)

	redirectURL := fmt.Sprintf("/page/cria_escala?data=%s&cargo=%s", data, cargoFilter)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// Helpers para mapear Hora1...Hora24 dinamicamente
func getQuadro(d *models.DiaDeTrabalho, h int) models.Quadro {
	switch h {
	case 1:
		return d.Hora1
	case 2:
		return d.Hora2
	case 3:
		return d.Hora3
	case 4:
		return d.Hora4
	case 5:
		return d.Hora5
	case 6:
		return d.Hora6
	case 7:
		return d.Hora7
	case 8:
		return d.Hora8
	case 9:
		return d.Hora9
	case 10:
		return d.Hora10
	case 11:
		return d.Hora11
	case 12:
		return d.Hora12
	case 13:
		return d.Hora13
	case 14:
		return d.Hora14
	case 15:
		return d.Hora15
	case 16:
		return d.Hora16
	case 17:
		return d.Hora17
	case 18:
		return d.Hora18
	case 19:
		return d.Hora19
	case 20:
		return d.Hora20
	case 21:
		return d.Hora21
	case 22:
		return d.Hora22
	case 23:
		return d.Hora23
	case 24:
		return d.Hora24
	}
	return models.Quadro{}
}

func setQuadro(d *models.DiaDeTrabalho, h int, q models.Quadro) {
	switch h {
	case 1:
		d.Hora1 = q
	case 2:
		d.Hora2 = q
	case 3:
		d.Hora3 = q
	case 4:
		d.Hora4 = q
	case 5:
		d.Hora5 = q
	case 6:
		d.Hora6 = q
	case 7:
		d.Hora7 = q
	case 8:
		d.Hora8 = q
	case 9:
		d.Hora9 = q
	case 10:
		d.Hora10 = q
	case 11:
		d.Hora11 = q
	case 12:
		d.Hora12 = q
	case 13:
		d.Hora13 = q
	case 14:
		d.Hora14 = q
	case 15:
		d.Hora15 = q
	case 16:
		d.Hora16 = q
	case 17:
		d.Hora17 = q
	case 18:
		d.Hora18 = q
	case 19:
		d.Hora19 = q
	case 20:
		d.Hora20 = q
	case 21:
		d.Hora21 = q
	case 22:
		d.Hora22 = q
	case 23:
		d.Hora23 = q
	case 24:
		d.Hora24 = q
	}
}

// --- NOVAS FUNÇÕES DE LISTAGEM E HISTÓRICO ---

type ListaFuncData struct {
	Funcionarios []models.Funcionario
	Page         int
	PrevPage     int
	NextPage     int
	QueryID      string
	QueryNome    string
	QueryCargo   string
}

func (app *App) PageListarFuncionarios(w http.ResponseWriter, r *http.Request) {
	qID := r.URL.Query().Get("q_id")
	qNome := r.URL.Query().Get("q_nome")
	qCargo := r.URL.Query().Get("q_cargo")

	// Paginação
	pageStr := r.URL.Query().Get("page")
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	limit := 10
	offset := (page - 1) * limit

	sqlQuery := "SELECT id, nome, cargo FROM funcionarios WHERE 1=1"
	var args []interface{}

	if qID != "" {
		sqlQuery += " AND CAST(id AS TEXT) LIKE ?"
		args = append(args, "%"+qID+"%")
	}
	if qNome != "" {
		sqlQuery += " AND nome LIKE ?"
		args = append(args, "%"+qNome+"%")
	}
	if qCargo != "" {
		sqlQuery += " AND cargo LIKE ?"
		args = append(args, "%"+qCargo+"%")
	}

	sqlQuery += " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := app.DB.Query(sqlQuery, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var lista []models.Funcionario
	for rows.Next() {
		var f models.Funcionario
		rows.Scan(&f.ID, &f.Nome, &f.Cargo)
		lista = append(lista, f)
	}

	data := ListaFuncData{
		Funcionarios: lista,
		Page:         page,
		PrevPage:     page - 1,
		NextPage:     page + 1,
		QueryID:      qID,
		QueryNome:    qNome,
		QueryCargo:   qCargo,
	}
	app.Tmpl.ExecuteTemplate(w, "lista_funcionarios.html", data)
}

type HistoricoData struct {
	ID           int
	Nome         string
	DataInicio   string
	DataFim      string
	Produtos     []models.Produto
	Soma         int
	Media        float64
	MediaPorHora float64
}

func (app *App) PageHistoricoFuncionario(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, _ := strconv.Atoi(idStr)
	dataInicio := r.URL.Query().Get("data_inicio")
	dataFim := r.URL.Query().Get("data_fim")

	if dataInicio == "" {
		dataInicio = time.Now().Format("2006-01-02")
	}
	if dataFim == "" {
		dataFim = time.Now().Format("2006-01-02")
	}

	// Busca nome do funcionário para exibir no cabeçalho
	var nome string
	app.DB.QueryRow("SELECT nome FROM funcionarios WHERE id = ?", id).Scan(&nome)

	rows, err := app.DB.Query("SELECT id, data, hora, tipo, quantidade, funcionario_id FROM produtos WHERE funcionario_id = ? AND data >= ? AND data <= ? ORDER BY data DESC, hora DESC", id, dataInicio, dataFim)
	if err != nil {
		http.Error(w, "Erro ao buscar histórico: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var lista []models.Produto
	var soma int
	uniqueHours := make(map[string]bool)

	for rows.Next() {
		var p models.Produto
		rows.Scan(&p.ID, &p.Data, &p.Hora, &p.Tipo, &p.Quantidade, &p.FuncionarioID)
		lista = append(lista, p)
		soma += p.Quantidade

		// Identifica horas únicas (Data + Hora(HH))
		if len(p.Hora) >= 2 {
			uniqueHours[p.Data+p.Hora[:2]] = true
		}
	}

	var media float64
	if len(lista) > 0 {
		media = float64(soma) / float64(len(lista))
	}
	var mediaPorHora float64
	if len(uniqueHours) > 0 {
		mediaPorHora = float64(soma) / float64(len(uniqueHours))
	}

	data := HistoricoData{
		ID:           id,
		Nome:         nome,
		DataInicio:   dataInicio,
		DataFim:      dataFim,
		Produtos:     lista,
		Soma:         soma,
		Media:        media,
		MediaPorHora: mediaPorHora,
	}
	app.Tmpl.ExecuteTemplate(w, "historico_funcionario.html", data)
}
