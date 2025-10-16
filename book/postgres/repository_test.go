//go:build !integration

package postgres

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/eminetto/post-turso/book"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Testes Unitários para Repository PostgreSQL

Estes testes usam sqlmock para simular o banco de dados sem precisar
de um banco real ou containers.

Executar com: go test ./book/postgres/...
(Sem -tags=integration)

Diferenças vs testes de integração:
- Rápidos (milissegundos)
- Sem dependências externas
- Testam lógica SQL, não comportamento real do DB
- Bons para CI/CD rápido
*/

func TestRepository_Insert_Unit(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &Repository{DB: db}
	ctx := context.Background()

	// Preparar expectativa
	rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
	mock.ExpectQuery(regexp.QuoteMeta(
		`INSERT INTO books (title, author, category)
		VALUES ($1, $2, $3)
		RETURNING id`,
	)).WithArgs("Test Title", "Test Author", 1).WillReturnRows(rows)

	// Executar
	id, err := repo.Insert(ctx, book.Book{
		Title:    "Test Title",
		Author:   "Test Author",
		Category: book.WantToRead,
	})

	// Validar
	require.NoError(t, err)
	assert.Equal(t, int64(1), id)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Select_Unit(t *testing.T) {
	t.Run("select existing book", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := &Repository{DB: db}
		ctx := context.Background()

		// Preparar expectativa
		rows := sqlmock.NewRows([]string{"id", "title", "author", "category"}).
			AddRow(1, "Clean Code", "Robert Martin", 3)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT id, title, author, category FROM books WHERE id = $1`,
		)).WithArgs(1).WillReturnRows(rows)

		// Executar
		b, err := repo.Select(ctx, 1)

		// Validar
		require.NoError(t, err)
		assert.Equal(t, int64(1), b.ID)
		assert.Equal(t, "Clean Code", b.Title)
		assert.Equal(t, "Robert Martin", b.Author)
		assert.Equal(t, book.Read, b.Category)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("select non-existent book returns error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := &Repository{DB: db}
		ctx := context.Background()

		// Preparar expectativa - retorna resultado vazio
		rows := sqlmock.NewRows([]string{"id", "title", "author", "category"})

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT id, title, author, category FROM books WHERE id = $1`,
		)).WithArgs(999).WillReturnRows(rows)

		// Executar
		_, err = repo.Select(ctx, 999)

		// Validar
		require.Error(t, err)
		assert.Equal(t, ErrNotFound, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_SelectAll_Unit(t *testing.T) {
	t.Run("select all books", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := &Repository{DB: db}
		ctx := context.Background()

		// Preparar expectativa
		rows := sqlmock.NewRows([]string{"id", "title", "author", "category"}).
			AddRow(1, "Book 1", "Author 1", 1).
			AddRow(2, "Book 2", "Author 2", 2).
			AddRow(3, "Book 3", "Author 3", 3)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT id, title, author, category FROM books ORDER BY id`,
		)).WillReturnRows(rows)

		// Executar
		books, err := repo.SelectAll(ctx)

		// Validar
		require.NoError(t, err)
		assert.Equal(t, 3, len(books))
		assert.Equal(t, "Book 1", books[0].Title)
		assert.Equal(t, "Book 2", books[1].Title)
		assert.Equal(t, "Book 3", books[2].Title)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("select all from empty database", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := &Repository{DB: db}
		ctx := context.Background()

		// Preparar expectativa - resultado vazio
		rows := sqlmock.NewRows([]string{"id", "title", "author", "category"})

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT id, title, author, category FROM books ORDER BY id`,
		)).WillReturnRows(rows)

		// Executar
		_, err = repo.SelectAll(ctx)

		// Validar
		require.Error(t, err)
		assert.Equal(t, ErrNotFound, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_Update_Unit(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &Repository{DB: db}
	ctx := context.Background()

	// Preparar expectativa
	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE books
		SET title = $1, author = $2, category = $3
		WHERE id = $4`,
	)).WithArgs("Updated Title", "Updated Author", 2, 1).
		WillReturnResult(sqlmock.NewResult(0, 1)) // 1 row affected

	// Executar
	err = repo.Update(ctx, book.Book{
		ID:       1,
		Title:    "Updated Title",
		Author:   "Updated Author",
		Category: book.Reading,
	})

	// Validar
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Update_NotFound_Unit(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &Repository{DB: db}
	ctx := context.Background()

	// Preparar expectativa - 0 rows affected
	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE books
		SET title = $1, author = $2, category = $3
		WHERE id = $4`,
	)).WithArgs("Title", "Author", 1, 999).
		WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows affected

	// Executar
	err = repo.Update(ctx, book.Book{
		ID:       999,
		Title:    "Title",
		Author:   "Author",
		Category: book.WantToRead,
	})

	// Validar
	require.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Delete_Unit(t *testing.T) {
	t.Run("delete existing book", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := &Repository{DB: db}
		ctx := context.Background()

		// Preparar expectativa
		mock.ExpectExec(regexp.QuoteMeta(
			`DELETE FROM books WHERE id = $1`,
		)).WithArgs(1).WillReturnResult(sqlmock.NewResult(0, 1)) // 1 row affected

		// Executar
		err = repo.Delete(ctx, 1)

		// Validar
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("delete non-existent book", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := &Repository{DB: db}
		ctx := context.Background()

		// Preparar expectativa - 0 rows affected
		mock.ExpectExec(regexp.QuoteMeta(
			`DELETE FROM books WHERE id = $1`,
		)).WithArgs(999).WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows affected

		// Executar
		err = repo.Delete(ctx, 999)

		// Validar
		require.Error(t, err)
		assert.Equal(t, ErrNotFound, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_CreateTable_Unit(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &Repository{DB: db}
	ctx := context.Background()

	// Preparar expectativa
	mock.ExpectExec(regexp.QuoteMeta(
		`CREATE TABLE IF NOT EXISTS books (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			author TEXT NOT NULL,
			category INTEGER NOT NULL
		)`,
	)).WillReturnResult(sqlmock.NewResult(0, 0))

	// Executar
	err = repo.CreateTable(ctx)

	// Validar
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
