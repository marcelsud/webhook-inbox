//go:build integration

package turso

import (
	"context"
	"fmt"
	"testing"

	"github.com/eminetto/post-turso/book"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Testes de Integração para o Repository usando testcontainers.

Estes testes verificam a integração completa do repositório com um banco de dados real.
Execute com: go test -tags=integration ./...

Diferente dos testes unitários (repository_test.go), esses:
- Usam um banco de dados real (SQLite in-memory)
- Testam toda a stack (não usam mocks)
- São mais lentes mas mais próximos da realidade
- Servem para validar queries SQL e comportamento real
*/

func TestRepository_Insert_Integration(t *testing.T) {
	t.Run("insert single book", func(t *testing.T) {
		ctx := context.Background()
		db := SetupLocalSQLite(t)
		defer db.Close()

		CreateTestSchema(t, ctx, db)

		repo := &Repository{}
		repo.SetDB(db)

		// Preparar livro de teste
		testBook := book.Book{
			Title:    "Neuromancer",
			Author:   "William Gibson",
			Category: book.Read,
		}

		// Executar
		id, err := repo.Insert(ctx, testBook)

		// Validar
		require.NoError(t, err)
		assert.Equal(t, int64(1), id)
		AssertBookCount(t, ctx, db, 1)
	})

	t.Run("insert multiple books", func(t *testing.T) {
		ctx := context.Background()
		db := SetupLocalSQLite(t)
		defer db.Close()

		CreateTestSchema(t, ctx, db)

		repo := &Repository{}
		repo.SetDB(db)

		books := []book.Book{
			{Title: "Book 1", Author: "Author 1", Category: book.Read},
			{Title: "Book 2", Author: "Author 2", Category: book.Reading},
			{Title: "Book 3", Author: "Author 3", Category: book.WantToRead},
		}

		// Executar
		for _, b := range books {
			id, err := repo.Insert(ctx, b)
			require.NoError(t, err)
			assert.Greater(t, id, int64(0))
		}

		// Validar
		AssertBookCount(t, ctx, db, 3)
	})

	t.Run("insert returns sequential IDs", func(t *testing.T) {
		ctx := context.Background()
		db := SetupLocalSQLite(t)
		defer db.Close()

		CreateTestSchema(t, ctx, db)

		repo := &Repository{}
		repo.SetDB(db)

		// Executar
		id1, _ := repo.Insert(ctx, book.Book{Title: "Book 1", Author: "A1", Category: book.Read})
		id2, _ := repo.Insert(ctx, book.Book{Title: "Book 2", Author: "A2", Category: book.Read})
		id3, _ := repo.Insert(ctx, book.Book{Title: "Book 3", Author: "A3", Category: book.Read})

		// Validar
		assert.Equal(t, int64(1), id1)
		assert.Equal(t, int64(2), id2)
		assert.Equal(t, int64(3), id3)
	})
}

func TestRepository_Select_Integration(t *testing.T) {
	t.Run("select existing book", func(t *testing.T) {
		ctx := context.Background()
		db := SetupLocalSQLite(t)
		defer db.Close()

		CreateTestSchema(t, ctx, db)
		PopulateSampleData(t, ctx, db)

		repo := &Repository{}
		repo.SetDB(db)

		// Executar
		b, err := repo.Select(ctx, 1)

		// Validar
		require.NoError(t, err)
		assert.Equal(t, int64(1), b.ID)
		assert.Equal(t, "Neuromancer", b.Title)
		assert.Equal(t, "William Gibson", b.Author)
		assert.Equal(t, book.Read, b.Category)
	})

	t.Run("select non-existent book returns error", func(t *testing.T) {
		ctx := context.Background()
		db := SetupLocalSQLite(t)
		defer db.Close()

		CreateTestSchema(t, ctx, db)

		repo := &Repository{}
		repo.SetDB(db)

		// Executar
		_, err := repo.Select(ctx, 999)

		// Validar
		require.Error(t, err)
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("select returns correct book from multiple", func(t *testing.T) {
		ctx := context.Background()
		db := SetupLocalSQLite(t)
		defer db.Close()

		CreateTestSchema(t, ctx, db)
		PopulateSampleData(t, ctx, db)

		repo := &Repository{}
		repo.SetDB(db)

		// Executar - buscar o segundo livro
		b, err := repo.Select(ctx, 2)

		// Validar
		require.NoError(t, err)
		assert.Equal(t, "Dune", b.Title)
		assert.Equal(t, "Frank Herbert", b.Author)
		assert.Equal(t, book.WantToRead, b.Category)
	})
}

func TestRepository_SelectAll_Integration(t *testing.T) {
	t.Run("select all returns all books", func(t *testing.T) {
		ctx := context.Background()
		db := SetupLocalSQLite(t)
		defer db.Close()

		CreateTestSchema(t, ctx, db)
		PopulateSampleData(t, ctx, db)

		repo := &Repository{}
		repo.SetDB(db)

		// Executar
		books, err := repo.SelectAll(ctx)

		// Validar
		require.NoError(t, err)
		assert.Equal(t, 3, len(books))
		assert.Equal(t, "Neuromancer", books[0].Title)
		assert.Equal(t, "Dune", books[1].Title)
		assert.Equal(t, "1984", books[2].Title)
	})

	t.Run("select all empty database returns error", func(t *testing.T) {
		ctx := context.Background()
		db := SetupLocalSQLite(t)
		defer db.Close()

		CreateTestSchema(t, ctx, db)

		repo := &Repository{}
		repo.SetDB(db)

		// Executar
		_, err := repo.SelectAll(ctx)

		// Validar
		require.Error(t, err)
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("select all after insertions", func(t *testing.T) {
		ctx := context.Background()
		db := SetupLocalSQLite(t)
		defer db.Close()

		CreateTestSchema(t, ctx, db)

		repo := &Repository{}
		repo.SetDB(db)

		// Inserir alguns livros
		repo.Insert(ctx, book.Book{Title: "Book A", Author: "Author A", Category: book.Read})
		repo.Insert(ctx, book.Book{Title: "Book B", Author: "Author B", Category: book.Reading})

		// Executar
		books, err := repo.SelectAll(ctx)

		// Validar
		require.NoError(t, err)
		assert.Equal(t, 2, len(books))
	})
}

func TestRepository_Update_Integration(t *testing.T) {
	t.Run("update existing book", func(t *testing.T) {
		ctx := context.Background()
		db := SetupLocalSQLite(t)
		defer db.Close()

		CreateTestSchema(t, ctx, db)
		PopulateSampleData(t, ctx, db)

		repo := &Repository{}
		repo.SetDB(db)

		// Preparar atualização
		updatedBook := book.Book{
			ID:       1,
			Title:    "Neuromancer - Remastered",
			Author:   "William Gibson",
			Category: book.Reading,
		}

		// Executar
		err := repo.Update(ctx, updatedBook)

		// Validar
		require.NoError(t, err)

		// Verificar se foi atualizado
		b, _ := repo.Select(ctx, 1)
		assert.Equal(t, "Neuromancer - Remastered", b.Title)
		assert.Equal(t, book.Reading, b.Category)
	})

	t.Run("update multiple fields", func(t *testing.T) {
		ctx := context.Background()
		db := SetupLocalSQLite(t)
		defer db.Close()

		CreateTestSchema(t, ctx, db)
		PopulateSampleData(t, ctx, db)

		repo := &Repository{}
		repo.SetDB(db)

		// Atualizar vários campos
		updatedBook := book.Book{
			ID:       2,
			Title:    "Dune - Extended Edition",
			Author:   "Frank Herbert (Updated)",
			Category: book.Read,
		}

		// Executar
		err := repo.Update(ctx, updatedBook)

		// Validar
		require.NoError(t, err)

		b, _ := repo.Select(ctx, 2)
		assert.Equal(t, "Dune - Extended Edition", b.Title)
		assert.Equal(t, "Frank Herbert (Updated)", b.Author)
		assert.Equal(t, book.Read, b.Category)
	})

	t.Run("update preserves other records", func(t *testing.T) {
		ctx := context.Background()
		db := SetupLocalSQLite(t)
		defer db.Close()

		CreateTestSchema(t, ctx, db)
		PopulateSampleData(t, ctx, db)

		repo := &Repository{}
		repo.SetDB(db)

		// Atualizar apenas o primeiro livro
		repo.Update(ctx, book.Book{
			ID:       1,
			Title:    "Updated",
			Author:   "Updated",
			Category: book.Read,
		})

		// Validar que outros registros não foram alterados
		b2, _ := repo.Select(ctx, 2)
		assert.Equal(t, "Dune", b2.Title)

		b3, _ := repo.Select(ctx, 3)
		assert.Equal(t, "1984", b3.Title)
	})
}

func TestRepository_Delete_Integration(t *testing.T) {
	t.Run("delete existing book", func(t *testing.T) {
		ctx := context.Background()
		db := SetupLocalSQLite(t)
		defer db.Close()

		CreateTestSchema(t, ctx, db)
		PopulateSampleData(t, ctx, db)

		repo := &Repository{}
		repo.SetDB(db)

		// Validar que existe
		AssertBookCount(t, ctx, db, 3)

		// Executar
		err := repo.Delete(ctx, 1)

		// Validar
		require.NoError(t, err)
		AssertBookCount(t, ctx, db, 2)

		// Verificar que foi realmente deletado
		_, err = repo.Select(ctx, 1)
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("delete multiple books one by one", func(t *testing.T) {
		ctx := context.Background()
		db := SetupLocalSQLite(t)
		defer db.Close()

		CreateTestSchema(t, ctx, db)
		PopulateSampleData(t, ctx, db)

		repo := &Repository{}
		repo.SetDB(db)

		// Executar
		repo.Delete(ctx, 1)
		AssertBookCount(t, ctx, db, 2)

		repo.Delete(ctx, 2)
		AssertBookCount(t, ctx, db, 1)

		repo.Delete(ctx, 3)

		// Validar - banco vazio deve retornar erro
		_, err := repo.SelectAll(ctx)
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("delete non-existent book does not error", func(t *testing.T) {
		ctx := context.Background()
		db := SetupLocalSQLite(t)
		defer db.Close()

		CreateTestSchema(t, ctx, db)

		repo := &Repository{}
		repo.SetDB(db)

		// Executar - deleting non-existent record
		err := repo.Delete(ctx, 999)

		// Validar - Go não retorna erro para DELETE sem matches
		require.NoError(t, err)
	})
}

func TestRepository_CRUD_Integration(t *testing.T) {
	t.Run("complete CRUD cycle", func(t *testing.T) {
		ctx := context.Background()
		db := SetupLocalSQLite(t)
		defer db.Close()

		CreateTestSchema(t, ctx, db)

		repo := &Repository{}
		repo.SetDB(db)

		// CREATE
		newBook := book.Book{
			Title:    "The Foundation",
			Author:   "Isaac Asimov",
			Category: book.Read,
		}
		id, err := repo.Insert(ctx, newBook)
		require.NoError(t, err)
		assert.Equal(t, int64(1), id)

		// READ
		retrieved, err := repo.Select(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, newBook.Title, retrieved.Title)
		assert.Equal(t, newBook.Author, retrieved.Author)

		// UPDATE
		retrieved.Title = "The Foundation - Special Edition"
		retrieved.Category = book.Reading
		err = repo.Update(ctx, retrieved)
		require.NoError(t, err)

		// READ updated
		updated, err := repo.Select(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, "The Foundation - Special Edition", updated.Title)
		assert.Equal(t, book.Reading, updated.Category)

		// DELETE
		err = repo.Delete(ctx, id)
		require.NoError(t, err)

		// READ after delete
		_, err = repo.Select(ctx, id)
		assert.Equal(t, ErrNotFound, err)
	})
}

func TestRepository_Concurrent_Integration(t *testing.T) {
	t.Run("concurrent inserts", func(t *testing.T) {
		ctx := context.Background()
		db := SetupLocalSQLite(t)
		defer db.Close()

		CreateTestSchema(t, ctx, db)

		repo := &Repository{}
		repo.SetDB(db)

		// Executar inserts concorrentes
		numGoroutines := 5
		done := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				b := book.Book{
					Title:    fmt.Sprintf("Book %d", index),
					Author:   fmt.Sprintf("Author %d", index),
					Category: book.Read,
				}
				_, err := repo.Insert(ctx, b)
				done <- err
			}(i)
		}

		// Verificar resultados
		for i := 0; i < numGoroutines; i++ {
			require.NoError(t, <-done)
		}

		// Validar
		AssertBookCount(t, ctx, db, numGoroutines)
	})
}
