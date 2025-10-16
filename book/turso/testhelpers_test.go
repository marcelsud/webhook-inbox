//go:build integration

package turso

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/glebarez/go-sqlite"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

/*
Test Helpers para testes de integração com testcontainers.
Referência: https://eltonminetto.dev/post/2024-02-15-using-test-helpers/
*/

// SQLiteContainer representa um container SQLite para testes
type SQLiteContainer struct {
	container testcontainers.Container
	db        *sql.DB
}

// SetupSQLiteContainer cria e inicia um container SQLite para testes
// Retorna um cleanup function que deve ser deferred
func SetupSQLiteContainer(t *testing.T, ctx context.Context) (*SQLiteContainer, func()) {
	t.Helper()

	// Para testes locais com SQLite, criamos um banco em arquivo temporário
	// Uma alternativa seria usar uma imagem de teste, mas SQLite é embarcado
	req := testcontainers.ContainerRequest{
		Image: "kevinconway/sqlite:latest",
		Name:  fmt.Sprintf("sqlite-test-%d", time.Now().UnixNano()),
		Cmd:   []string{},
		// Aguardar que o container esteja pronto
		WaitingFor: wait.ForLog("ready").WithStartupTimeout(10 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	// Para SQLite, usaremos o arquivo local em vez do container
	// Portanto, retornamos um helper simplificado
	cleanup := func() {
		if container != nil {
			_ = container.Terminate(ctx)
		}
	}

	return &SQLiteContainer{
		container: container,
	}, cleanup
}

// SetupLocalSQLite cria um banco de dados SQLite local para testes
// Essa é a abordagem pragmática já que SQLite não precisa de container
func SetupLocalSQLite(t *testing.T) *sql.DB {
	t.Helper()

	// Criar banco de dados em memória para testes rápidos
	// Usar ?mode=memory&cache=shared para suportar acesso concorrente
	db, err := sql.Open("sqlite", "file::memory:?mode=memory&cache=shared")
	require.NoError(t, err)

	// Testar conexão
	err = db.Ping()
	require.NoError(t, err)

	return db
}

// CreateTestSchema cria o schema de tabelas para testes
func CreateTestSchema(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	schema := `
	CREATE TABLE IF NOT EXISTS books (
		ID INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		author TEXT NOT NULL,
		category INTEGER NOT NULL
	);
	`

	_, err := db.ExecContext(ctx, schema)
	require.NoError(t, err)
}

// CleanupDatabase remove todos os registros da tabela books
func CleanupDatabase(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	_, err := db.ExecContext(ctx, "DELETE FROM books")
	require.NoError(t, err)
}

// PopulateSampleData popula dados de exemplo para testes
func PopulateSampleData(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	const insertSQL = `
	INSERT INTO books (title, author, category)
	VALUES (?, ?, ?)
	`

	// Inserir alguns livros de exemplo
	testBooks := []struct {
		title    string
		author   string
		category int
	}{
		{"Neuromancer", "William Gibson", 3},
		{"Dune", "Frank Herbert", 1},
		{"1984", "George Orwell", 2},
	}

	for _, book := range testBooks {
		_, err := db.ExecContext(ctx, insertSQL, book.title, book.author, book.category)
		require.NoError(t, err)
	}
}

// AssertBookCount verifica quantos livros estão no banco
func AssertBookCount(t *testing.T, ctx context.Context, db *sql.DB, expected int) {
	t.Helper()

	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM books").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, expected, count)
}

// GetBookByID busca um livro pelo ID (helper para assertions)
func GetBookByID(t *testing.T, ctx context.Context, db *sql.DB, id int64) (title, author string, category int) {
	t.Helper()

	err := db.QueryRowContext(ctx, "SELECT title, author, category FROM books WHERE id = ?", id).
		Scan(&title, &author, &category)
	require.NoError(t, err)

	return
}
