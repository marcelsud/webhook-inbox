package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/eminetto/post-turso/book"
	_ "github.com/lib/pq" // PostgreSQL driver
)

/*
PostgreSQL Repository Implementation

Esta implementação demonstra:
- Como adaptar o mesmo Repository interface para diferentes bancos
- Diferenças de sintaxe entre SQLite e PostgreSQL
- Uso de placeholders ($1, $2) ao invés de (?)
- SERIAL ao invés de AUTOINCREMENT
- Integração com testcontainers para testes reais
*/

type Repository struct {
	DB *sql.DB
}

var ErrNotFound = errors.New("not found")

// NewRepository cria uma nova instância do repositório PostgreSQL com pool padrão (25, 5, 5 min)
func NewRepository(connectionString string) (*Repository, error) {
	return NewRepositoryWithPoolConfig(connectionString, 25, 5, 5)
}

// NewRepositoryWithPoolConfig cria uma nova instância do repositório PostgreSQL com configuração customizável
// maxOpenConns: máximo de conexões simultâneas (0 = ilimitado)
// maxIdleConns: máximo de conexões inativas mantidas no pool
// maxLifeMinutes: duração máxima em minutos que uma conexão pode ser reutilizada
func NewRepositoryWithPoolConfig(connectionString string, maxOpenConns, maxIdleConns, maxLifeMinutes int) (*Repository, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("opening postgres connection: %w", err)
	}

	// Testar conexão
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging postgres: %w", err)
	}

	// Configurar pool de conexões
	if maxOpenConns > 0 {
		db.SetMaxOpenConns(maxOpenConns)
	}
	if maxIdleConns > 0 {
		db.SetMaxIdleConns(maxIdleConns)
	}
	if maxLifeMinutes > 0 {
		db.SetConnMaxLifetime(time.Duration(maxLifeMinutes) * time.Minute)
	}

	return &Repository{
		DB: db,
	}, nil
}

// Select busca um livro por ID
func (r *Repository) Select(ctx context.Context, id int64) (book.Book, error) {
	// PostgreSQL usa $1, $2, etc ao invés de ?
	query := "SELECT id, title, author, category FROM books WHERE id = $1"

	var b book.Book
	err := r.DB.QueryRowContext(ctx, query, id).Scan(
		&b.ID,
		&b.Title,
		&b.Author,
		&b.Category,
	)

	if err == sql.ErrNoRows {
		return book.Book{}, ErrNotFound
	}

	if err != nil {
		return book.Book{}, fmt.Errorf("selecting book: %w", err)
	}

	return b, nil
}

// SelectAll retorna todos os livros
func (r *Repository) SelectAll(ctx context.Context) ([]book.Book, error) {
	query := "SELECT id, title, author, category FROM books ORDER BY id"

	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("selecting books: %w", err)
	}
	defer rows.Close()

	var books []book.Book

	for rows.Next() {
		var b book.Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, &b.Category); err != nil {
			return nil, fmt.Errorf("scanning book: %w", err)
		}
		books = append(books, b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating books: %w", err)
	}

	if len(books) == 0 {
		return nil, ErrNotFound
	}

	return books, nil
}

// Insert insere um novo livro e retorna o ID gerado
func (r *Repository) Insert(ctx context.Context, b book.Book) (int64, error) {
	// PostgreSQL retorna o ID usando RETURNING
	query := `
		INSERT INTO books (title, author, category)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	var id int64
	err := r.DB.QueryRowContext(ctx, query, b.Title, b.Author, b.Category).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("inserting book: %w", err)
	}

	return id, nil
}

// Update atualiza um livro existente
func (r *Repository) Update(ctx context.Context, b book.Book) error {
	query := `
		UPDATE books
		SET title = $1, author = $2, category = $3
		WHERE id = $4
	`

	result, err := r.DB.ExecContext(ctx, query, b.Title, b.Author, b.Category, b.ID)
	if err != nil {
		return fmt.Errorf("updating book: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("getting rows affected: %w", err)
	}

	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// Delete remove um livro por ID
func (r *Repository) Delete(ctx context.Context, id int64) error {
	query := "DELETE FROM books WHERE id = $1"

	result, err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting book: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("getting rows affected: %w", err)
	}

	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// Close fecha a conexão com o banco de dados
func (r *Repository) Close(ctx context.Context) error {
	if r.DB != nil {
		return r.DB.Close()
	}
	return nil
}

// CreateTable cria a tabela books (útil para testes)
func (r *Repository) CreateTable(ctx context.Context) error {
	// PostgreSQL usa SERIAL ao invés de AUTOINCREMENT
	query := `
		CREATE TABLE IF NOT EXISTS books (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			author TEXT NOT NULL,
			category INTEGER NOT NULL
		)
	`

	_, err := r.DB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("creating table: %w", err)
	}

	return nil
}

// DropTable remove a tabela books (útil para testes)
func (r *Repository) DropTable(ctx context.Context) error {
	query := "DROP TABLE IF EXISTS books CASCADE"

	_, err := r.DB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("dropping table: %w", err)
	}

	return nil
}
