package models

// Dados: Funcionarios
type Funcionario struct {
	ID    int    `json:"id"`
	Nome  string `json:"nome"`
	Cargo string `json:"cargo"`
}

// Dados: Produtos
type Produto struct {
	ID            int    `json:"id"`
	Data          string `json:"data"`
	Hora          string `json:"hora"`
	Tipo          string `json:"tipo"`
	Quantidade    int    `json:"quantidade"`
	FuncionarioID int    `json:"funcionario_id"`
}

//  ----

// EscalaPessoa unifica os dados de diferentes cargos para a escala
type EscalaPessoa struct {
	Data              string `json:"data"`
	FuncionarioID     int    `json:"funcionario_id"`
	NomeDoFuncionario string `json:"nome_do_funcionario"`
	Cargo             string `json:"cargo"`      // Adicionado para filtro
	Presente          bool   `json:"presente"`   // Corrigido de Veinho
	Intervalos        bool   `json:"intervalos"` // Corrigido de Entevalos
	Descanso          bool   `json:"descanso"`
	// Campos opcionais de Operador/Empacotador
	Caixa1  string `json:"caixa1,omitempty"`
	Caixa2  string `json:"caixa2,omitempty"`
	Caixa3  string `json:"caixa3,omitempty"`
	Tarefa1 string `json:"tarefa1,omitempty"`
	Tarefa2 string `json:"tarefa2,omitempty"`
	Tarefa3 string `json:"tarefa3,omitempty"`
}

type Quadro struct {
	Pessoas []EscalaPessoa `json:"pessoas"`
}

type DiaDeTrabalho struct {
	Data   string `json:"data"`
	Hora1  Quadro `json:"hora1"`
	Hora2  Quadro `json:"hora2"`
	Hora3  Quadro `json:"hora3"`
	Hora4  Quadro `json:"hora4"`
	Hora5  Quadro `json:"hora5"`
	Hora6  Quadro `json:"hora6"`
	Hora7  Quadro `json:"hora7"`
	Hora8  Quadro `json:"hora8"`
	Hora9  Quadro `json:"hora9"`
	Hora10 Quadro `json:"hora10"`
	Hora11 Quadro `json:"hora11"`
	Hora12 Quadro `json:"hora12"`
	Hora13 Quadro `json:"hora13"`
	Hora14 Quadro `json:"hora14"`
	Hora15 Quadro `json:"hora15"`
	Hora16 Quadro `json:"hora16"`
	Hora17 Quadro `json:"hora17"`
	Hora18 Quadro `json:"hora18"`
	Hora19 Quadro `json:"hora19"`
	Hora20 Quadro `json:"hora20"`
	Hora21 Quadro `json:"hora21"`
	Hora22 Quadro `json:"hora22"`
	Hora23 Quadro `json:"hora23"`
	Hora24 Quadro `json:"hora24"`
}
