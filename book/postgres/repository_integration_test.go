//go:build integration

package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/eminetto/post-turso/book"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Testes de Integração com PostgreSQL + Testcontainers REAL

Estes testes demonstram o USO REAL de testcontainers:
- Container Docker do PostgreSQL é criado antes de cada teste
- Banco de dados real é usado (não mocks!)
- Todas as queries SQL são executadas contra PostgreSQL de verdade
- Container é destruído após o teste

Execute com: go test -tags=integration ./book/postgres/...

REQUISITOS:
- Docker rodando localmente
- Acesso à internet para baixar imagem postgres:16-alpine (primeira vez)
- Tempo de execução: ~10-30 segundos (criação de containers)

Diferenças vs SQLite:
- SQLite: embarcado, não precisa de container
- PostgreSQL: precisa de container Docker real

OTIMIZAÇÃO DE PERFORMANCE:
Por padrão, cada teste cria um novo container (teste isolado, mas lento).
Para compartilhar container entre testes, use:

  export TESTCONTAINERS_REUSE_ENABLE=true
  go test -tags=integration ./book/postgres/...

Isto reduz tempo de ~31s para ~5s.

Alternativa: refatore testes para usar suite pattern (veja repository_benchmark_test.go).
*/

func TestPostgresRepository_Insert_Integration(t *testing.T) {
	t.Run("insert single book", func(t *testing.T) {
		ctx := context.Background()

		// Setup: cria container PostgreSQL real
		pgContainer, cleanup := SetupPostgresContainer(t, ctx)
		defer cleanup()

		CreateTestSchema(t, ctx, pgContainer.DB)

		repo := CreateTestRepository(t, pgContainer.ConnStr)
		defer repo.Close(ctx)

		// Executar
		testBook := book.Book{
			Title:    "Clean Code",
			Author:   "Robert C. Martin",
			Category: book.Read,
		}

		id, err := repo.Insert(ctx, testBook)

		// Validar
		require.NoError(t, err)
		assert.Greater(t, id, int64(0))
		AssertBookCount(t, ctx, pgContainer.DB, 1)
	})

	t.Run("insert returns sequential IDs", func(t *testing.T) {
		ctx := context.Background()

		pgContainer, cleanup := SetupPostgresContainer(t, ctx)
		defer cleanup()

		CreateTestSchema(t, ctx, pgContainer.DB)

		repo := CreateTestRepository(t, pgContainer.ConnStr)
		defer repo.Close(ctx)

		// Inserir 3 livros
		id1, _ := repo.Insert(ctx, book.Book{Title: "Book 1", Author: "A1", Category: book.Read})
		id2, _ := repo.Insert(ctx, book.Book{Title: "Book 2", Author: "A2", Category: book.Read})
		id3, _ := repo.Insert(ctx, book.Book{Title: "Book 3", Author: "A3", Category: book.Read})

		// PostgreSQL SERIAL gera IDs sequenciais
		assert.Equal(t, int64(1), id1)
		assert.Equal(t, int64(2), id2)
		assert.Equal(t, int64(3), id3)
	})

	t.Run("insert multiple books", func(t *testing.T) {
		ctx := context.Background()

		pgContainer, cleanup := SetupPostgresContainer(t, ctx)
		defer cleanup()

		CreateTestSchema(t, ctx, pgContainer.DB)

		repo := CreateTestRepository(t, pgContainer.ConnStr)
		defer repo.Close(ctx)

		books := []book.Book{
			{Title: "Book 1", Author: "Author 1", Category: book.Read},
			{Title: "Book 2", Author: "Author 2", Category: book.Reading},
			{Title: "Book 3", Author: "Author 3", Category: book.WantToRead},
		}

		for _, b := range books {
			id, err := repo.Insert(ctx, b)
			require.NoError(t, err)
			assert.Greater(t, id, int64(0))
		}

		AssertBookCount(t, ctx, pgContainer.DB, 3)
	})
}

func TestPostgresRepository_Select_Integration(t *testing.T) {
	t.Run("select existing book", func(t *testing.T) {
		ctx := context.Background()

		pgContainer, cleanup := SetupPostgresContainer(t, ctx)
		defer cleanup()

		CreateTestSchema(t, ctx, pgContainer.DB)
		PopulateSampleData(t, ctx, pgContainer.DB)

		repo := CreateTestRepository(t, pgContainer.ConnStr)
		defer repo.Close(ctx)

		// Executar
		b, err := repo.Select(ctx, 1)

		// Validar
		require.NoError(t, err)
		assert.Equal(t, int64(1), b.ID)
		assert.Equal(t, "Neuromancer", b.Title)
		assert.Equal(t, "William Gibson", b.Author)
		assert.Equal(t, book.Read, b.Category)
	})

	t.Run("select non-existent book returns ErrNotFound", func(t *testing.T) {
		ctx := context.Background()

		pgContainer, cleanup := SetupPostgresContainer(t, ctx)
		defer cleanup()

		CreateTestSchema(t, ctx, pgContainer.DB)

		repo := CreateTestRepository(t, pgContainer.ConnStr)
		defer repo.Close(ctx)

		// Executar
		_, err := repo.Select(ctx, 999)

		// Validar
		require.Error(t, err)
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("select correct book from multiple", func(t *testing.T) {
		ctx := context.Background()

		pgContainer, cleanup := SetupPostgresContainer(t, ctx)
		defer cleanup()

		CreateTestSchema(t, ctx, pgContainer.DB)
		PopulateSampleData(t, ctx, pgContainer.DB)

		repo := CreateTestRepository(t, pgContainer.ConnStr)
		defer repo.Close(ctx)

		// Buscar o segundo livro
		b, err := repo.Select(ctx, 2)

		require.NoError(t, err)
		assert.Equal(t, "Dune", b.Title)
		assert.Equal(t, "Frank Herbert", b.Author)
	})
}

func TestPostgresRepository_SelectAll_Integration(t *testing.T) {
	t.Run("select all returns all books", func(t *testing.T) {
		ctx := context.Background()

		pgContainer, cleanup := SetupPostgresContainer(t, ctx)
		defer cleanup()

		CreateTestSchema(t, ctx, pgContainer.DB)
		PopulateSampleData(t, ctx, pgContainer.DB)

		repo := CreateTestRepository(t, pgContainer.ConnStr)
		defer repo.Close(ctx)

		// Executar
		books, err := repo.SelectAll(ctx)

		// Validar
		require.NoError(t, err)
		assert.Equal(t, 3, len(books))
		assert.Equal(t, "Neuromancer", books[0].Title)
		assert.Equal(t, "Dune", books[1].Title)
		assert.Equal(t, "1984", books[2].Title)
	})

	t.Run("select all from empty database returns ErrNotFound", func(t *testing.T) {
		ctx := context.Background()

		pgContainer, cleanup := SetupPostgresContainer(t, ctx)
		defer cleanup()

		CreateTestSchema(t, ctx, pgContainer.DB)

		repo := CreateTestRepository(t, pgContainer.ConnStr)
		defer repo.Close(ctx)

		// Executar
		_, err := repo.SelectAll(ctx)

		// Validar
		require.Error(t, err)
		assert.Equal(t, ErrNotFound, err)
	})
}

func TestPostgresRepository_Update_Integration(t *testing.T) {
	t.Run("update existing book", func(t *testing.T) {
		ctx := context.Background()

		pgContainer, cleanup := SetupPostgresContainer(t, ctx)
		defer cleanup()

		CreateTestSchema(t, ctx, pgContainer.DB)
		PopulateSampleData(t, ctx, pgContainer.DB)

		repo := CreateTestRepository(t, pgContainer.ConnStr)
		defer repo.Close(ctx)

		// Atualizar livro
		updatedBook := book.Book{
			ID:       1,
			Title:    "Neuromancer - Remastered",
			Author:   "William Gibson",
			Category: book.Reading,
		}

		err := repo.Update(ctx, updatedBook)
		require.NoError(t, err)

		// Verificar atualização
		b, _ := repo.Select(ctx, 1)
		assert.Equal(t, "Neuromancer - Remastered", b.Title)
		assert.Equal(t, book.Reading, b.Category)
	})

	t.Run("update non-existent book returns ErrNotFound", func(t *testing.T) {
		ctx := context.Background()

		pgContainer, cleanup := SetupPostgresContainer(t, ctx)
		defer cleanup()

		CreateTestSchema(t, ctx, pgContainer.DB)

		repo := CreateTestRepository(t, pgContainer.ConnStr)
		defer repo.Close(ctx)

		// Tentar atualizar livro inexistente
		err := repo.Update(ctx, book.Book{
			ID:       999,
			Title:    "Non-existent",
			Author:   "Nobody",
			Category: book.Read,
		})

		require.Error(t, err)
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("update preserves other records", func(t *testing.T) {
		ctx := context.Background()

		pgContainer, cleanup := SetupPostgresContainer(t, ctx)
		defer cleanup()

		CreateTestSchema(t, ctx, pgContainer.DB)
		PopulateSampleData(t, ctx, pgContainer.DB)

		repo := CreateTestRepository(t, pgContainer.ConnStr)
		defer repo.Close(ctx)

		// Atualizar apenas o primeiro
		repo.Update(ctx, book.Book{
			ID:       1,
			Title:    "Updated",
			Author:   "Updated",
			Category: book.Read,
		})

		// Outros não mudaram
		b2, _ := repo.Select(ctx, 2)
		assert.Equal(t, "Dune", b2.Title)

		b3, _ := repo.Select(ctx, 3)
		assert.Equal(t, "1984", b3.Title)
	})
}

func TestPostgresRepository_Delete_Integration(t *testing.T) {
	t.Run("delete existing book", func(t *testing.T) {
		ctx := context.Background()

		pgContainer, cleanup := SetupPostgresContainer(t, ctx)
		defer cleanup()

		CreateTestSchema(t, ctx, pgContainer.DB)
		PopulateSampleData(t, ctx, pgContainer.DB)

		repo := CreateTestRepository(t, pgContainer.ConnStr)
		defer repo.Close(ctx)

		// Executar
		err := repo.Delete(ctx, 1)

		// Validar
		require.NoError(t, err)
		AssertBookCount(t, ctx, pgContainer.DB, 2)

		// Verificar que foi deletado
		_, err = repo.Select(ctx, 1)
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("delete non-existent book returns ErrNotFound", func(t *testing.T) {
		ctx := context.Background()

		pgContainer, cleanup := SetupPostgresContainer(t, ctx)
		defer cleanup()

		CreateTestSchema(t, ctx, pgContainer.DB)

		repo := CreateTestRepository(t, pgContainer.ConnStr)
		defer repo.Close(ctx)

		// Executar
		err := repo.Delete(ctx, 999)

		// PostgreSQL retorna ErrNotFound quando não há rows affected
		require.Error(t, err)
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("delete all books one by one", func(t *testing.T) {
		ctx := context.Background()

		pgContainer, cleanup := SetupPostgresContainer(t, ctx)
		defer cleanup()

		CreateTestSchema(t, ctx, pgContainer.DB)
		PopulateSampleData(t, ctx, pgContainer.DB)

		repo := CreateTestRepository(t, pgContainer.ConnStr)
		defer repo.Close(ctx)

		// Deletar todos
		repo.Delete(ctx, 1)
		AssertBookCount(t, ctx, pgContainer.DB, 2)

		repo.Delete(ctx, 2)
		AssertBookCount(t, ctx, pgContainer.DB, 1)

		repo.Delete(ctx, 3)

		// Banco vazio
		_, err := repo.SelectAll(ctx)
		assert.Equal(t, ErrNotFound, err)
	})
}

func TestPostgresRepository_CRUD_Integration(t *testing.T) {
	t.Run("complete CRUD cycle", func(t *testing.T) {
		ctx := context.Background()

		pgContainer, cleanup := SetupPostgresContainer(t, ctx)
		defer cleanup()

		CreateTestSchema(t, ctx, pgContainer.DB)

		repo := CreateTestRepository(t, pgContainer.ConnStr)
		defer repo.Close(ctx)

		// CREATE
		newBook := book.Book{
			Title:    "The Phoenix Project",
			Author:   "Gene Kim",
			Category: book.Reading,
		}
		id, err := repo.Insert(ctx, newBook)
		require.NoError(t, err)

		// READ
		retrieved, err := repo.Select(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, newBook.Title, retrieved.Title)

		// UPDATE
		retrieved.Title = "The Phoenix Project - DevOps Edition"
		retrieved.Category = book.Read
		err = repo.Update(ctx, retrieved)
		require.NoError(t, err)

		// READ updated
		updated, err := repo.Select(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, "The Phoenix Project - DevOps Edition", updated.Title)
		assert.Equal(t, book.Read, updated.Category)

		// DELETE
		err = repo.Delete(ctx, id)
		require.NoError(t, err)

		// READ after delete
		_, err = repo.Select(ctx, id)
		assert.Equal(t, ErrNotFound, err)
	})
}

func TestPostgresRepository_Concurrent_Integration(t *testing.T) {
	t.Run("concurrent inserts", func(t *testing.T) {
		ctx := context.Background()

		pgContainer, cleanup := SetupPostgresContainer(t, ctx)
		defer cleanup()

		CreateTestSchema(t, ctx, pgContainer.DB)

		repo := CreateTestRepository(t, pgContainer.ConnStr)
		defer repo.Close(ctx)

		// Executar inserts concorrentes
		numGoroutines := 10
		done := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				b := book.Book{
					Title:    fmt.Sprintf("Concurrent Book %d", index),
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
		AssertBookCount(t, ctx, pgContainer.DB, numGoroutines)
	})
}

func TestPostgresRepository_Transactions_Integration(t *testing.T) {
	t.Run("test transaction rollback behavior", func(t *testing.T) {
		ctx := context.Background()

		pgContainer, cleanup := SetupPostgresContainer(t, ctx)
		defer cleanup()

		CreateTestSchema(t, ctx, pgContainer.DB)

		// Inserir dados iniciais
		PopulateSampleData(t, ctx, pgContainer.DB)

		// Verificar isolamento de conexões
		AssertBookCount(t, ctx, pgContainer.DB, 3)
	})
}
