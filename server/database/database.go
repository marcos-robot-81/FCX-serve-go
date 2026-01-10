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

	return db
}
