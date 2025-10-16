//go:build integration

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

/*
Test Helpers para PostgreSQL com Testcontainers REAL

Este é um exemplo REAL de uso de testcontainers:
- Sobe um container Docker do PostgreSQL
- Cria banco de dados de teste
- Retorna connection string
- Cleanup automático após testes

Diferente do SQLite (que é embarcado), PostgreSQL PRECISA de testcontainers.

Referências:
- https://golang.testcontainers.org/modules/postgres/
- https://eltonminetto.dev/post/2024-02-15-using-test-helpers/
*/

const (
	defaultDatabase = "testdb"
	defaultUser     = "testuser"
	defaultPassword = "testpass"
)

// PostgresContainer encapsula o container e a conexão
type PostgresContainer struct {
	Container testcontainers.Container
	DB        *sql.DB
	ConnStr   string
}

// SetupPostgresContainer cria e inicia um container PostgreSQL real
// Este é o VERDADEIRO uso de testcontainers!
func SetupPostgresContainer(t *testing.T, ctx context.Context) (*PostgresContainer, func()) {
	t.Helper()

	// Criar container PostgreSQL usando o módulo oficial
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase(defaultDatabase),
		postgres.WithUsername(defaultUser),
		postgres.WithPassword(defaultPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)

	// Obter connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Conectar ao banco
	db, err := sql.Open("postgres", connStr)
	require.NoError(t, err)

	// Verificar conexão
	err = db.PingContext(ctx)
	require.NoError(t, err)

	container := &PostgresContainer{
		Container: pgContainer,
		DB:        db,
		ConnStr:   connStr,
	}

	// Cleanup function
	cleanup := func() {
		if db != nil {
			_ = db.Close()
		}
		if pgContainer != nil {
			_ = pgContainer.Terminate(ctx)
		}
	}

	return container, cleanup
}

// CreateTestSchema cria a tabela books no PostgreSQL
func CreateTestSchema(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	schema := `
		CREATE TABLE IF NOT EXISTS books (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			author TEXT NOT NULL,
			category INTEGER NOT NULL
		)
	`

	_, err := db.ExecContext(ctx, schema)
	require.NoError(t, err)
}

// DropTestSchema remove a tabela books
func DropTestSchema(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	_, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS books CASCADE")
	require.NoError(t, err)
}

// CleanupDatabase remove todos os registros da tabela books
func CleanupDatabase(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	_, err := db.ExecContext(ctx, "TRUNCATE TABLE books RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

// PopulateSampleData insere dados de exemplo para testes
func PopulateSampleData(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

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
		query := `INSERT INTO books (title, author, category) VALUES ($1, $2, $3)`
		_, err := db.ExecContext(ctx, query, book.title, book.author, book.category)
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

	query := "SELECT title, author, category FROM books WHERE id = $1"
	err := db.QueryRowContext(ctx, query, id).Scan(&title, &author, &category)
	require.NoError(t, err)

	return
}

// WaitForDatabase aguarda o banco estar pronto (útil para testes)
func WaitForDatabase(t *testing.T, db *sql.DB, maxAttempts int) {
	t.Helper()

	for i := 0; i < maxAttempts; i++ {
		if err := db.Ping(); err == nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	require.FailNow(t, "database not ready after maximum attempts")
}

// CreateTestRepository cria um repositório para testes
func CreateTestRepository(t *testing.T, connStr string) *Repository {
	t.Helper()

	repo, err := NewRepository(connStr)
	require.NoError(t, err)

	return repo
}

// GetContainerLogs obtém os logs do container (útil para debug)
func GetContainerLogs(t *testing.T, ctx context.Context, container testcontainers.Container) string {
	t.Helper()

	logs, err := container.Logs(ctx)
	if err != nil {
		return fmt.Sprintf("error getting logs: %v", err)
	}
	defer logs.Close()

	buf := make([]byte, 1024)
	n, _ := logs.Read(buf)
	return string(buf[:n])
}
