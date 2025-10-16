package main

import (
	"context"
	"fmt"

	"github.com/eminetto/post-turso/book"
	"github.com/eminetto/post-turso/book/postgres"
	"github.com/eminetto/post-turso/config"
)

/*
CLI PostgreSQL - Exemplo de uso do repositório PostgreSQL

Este CLI demonstra:
- Como usar Config para carregar variáveis PostgreSQL
- Como conectar ao PostgreSQL
- Como usar o postgres.Repository
- Como usar o book.Service
- Como executar operações CRUD

Execute com:
  go run cmd/cli-postgres/main.go

Ou com Makefile:
  make cli-postgres

Certifique-se de que:
1. PostgreSQL está rodando (docker-compose up)
2. .env está configurado com POSTGRES_* variables
3. Migrations foram aplicadas (make migrate-up)
*/

func main() {
	// 1. Carregar configuração
	cfg, err := config.GetConfig()
	if err != nil {
		fmt.Printf("❌ Error loading config: %v\n", err)
		return
	}

	// 1a. Validar configuração PostgreSQL
	if err := cfg.ValidatePostgres(); err != nil {
		fmt.Printf("❌ Configuration validation failed: %v\n", err)
		return
	}

	ctx := context.Background()

	// 2. Conectar ao PostgreSQL
	connStr := cfg.PostgresConnectionString()
	fmt.Printf("🔗 Connecting to PostgreSQL at %s:%s...\n", cfg.PostgresHost, cfg.PostgresPort)

	// Use configurable pool settings from config, with defaults if not set
	repo, err := postgres.NewRepositoryWithPoolConfig(
		connStr,
		cfg.GetPostgresMaxOpenConns(),
		cfg.GetPostgresMaxIdleConns(),
		cfg.GetPostgresConnMaxLifeMinutes(),
	)
	if err != nil {
		fmt.Printf("❌ Error connecting to PostgreSQL: %v\n", err)
		return
	}
	defer repo.Close(ctx)
	fmt.Println("✅ Connected to PostgreSQL!")

	// 3. Criar service
	s := book.NewService(repo)

	// 4. Criar um livro de exemplo
	fmt.Println("\n📝 Creating a new book...")
	newBook, err := s.Create(ctx, "The Pragmatic Programmer", "Andy Hunt & Dave Thomas", book.WantToRead)
	if err != nil {
		fmt.Printf("❌ Error creating book: %v\n", err)
		return
	}

	fmt.Println("✅ Book created successfully!")
	fmt.Printf("   ID:       %d\n", newBook.ID)
	fmt.Printf("   Title:    %s\n", newBook.Title)
	fmt.Printf("   Author:   %s\n", newBook.Author)
	fmt.Printf("   Category: %s\n", newBook.Category.String())

	// 5. Listar todos os livros
	fmt.Println("\n📚 All books in database:")
	books, err := s.List(ctx)
	if err != nil {
		fmt.Printf("❌ Error listing books: %v\n", err)
		return
	}

	if len(books) == 0 {
		fmt.Println("   (no books yet)")
	} else {
		for _, b := range books {
			fmt.Printf("   [%d] %s by %s (%s)\n", b.ID, b.Title, b.Author, b.Category.String())
		}
	}

	// 6. Buscar um livro específico
	if len(books) > 0 {
		firstBookID := books[0].ID
		fmt.Printf("\n🔍 Fetching book with ID %d...\n", firstBookID)

		retrieved, err := s.Get(ctx, firstBookID)
		if err != nil {
			fmt.Printf("❌ Error retrieving book: %v\n", err)
			return
		}

		fmt.Printf("✅ Found: %s by %s\n", retrieved.Title, retrieved.Author)

		// 7. Atualizar o livro
		fmt.Printf("\n✏️  Updating book %d...\n", firstBookID)

		retrieved.Category = book.Reading
		err = s.Update(ctx, retrieved.ID, retrieved.Title, retrieved.Author, retrieved.Category)
		if err != nil {
			fmt.Printf("❌ Error updating book: %v\n", err)
			return
		}

		fmt.Printf("✅ Book updated! New category: %s\n", retrieved.Category.String())

		// 8. Deletar o livro
		fmt.Printf("\n🗑️  Deleting book %d...\n", firstBookID)

		err = s.Delete(ctx, firstBookID)
		if err != nil {
			fmt.Printf("❌ Error deleting book: %v\n", err)
			return
		}

		fmt.Println("✅ Book deleted!")
	}

	// 9. Listar novamente
	fmt.Println("\n📚 Books after deletion:")
	books, err = s.List(ctx)
	if err != nil {
		// Esperado se não houver livros
		fmt.Println("   (no books)")
	} else {
		for _, b := range books {
			fmt.Printf("   [%d] %s by %s\n", b.ID, b.Title, b.Author)
		}
	}

	fmt.Println("\n✅ CLI completed successfully!")
}
