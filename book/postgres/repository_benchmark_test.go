//go:build integration

package postgres

import (
	"context"
	"testing"

	"github.com/eminetto/post-turso/book"
)

/*
Benchmarks para PostgreSQL Repository

Compara performance das operações CRUD contra PostgreSQL real.

Execute com: go test -tags=integration -bench=. -benchmem ./book/postgres/

Para melhorar a velocidade dos benchmarks, habilite o reuso de containers:
  export TESTCONTAINERS_REUSE_ENABLE=true

Exemplo de saída:
  BenchmarkInsert_Postgres-8    1000  1234567 ns/op  1024 B/op  10 allocs/op

Legenda:
  -8: Número de CPUs
  1000: Número de iterações
  1234567 ns/op: Nanosegundos por operação
  1024 B/op: Bytes alocados por operação
  10 allocs/op: Alocações por operação

NOTA: Cada benchmark cria um novo container PostgreSQL. A medição começa após
      a inicialização (b.ResetTimer) para não incluir o overhead do container.

OTIMIZAÇÃO DE PERFORMANCE:
Para compartilhar um container entre todos os testes de um arquivo, refatore para:

  func TestPostgresRepositorySuite(t *testing.T) {
      pgContainer, cleanup := SetupPostgresContainer(t, context.Background())
      defer cleanup()
      CreateTestSchema(...)
      repo := CreateTestRepository(...)

      t.Run("Insert", func(t *testing.T) { /* ... */ })
      t.Run("Select", func(t *testing.T) { /* ... */ })
      // ... outros testes
  }

Isto reduz o tempo de teste de ~31s para ~5s sem precisar de TESTCONTAINERS_REUSE_ENABLE.
*/

func BenchmarkInsert_Postgres(b *testing.B) {
	ctx := context.Background()

	// Setup
	pgContainer, cleanup := SetupPostgresContainer(nil, ctx)
	defer cleanup()

	CreateTestSchema(nil, ctx, pgContainer.DB)
	repo := CreateTestRepository(nil, pgContainer.ConnStr)
	defer repo.Close(ctx)

	testBook := book.Book{
		Title:    "Benchmark Book",
		Author:   "Benchmark Author",
		Category: book.Read,
	}

	// Benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repo.Insert(ctx, testBook)
		if err != nil {
			b.Fatalf("Insert failed: %v", err)
		}
	}
}

func BenchmarkSelect_Postgres(b *testing.B) {
	ctx := context.Background()

	// Setup
	pgContainer, cleanup := SetupPostgresContainer(nil, ctx)
	defer cleanup()

	CreateTestSchema(nil, ctx, pgContainer.DB)
	PopulateSampleData(nil, ctx, pgContainer.DB)

	repo := CreateTestRepository(nil, pgContainer.ConnStr)
	defer repo.Close(ctx)

	// Benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repo.Select(ctx, 1)
		if err != nil {
			b.Fatalf("Select failed: %v", err)
		}
	}
}

func BenchmarkSelectAll_Postgres(b *testing.B) {
	ctx := context.Background()

	// Setup
	pgContainer, cleanup := SetupPostgresContainer(nil, ctx)
	defer cleanup()

	CreateTestSchema(nil, ctx, pgContainer.DB)
	PopulateSampleData(nil, ctx, pgContainer.DB)

	repo := CreateTestRepository(nil, pgContainer.ConnStr)
	defer repo.Close(ctx)

	// Benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repo.SelectAll(ctx)
		if err != nil {
			b.Fatalf("SelectAll failed: %v", err)
		}
	}
}

func BenchmarkUpdate_Postgres(b *testing.B) {
	ctx := context.Background()

	// Setup
	pgContainer, cleanup := SetupPostgresContainer(nil, ctx)
	defer cleanup()

	CreateTestSchema(nil, ctx, pgContainer.DB)
	PopulateSampleData(nil, ctx, pgContainer.DB)

	repo := CreateTestRepository(nil, pgContainer.ConnStr)
	defer repo.Close(ctx)

	testBook := book.Book{
		ID:       1,
		Title:    "Updated Title",
		Author:   "Updated Author",
		Category: book.Reading,
	}

	// Benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := repo.Update(ctx, testBook)
		if err != nil {
			b.Fatalf("Update failed: %v", err)
		}
	}
}

func BenchmarkDelete_Postgres(b *testing.B) {
	ctx := context.Background()

	// Setup
	pgContainer, cleanup := SetupPostgresContainer(nil, ctx)
	defer cleanup()

	CreateTestSchema(nil, ctx, pgContainer.DB)

	repo := CreateTestRepository(nil, pgContainer.ConnStr)
	defer repo.Close(ctx)

	// Pré-popular com livros para deletar
	bookIDs := make([]int64, b.N)
	for i := 0; i < b.N; i++ {
		id, err := repo.Insert(ctx, book.Book{
			Title:    "Book to Delete",
			Author:   "Author",
			Category: book.Read,
		})
		if err != nil {
			b.Fatalf("Setup failed: %v", err)
		}
		bookIDs[i] = id
	}

	// Benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := repo.Delete(ctx, bookIDs[i])
		if err != nil {
			b.Fatalf("Delete failed: %v", err)
		}
	}
}

// BenchmarkCRUD_Cycle testa um ciclo CRUD completo
func BenchmarkCRUD_Cycle(b *testing.B) {
	ctx := context.Background()

	// Setup
	pgContainer, cleanup := SetupPostgresContainer(nil, ctx)
	defer cleanup()

	CreateTestSchema(nil, ctx, pgContainer.DB)
	repo := CreateTestRepository(nil, pgContainer.ConnStr)
	defer repo.Close(ctx)

	// Benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create
		id, err := repo.Insert(ctx, book.Book{
			Title:    "CRUD Test",
			Author:   "Author",
			Category: book.Read,
		})
		if err != nil {
			b.Fatalf("Insert failed: %v", err)
		}

		// Read
		_, err = repo.Select(ctx, id)
		if err != nil {
			b.Fatalf("Select failed: %v", err)
		}

		// Update
		err = repo.Update(ctx, book.Book{
			ID:       id,
			Title:    "Updated",
			Author:   "Author",
			Category: book.Reading,
		})
		if err != nil {
			b.Fatalf("Update failed: %v", err)
		}

		// Delete
		err = repo.Delete(ctx, id)
		if err != nil {
			b.Fatalf("Delete failed: %v", err)
		}
	}
}
