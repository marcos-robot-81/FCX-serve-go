package database

import (
	"database/sql"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

func Conectar() *sql.DB {
	// Garante que a pasta de dados existe
	if err := os.MkdirAll("./dados", 0755); err != nil {
		log.Fatal("Erro ao criar pasta de dados:", err)
	}

	db, err := sql.Open("sqlite", "./dados/dados.db")
	if err != nil {
		log.Fatal(err)
	}

	// Cria as tabelas iniciais
	sqlStmtFunc := `create table if not exists funcionarios (id integer not null primary key, nome text, cargo text);`
	db.Exec(sqlStmtFunc)

	// VERIFICAÇÃO DE MIGRAÇÃO (ativo):
	if _, err := db.Query("SELECT ativo FROM funcionarios LIMIT 1"); err != nil {
		log.Println("Coluna 'ativo' não encontrada em 'funcionarios'. Adicionando...")
		db.Exec("ALTER TABLE funcionarios ADD COLUMN ativo INTEGER DEFAULT 1")
	}

	// VERIFICAÇÃO DE MIGRAÇÃO:
	// Tenta verificar se a coluna 'funcionario_id' existe. Se der erro, é porque a tabela é antiga.
	if rows, err := db.Query("SELECT funcionario_id FROM produtos LIMIT 1"); err != nil {
		log.Println("Esquema antigo detectado em 'produtos'. Recriando tabela para corrigir erro...")
		db.Exec("DROP TABLE IF EXISTS produtos")
	} else {
		rows.Close()
	}

	sqlStmtProd := `create table if not exists produtos (id integer not null primary key, data text, hora text, tipo text, quantidade integer, funcionario_id integer);`
	db.Exec(sqlStmtProd)

	// Tabela para armazenar o JSON da escala diária (DiaDeTrabalho)
	sqlStmtEscala := `create table if not exists escalas (data text primary key, json_content text);`
	db.Exec(sqlStmtEscala)

	// Tabela de Cargos (Configurável)
	sqlStmtCargos := `create table if not exists cargos (id integer not null primary key, nome text unique);`
	db.Exec(sqlStmtCargos)

	// Se a tabela de cargos estiver vazia, insere os padrões
	var countCargos int
	db.QueryRow("SELECT count(*) FROM cargos").Scan(&countCargos)
	if countCargos == 0 {
		defaults := []string{"Operador", "Auxiliar", "Empacotador", "Apoio", "Líder"}
		for _, c := range defaults {
			db.Exec("INSERT INTO cargos (nome) VALUES (?)", c)
		}
	}

	// Tabela de Configurações (Senha)
	sqlStmtConfig := `create table if not exists configuracoes (chave text primary key, valor text);`
	db.Exec(sqlStmtConfig)

	// Define senha padrão se não existir
	var countConfig int
	db.QueryRow("SELECT count(*) FROM configuracoes WHERE chave = 'senha_admin'").Scan(&countConfig)
	if countConfig == 0 {
		db.Exec("INSERT INTO configuracoes (chave, valor) VALUES ('senha_admin', '8080')")
	}

	// Tabela de Tipos de Produtos (Configurável)
	sqlStmtTipos := `create table if not exists tipos_produtos (id integer not null primary key, nome text unique);`
	db.Exec(sqlStmtTipos)

	var countTipos int
	db.QueryRow("SELECT count(*) FROM tipos_produtos").Scan(&countTipos)
	if countTipos == 0 {
		db.Exec("INSERT INTO tipos_produtos (nome) VALUES ('Paninho')")
	}

	return db
}
