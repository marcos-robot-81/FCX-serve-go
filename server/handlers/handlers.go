package handlers

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"fcx-box/models" // <--- VERIFIQUE SEU go.mod
)

// App guarda as dependências que os handlers precisam
type App struct {
	DB   *sql.DB
	Tmpl *template.Template
}

// Helper para buscar cargos do banco
func (app *App) GetCargos() []string {
	rows, err := app.DB.Query("SELECT nome FROM cargos ORDER BY id")
	if err != nil {
		return []string{}
	}
	defer rows.Close()
	var cargos []string
	for rows.Next() {
		var c string
		rows.Scan(&c)
		cargos = append(cargos, c)
	}
	return cargos
}

// Helper para buscar tipos de produtos
func (app *App) GetTiposProdutos() []string {
	rows, err := app.DB.Query("SELECT nome FROM tipos_produtos ORDER BY nome")
	if err != nil {
		return []string{"Paninho"}
	}
	defer rows.Close()
	var tipos []string
	for rows.Next() {
		var t string
		rows.Scan(&t)
		tipos = append(tipos, t)
	}
	return tipos
}

// Helper para gerenciar senha
func (app *App) GetSenha() string {
	var senha string
	err := app.DB.QueryRow("SELECT valor FROM configuracoes WHERE chave = 'senha_admin'").Scan(&senha)
	if err != nil {
		return "8080" // Fallback padrão
	}
	return senha
}

func (app *App) SetSenha(nova string) {
	// Remove a senha anterior para evitar duplicatas caso a tabela não tenha PK definida corretamente
	app.DB.Exec("DELETE FROM configuracoes WHERE chave = 'senha_admin'")
	app.DB.Exec("INSERT INTO configuracoes (chave, valor) VALUES ('senha_admin', ?)", nova)
}

// --- API JSON ---

func (app *App) ListarFuncionarios(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	rows, err := app.DB.Query("SELECT id, nome, cargo FROM funcionarios WHERE ativo = 1")
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
	// Verifica autenticação (Protegido por senha)
	if !app.isAuthenticated(w, r) {
		http.Redirect(w, r, "/page/configurar", http.StatusSeeOther)
		return
	}

	cargos := app.GetCargos()
	app.Tmpl.ExecuteTemplate(w, "add_funcionario.html", cargos)
}

func (app *App) ActionSalvarFuncionario(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if !app.isAuthenticated(w, r) {
			http.Redirect(w, r, "/page/configurar", http.StatusSeeOther)
			return
		}

		idStr := strings.TrimSpace(r.FormValue("id"))
		nome := r.FormValue("nome")
		cargo := r.FormValue("cargo")

		if idStr != "" {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				alertAndBack(w, "ID inválido: deve ser numérico")
				return
			}
			// Verifica se o ID já existe
			var exists int
			err = app.DB.QueryRow("SELECT 1 FROM funcionarios WHERE id = ?", id).Scan(&exists)
			if err == nil {
				// ID já existe: exibe alerta e volta para a página anterior
				alertAndBack(w, fmt.Sprintf("O ID %d já está ocupado!", id))
				return
			}
			// Insere com ID manual
			if _, err := app.DB.Exec("INSERT INTO funcionarios (id, nome, cargo) VALUES (?, ?, ?)", id, nome, cargo); err != nil {
				log.Printf("Erro ao salvar funcionário (ID manual): %v", err)
				alertAndBack(w, "Erro ao salvar funcionário. Tente novamente.")
				return
			}
		} else {
			// Insere sem ID (o banco gera automaticamente)
			if _, err := app.DB.Exec("INSERT INTO funcionarios (nome, cargo) VALUES (?, ?)", nome, cargo); err != nil {
				log.Printf("Erro ao salvar funcionário (Auto ID): %v", err)
				alertAndBack(w, "Erro ao salvar funcionário. Tente novamente.")
				return
			}
		}

		http.Redirect(w, r, "/page/deletar_funcionario", http.StatusSeeOther)
	}
}

type RetiradaViewData struct {
	Funcionarios []models.Funcionario
	Produtos     []string
}

func (app *App) PageNovaRetirada(w http.ResponseWriter, r *http.Request) {
	rows, _ := app.DB.Query("SELECT id, nome, cargo FROM funcionarios WHERE ativo = 1")
	defer rows.Close()
	var lista []models.Funcionario
	for rows.Next() {
		var f models.Funcionario
		rows.Scan(&f.ID, &f.Nome, &f.Cargo)
		lista = append(lista, f)
	}

	produtos := app.GetTiposProdutos()

	data := RetiradaViewData{
		Funcionarios: lista,
		Produtos:     produtos,
	}
	app.Tmpl.ExecuteTemplate(w, "retirada.html", data)
}

func (app *App) ActionSalvarRetirada(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		funcID, err := strconv.Atoi(strings.TrimSpace(r.FormValue("funcionario_id")))
		if err != nil {
			alertAndBack(w, "ID do funcionário inválido")
			return
		}
		tipo := r.FormValue("tipo")
		qtd, err := strconv.Atoi(strings.TrimSpace(r.FormValue("quantidade")))
		if err != nil {
			alertAndBack(w, "Quantidade inválida")
			return
		}

		// Verifica se o funcionário está ativo
		var ativo int
		err = app.DB.QueryRow("SELECT ativo FROM funcionarios WHERE id = ?", funcID).Scan(&ativo)
		if err != nil {
			alertAndBack(w, "Funcionário não encontrado")
			return
		}
		if ativo == 0 {
			alertAndBack(w, "Funcionário desativado! Não é possível realizar retiradas.")
			return
		}

		agora := time.Now()
		data := agora.Format("2006-01-02")
		hora := agora.Format("15:04:05")

		if _, err := app.DB.Exec("INSERT INTO produtos (data, hora, tipo, quantidade, funcionario_id) VALUES (?, ?, ?, ?, ?)", data, hora, tipo, qtd, funcID); err != nil {
			log.Printf("Erro ao salvar produto: %v", err)
			alertAndBack(w, "Erro ao salvar produto")
			return
		}

		nomeTabelaDia := fmt.Sprintf("retiradas_%s", agora.Format("20060102"))
		if _, err := app.DB.Exec(fmt.Sprintf(`create table if not exists %s (id integer not null primary key, data text, hora text, tipo text, quantidade integer, funcionario_id integer);`, nomeTabelaDia)); err != nil {
			log.Printf("Erro ao criar tabela diária: %v", err)
		}
		if _, err := app.DB.Exec(fmt.Sprintf("INSERT INTO %s (data, hora, tipo, quantidade, funcionario_id) VALUES (?, ?, ?, ?, ?)", nomeTabelaDia), data, hora, tipo, qtd, funcID); err != nil {
			log.Printf("Erro ao salvar na tabela do dia: %v", err)
		}

		http.Redirect(w, r, "/page/home", http.StatusSeeOther)
	}
}

// Estrutura auxiliar para a view de gerenciamento (inclui status)
type FuncionarioView struct {
	models.Funcionario
	Ativo int
}

type DeleteFuncData struct {
	Funcionarios []FuncionarioView
	Page         int
	PrevPage     int
	NextPage     int
	QueryID      string
	QueryNome    string
	QueryCargo   string
	QueryStatus  string
}

func (app *App) PageDeletarFuncionario(w http.ResponseWriter, r *http.Request) {
	// Verifica autenticação (Protegido por senha)
	if !app.isAuthenticated(w, r) {
		http.Redirect(w, r, "/page/configurar", http.StatusSeeOther)
		return
	}

	qID := r.URL.Query().Get("q_id")
	qNome := r.URL.Query().Get("q_nome")
	qCargo := r.URL.Query().Get("q_cargo")
	qStatus := r.URL.Query().Get("q_status")

	// Paginação
	pageStr := r.URL.Query().Get("page")
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	limit := 10
	offset := (page - 1) * limit

	// Busca todos (ativos e inativos) para permitir reativação
	sqlQuery := "SELECT id, nome, cargo, ativo FROM funcionarios WHERE 1=1"
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
	if qStatus != "" && qStatus != "all" {
		sqlQuery += " AND ativo = ?"
		args = append(args, qStatus)
	}

	sqlQuery += " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := app.DB.Query(sqlQuery, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var lista []FuncionarioView
	for rows.Next() {
		var f FuncionarioView
		rows.Scan(&f.ID, &f.Nome, &f.Cargo, &f.Ativo)
		lista = append(lista, f)
	}

	data := DeleteFuncData{
		Funcionarios: lista,
		Page:         page,
		PrevPage:     page - 1,
		NextPage:     page + 1,
		QueryID:      qID,
		QueryNome:    qNome,
		QueryCargo:   qCargo,
		QueryStatus:  qStatus,
	}
	app.Tmpl.ExecuteTemplate(w, "delete_funcionario.html", data)
}

func (app *App) ActionStatusFuncionario(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if !app.isAuthenticated(w, r) {
			http.Redirect(w, r, "/page/configurar", http.StatusSeeOther)
			return
		}

		id, err := strconv.Atoi(r.FormValue("id"))
		status, errStatus := strconv.Atoi(r.FormValue("status")) // 1=Ativo, 0=Desativado, 2=Férias
		if err != nil {
			alertAndBack(w, "ID inválido")
			return
		}
		if errStatus != nil {
			alertAndBack(w, "Status inválido")
			return
		}

		// Atualiza o status (Soft Delete / Férias / Reativar)
		app.DB.Exec("UPDATE funcionarios SET ativo = ? WHERE id = ?", status, id)
		http.Redirect(w, r, "/page/deletar_funcionario", http.StatusSeeOther)
	}
}

func (app *App) ActionExcluirFuncionario(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if !app.isAuthenticated(w, r) {
			http.Redirect(w, r, "/page/configurar", http.StatusSeeOther)
			return
		}

		// Verifica senha enviada no form (Segurança extra para exclusão definitiva)
		senha := r.FormValue("senha")
		if senha != app.GetSenha() {
			alertAndBack(w, "Senha incorreta! Ação cancelada.")
			return
		}

		id, err := strconv.Atoi(r.FormValue("id"))
		if err != nil {
			alertAndBack(w, "ID inválido")
			return
		}

		// HARD DELETE: Apaga do banco de dados e remove histórico
		tx, err := app.DB.Begin()
		if err != nil {
			alertAndBack(w, "Erro interno no banco")
			return
		}
		defer tx.Rollback()

		// 1. Deleta histórico de produtos
		if _, err := tx.Exec("DELETE FROM produtos WHERE funcionario_id = ?", id); err != nil {
			log.Printf("Erro ao deletar produtos do funcionário %d: %v", id, err)
			alertAndBack(w, "Erro ao limpar histórico")
			return
		}

		// 2. Deleta o funcionário
		if _, err := tx.Exec("DELETE FROM funcionarios WHERE id = ?", id); err != nil {
			log.Printf("Erro ao deletar funcionário %d: %v", id, err)
			alertAndBack(w, "Erro ao deletar funcionário")
			return
		}

		if err := tx.Commit(); err != nil {
			alertAndBack(w, "Erro ao confirmar exclusão")
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
		newID, err := strconv.Atoi(strings.TrimSpace(r.FormValue("id")))
		if err != nil {
			alertAndBack(w, "ID inválido")
			return
		}
		nome := r.FormValue("nome")
		cargo := r.FormValue("cargo")

		// Se o ID mudou, verifica se o novo ID já existe
		if oldID != newID {
			var exists int
			err = app.DB.QueryRow("SELECT 1 FROM funcionarios WHERE id = ?", newID).Scan(&exists)
			if err == nil {
				alertAndBack(w, fmt.Sprintf("O ID %d já está ocupado!", newID))
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
		rows, err = app.DB.Query("SELECT id, nome, cargo FROM funcionarios WHERE cargo = ? AND ativo = 1 ORDER BY nome", cargoFilter)
	} else {
		// Se não tem, busca todos
		rows, err = app.DB.Query("SELECT id, nome, cargo FROM funcionarios WHERE ativo = 1 ORDER BY nome")
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
	cargos := app.GetCargos()
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

	cargosDB := app.GetCargos()

	// Se nenhum cargo foi selecionado, seleciona todos por padrão
	if len(cargosSelected) == 0 {
		cargosSelected = cargosDB
	}

	tabelas := make(map[string][]models.Quadro)
	var cargosOrdenados []string
	selMap := make(map[string]bool)
	for _, c := range cargosSelected {
		selMap[c] = true
	}

	for _, cargo := range cargosDB {
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

	// Verifica se foi solicitado exportação
	format := r.URL.Query().Get("format")
	if format == "excel" || format == "libre" {
		filename := fmt.Sprintf("escala_%s", strings.ReplaceAll(data, "-", ""))

		if format == "excel" {
			// Gera XML Spreadsheet 2003 (Compatível com Excel)
			w.Header().Set("Content-Type", "application/vnd.ms-excel")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=%s.xls", filename))

			w.Write([]byte(`<?xml version="1.0"?><?mso-application progid="Excel.Sheet"?><Workbook xmlns="urn:schemas-microsoft-com:office:spreadsheet" xmlns:ss="urn:schemas-microsoft-com:office:spreadsheet"><Worksheet ss:Name="Escala"><Table>`))
			fmt.Fprintf(w, `<Row><Cell><Data ss:Type="String">Escala do Dia: %s</Data></Cell></Row>`, data)
			w.Write([]byte(`<Row></Row>`))

			for _, cargo := range cargosOrdenados {
				fmt.Fprintf(w, `<Row><Cell><Data ss:Type="String">Cargo: %s</Data></Cell></Row>`, escapeXML(cargo))
				if cargo == "Operador" {
					w.Write([]byte(`<Row><Cell><Data ss:Type="String">Hora</Data></Cell><Cell><Data ss:Type="String">Funcionário</Data></Cell><Cell><Data ss:Type="String">Caixa 1</Data></Cell><Cell><Data ss:Type="String">Caixa 2</Data></Cell><Cell><Data ss:Type="String">Extra</Data></Cell></Row>`))
				} else {
					w.Write([]byte(`<Row><Cell><Data ss:Type="String">Hora</Data></Cell><Cell><Data ss:Type="String">Funcionário</Data></Cell><Cell><Data ss:Type="String">Tarefa 1</Data></Cell><Cell><Data ss:Type="String">Tarefa 2</Data></Cell><Cell><Data ss:Type="String">Tarefa 3</Data></Cell></Row>`))
				}

				horas := tabelas[cargo]
				for i, quadro := range horas {
					h := i + 1
					for _, p := range quadro.Pessoas {
						t1, t2, t3 := "", "", ""
						if cargo == "Operador" {
							t1, t2, t3 = p.Caixa1, p.Caixa2, p.Caixa3
						} else {
							t1, t2, t3 = p.Tarefa1, p.Tarefa2, p.Tarefa3
						}
						fmt.Fprintf(w, `<Row><Cell><Data ss:Type="Number">%d</Data></Cell><Cell><Data ss:Type="String">%s</Data></Cell><Cell><Data ss:Type="String">%s</Data></Cell><Cell><Data ss:Type="String">%s</Data></Cell><Cell><Data ss:Type="String">%s</Data></Cell></Row>`,
							h, escapeXML(p.NomeDoFuncionario), escapeXML(t1), escapeXML(t2), escapeXML(t3))
					}
				}
				w.Write([]byte(`<Row></Row>`))
			}
			w.Write([]byte(`</Table></Worksheet></Workbook>`))

		} else {
			// LibreOffice (ODS)
			w.Header().Set("Content-Type", "application/vnd.oasis.opendocument.spreadsheet")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=%s.fods", filename))

			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><office:document xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" xmlns:table="urn:oasis:names:tc:opendocument:xmlns:table:1.0" xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0" office:mimetype="application/vnd.oasis.opendocument.spreadsheet"><office:body><office:spreadsheet><table:table table:name="Escala">`))

			// Title Row
			fmt.Fprintf(w, `<table:table-row><table:table-cell office:value-type="string"><text:p>Escala do Dia: %s</text:p></table:table-cell></table:table-row>`, data)
			w.Write([]byte(`<table:table-row></table:table-row>`))

			for _, cargo := range cargosOrdenados {
				fmt.Fprintf(w, `<table:table-row><table:table-cell office:value-type="string"><text:p>Cargo: %s</text:p></table:table-cell></table:table-row>`, escapeXML(cargo))

				if cargo == "Operador" {
					w.Write([]byte(`<table:table-row><table:table-cell office:value-type="string"><text:p>Hora</text:p></table:table-cell><table:table-cell office:value-type="string"><text:p>Funcionário</text:p></table:table-cell><table:table-cell office:value-type="string"><text:p>Caixa 1</text:p></table:table-cell><table:table-cell office:value-type="string"><text:p>Caixa 2</text:p></table:table-cell><table:table-cell office:value-type="string"><text:p>Extra</text:p></table:table-cell></table:table-row>`))
				} else {
					w.Write([]byte(`<table:table-row><table:table-cell office:value-type="string"><text:p>Hora</text:p></table:table-cell><table:table-cell office:value-type="string"><text:p>Funcionário</text:p></table:table-cell><table:table-cell office:value-type="string"><text:p>Tarefa 1</text:p></table:table-cell><table:table-cell office:value-type="string"><text:p>Tarefa 2</text:p></table:table-cell><table:table-cell office:value-type="string"><text:p>Tarefa 3</text:p></table:table-cell></table:table-row>`))
				}

				horas := tabelas[cargo]
				for i, quadro := range horas {
					h := i + 1
					for _, p := range quadro.Pessoas {
						t1, t2, t3 := "", "", ""
						if cargo == "Operador" {
							t1, t2, t3 = p.Caixa1, p.Caixa2, p.Caixa3
						} else {
							t1, t2, t3 = p.Tarefa1, p.Tarefa2, p.Tarefa3
						}
						fmt.Fprintf(w, `<table:table-row><table:table-cell office:value-type="float" office:value="%d"><text:p>%d</text:p></table:table-cell><table:table-cell office:value-type="string"><text:p>%s</text:p></table:table-cell><table:table-cell office:value-type="string"><text:p>%s</text:p></table:table-cell><table:table-cell office:value-type="string"><text:p>%s</text:p></table:table-cell><table:table-cell office:value-type="string"><text:p>%s</text:p></table:table-cell></table:table-row>`,
							h, h, escapeXML(p.NomeDoFuncionario), escapeXML(t1), escapeXML(t2), escapeXML(t3))
					}
				}
				w.Write([]byte(`<table:table-row></table:table-row>`))
			}
			w.Write([]byte(`</table:table></office:spreadsheet></office:body></office:document>`))
		}
		return
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
	ID            int
	Nome          string
	DataInicio    string
	DataFim       string
	HoraInicio    string
	HoraFim       string
	ProdutoFilter string
	ListaProdutos []string
	Produtos      []models.Produto
	Soma          int
	Media         float64
	MediaPorHora  float64
}

func (app *App) PageHistoricoFuncionario(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, _ := strconv.Atoi(idStr)
	dataInicio := r.URL.Query().Get("data_inicio")
	dataFim := r.URL.Query().Get("data_fim")
	horaInicio := r.URL.Query().Get("hora_inicio")
	horaFim := r.URL.Query().Get("hora_fim")
	produtoFilter := r.URL.Query().Get("produto")
	export := r.URL.Query().Get("export")
	format := r.URL.Query().Get("format")

	if dataInicio == "" {
		dataInicio = time.Now().Format("2006-01-02")
	}
	if dataFim == "" {
		dataFim = time.Now().Format("2006-01-02")
	}

	// Busca nome do funcionário para exibir no cabeçalho
	var nome string
	app.DB.QueryRow("SELECT nome FROM funcionarios WHERE id = ?", id).Scan(&nome)

	// Construção da Query com filtros opcionais de hora
	sqlQuery := "SELECT id, data, hora, tipo, quantidade, funcionario_id FROM produtos WHERE funcionario_id = ? AND data >= ? AND data <= ?"
	args := []interface{}{id, dataInicio, dataFim}

	if horaInicio != "" {
		sqlQuery += " AND hora >= ?"
		args = append(args, horaInicio)
	}
	if horaFim != "" {
		sqlQuery += " AND hora <= ?"
		args = append(args, horaFim)
	}
	if produtoFilter != "" {
		sqlQuery += " AND tipo = ?"
		args = append(args, produtoFilter)
	}
	sqlQuery += " ORDER BY data DESC, hora DESC"

	rows, err := app.DB.Query(sqlQuery, args...)
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

	// Exportação para CSV
	if export == "true" {
		filename := fmt.Sprintf("historico_%s_%s", strings.ReplaceAll(nome, " ", "_"), time.Now().Format("20060102"))

		if format == "excel" {
			// Gera XML Spreadsheet 2003 (Compatível com Excel)
			w.Header().Set("Content-Type", "application/vnd.ms-excel")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=%s.xls", filename))

			w.Write([]byte(`<?xml version="1.0"?><?mso-application progid="Excel.Sheet"?><Workbook xmlns="urn:schemas-microsoft-com:office:spreadsheet" xmlns:ss="urn:schemas-microsoft-com:office:spreadsheet"><Worksheet ss:Name="Historico"><Table>`))

			// Cabeçalho
			w.Write([]byte(`<Row><Cell><Data ss:Type="String">ID</Data></Cell><Cell><Data ss:Type="String">Data</Data></Cell><Cell><Data ss:Type="String">Hora</Data></Cell><Cell><Data ss:Type="String">Produto</Data></Cell><Cell><Data ss:Type="String">Quantidade</Data></Cell><Cell><Data ss:Type="String">Funcionário</Data></Cell></Row>`))

			// Dados
			for _, p := range lista {
				fmt.Fprintf(w, `<Row><Cell><Data ss:Type="Number">%d</Data></Cell><Cell><Data ss:Type="String">%s</Data></Cell><Cell><Data ss:Type="String">%s</Data></Cell><Cell><Data ss:Type="String">%s</Data></Cell><Cell><Data ss:Type="Number">%d</Data></Cell><Cell><Data ss:Type="String">%s</Data></Cell></Row>`,
					p.ID, p.Data, p.Hora, escapeXML(p.Tipo), p.Quantidade, escapeXML(nome))
			}

			w.Write([]byte(`</Table></Worksheet></Workbook>`))
			return

		} else if format == "libre" {
			// Gera Flat ODS (Compatível com LibreOffice)
			w.Header().Set("Content-Type", "application/vnd.oasis.opendocument.spreadsheet")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=%s.fods", filename))

			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><office:document xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" xmlns:table="urn:oasis:names:tc:opendocument:xmlns:table:1.0" xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0" office:mimetype="application/vnd.oasis.opendocument.spreadsheet"><office:body><office:spreadsheet><table:table table:name="Historico">`))

			// Cabeçalho
			w.Write([]byte(`<table:table-row><table:table-cell office:value-type="string"><text:p>ID</text:p></table:table-cell><table:table-cell office:value-type="string"><text:p>Data</text:p></table:table-cell><table:table-cell office:value-type="string"><text:p>Hora</text:p></table:table-cell><table:table-cell office:value-type="string"><text:p>Produto</text:p></table:table-cell><table:table-cell office:value-type="string"><text:p>Quantidade</text:p></table:table-cell><table:table-cell office:value-type="string"><text:p>Funcionário</text:p></table:table-cell></table:table-row>`))

			// Dados
			for _, p := range lista {
				fmt.Fprintf(w, `<table:table-row><table:table-cell office:value-type="float" office:value="%d"><text:p>%d</text:p></table:table-cell><table:table-cell office:value-type="string"><text:p>%s</text:p></table:table-cell><table:table-cell office:value-type="string"><text:p>%s</text:p></table:table-cell><table:table-cell office:value-type="string"><text:p>%s</text:p></table:table-cell><table:table-cell office:value-type="float" office:value="%d"><text:p>%d</text:p></table:table-cell><table:table-cell office:value-type="string"><text:p>%s</text:p></table:table-cell></table:table-row>`,
					p.ID, p.ID, p.Data, p.Hora, escapeXML(p.Tipo), p.Quantidade, p.Quantidade, escapeXML(nome))
			}

			w.Write([]byte(`</table:table></office:spreadsheet></office:body></office:document>`))
			return

		} else {
			// CSV (Padrão)
			w.Header().Set("Content-Type", "text/csv")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=%s.csv", filename))

			writer := csv.NewWriter(w)
			defer writer.Flush()

			// BOM para compatibilidade com Excel (UTF-8)
			w.Write([]byte{0xEF, 0xBB, 0xBF})

			writer.Write([]string{"ID", "Data", "Hora", "Tipo", "Quantidade", "Funcionário"})
			for _, p := range lista {
				writer.Write([]string{
					strconv.Itoa(p.ID),
					p.Data,
					p.Hora,
					p.Tipo,
					strconv.Itoa(p.Quantidade),
					nome,
				})
			}
			return
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
		ID:            id,
		Nome:          nome,
		DataInicio:    dataInicio,
		DataFim:       dataFim,
		HoraInicio:    horaInicio,
		HoraFim:       horaFim,
		ProdutoFilter: produtoFilter,
		ListaProdutos: app.GetTiposProdutos(),
		Produtos:      lista,
		Soma:          soma,
		Media:         media,
		MediaPorHora:  mediaPorHora,
	}
	app.Tmpl.ExecuteTemplate(w, "historico_funcionario.html", data)
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// Helper para exibir alerta e voltar
func alertAndBack(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<script>alert('%s'); window.history.back();</script>", msg)
}

// --- CONFIGURAÇÃO ---

// Helper interno para verificar autenticação
func (app *App) isAuthenticated(w http.ResponseWriter, r *http.Request) bool {
	cookie, err := r.Cookie("admin_auth")
	if err != nil || cookie.Value != "true" {
		return false
	}
	// Estende a sessão por mais 1 minuto (sliding expiration) sempre que houver atividade
	http.SetCookie(w, &http.Cookie{
		Name:     "admin_auth",
		Value:    "true",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   60, // Reinicia contagem de 60 segundos
	})
	return true
}

type ConfigViewData struct {
	Cargos   []string
	Produtos []string
}

func (app *App) PageConfigurar(w http.ResponseWriter, r *http.Request) {
	if !app.isAuthenticated(w, r) {
		app.Tmpl.ExecuteTemplate(w, "login_config.html", nil)
		return
	}

	cargos := app.GetCargos()
	produtos := app.GetTiposProdutos()
	app.Tmpl.ExecuteTemplate(w, "configurar.html", ConfigViewData{Cargos: cargos, Produtos: produtos})
}

func (app *App) ActionAdicionarCargo(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if !app.isAuthenticated(w, r) {
			http.Redirect(w, r, "/page/configurar", http.StatusSeeOther)
			return
		}

		nome := strings.TrimSpace(r.FormValue("nome"))
		if nome != "" {
			app.DB.Exec("INSERT INTO cargos (nome) VALUES (?)", nome)
		}
		http.Redirect(w, r, "/page/configurar", http.StatusSeeOther)
	}
}

// --- AUTENTICAÇÃO E API DE SENHA ---

func (app *App) ActionLoginConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		senhaInput := r.FormValue("senha")
		senhaReal := app.GetSenha()

		if senhaInput == senhaReal {
			// Define cookie de sessão simples
			http.SetCookie(w, &http.Cookie{
				Name:     "admin_auth",
				Value:    "true",
				Path:     "/",
				HttpOnly: true,
				MaxAge:   60, // Expira em 1 minuto (60 segundos)
			})
			http.Redirect(w, r, "/page/configurar", http.StatusSeeOther)
			return
		}
		alertAndBack(w, "Senha incorreta!")
	}
}

func (app *App) APIUpdateSenha(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// Aceita via Form ou JSON
		novaSenha := r.FormValue("senha")
		if novaSenha == "" {
			var dados struct {
				Senha string `json:"senha"`
			}
			json.NewDecoder(r.Body).Decode(&dados)
			novaSenha = dados.Senha
		}

		if novaSenha != "" {
			app.SetSenha(novaSenha)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Senha atualizada com sucesso!"))
		} else {
			http.Error(w, "Nova senha não fornecida", http.StatusBadRequest)
		}
	} else {
		http.Error(w, "Método não permitido. Use POST.", http.StatusMethodNotAllowed)
	}
}

func (app *App) ActionUpdateSenhaGET(w http.ResponseWriter, r *http.Request) {
	// Esperado: /page/home/senha/NOVA_SENHA
	prefix := "/page/home/senha/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.NotFound(w, r)
		return
	}

	novaSenha := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, prefix))
	if novaSenha != "" {
		app.SetSenha(novaSenha)
		log.Printf("Senha de administrador alterada via URL para: %s", novaSenha)
		http.Redirect(w, r, "/page/home", http.StatusSeeOther)
	} else {
		http.Error(w, "Senha vazia não permitida", http.StatusBadRequest)
	}
}

func (app *App) ActionRemoverCargo(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if !app.isAuthenticated(w, r) {
			http.Redirect(w, r, "/page/configurar", http.StatusSeeOther)
			return
		}

		nome := strings.TrimSpace(r.FormValue("nome"))
		if _, err := app.DB.Exec("DELETE FROM cargos WHERE nome = ?", nome); err != nil {
			log.Printf("Erro ao remover cargo: %v", err)
		}
		http.Redirect(w, r, "/page/configurar", http.StatusSeeOther)
	}
}

func (app *App) ActionLogoutConfig(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "admin_auth",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/page/home", http.StatusSeeOther)
}

// --- GERENCIAMENTO DE PRODUTOS ---

func (app *App) ActionAdicionarProduto(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if !app.isAuthenticated(w, r) {
			http.Redirect(w, r, "/page/configurar", http.StatusSeeOther)
			return
		}

		nome := strings.TrimSpace(r.FormValue("nome"))
		if nome != "" {
			app.DB.Exec("INSERT INTO tipos_produtos (nome) VALUES (?)", nome)
		}
		http.Redirect(w, r, "/page/configurar", http.StatusSeeOther)
	}
}

func (app *App) ActionRemoverProduto(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if !app.isAuthenticated(w, r) {
			http.Redirect(w, r, "/page/configurar", http.StatusSeeOther)
			return
		}

		nome := strings.TrimSpace(r.FormValue("nome"))
		if _, err := app.DB.Exec("DELETE FROM tipos_produtos WHERE nome = ?", nome); err != nil {
			log.Printf("Erro ao remover produto: %v", err)
		}
		http.Redirect(w, r, "/page/configurar", http.StatusSeeOther)
	}
}
