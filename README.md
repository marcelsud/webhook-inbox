# Talk about The Go Way 🎯

Um projeto educacional que demonstra as **melhores práticas de desenvolvimento em Go** através de uma aplicação completa de gerenciamento de livros com API REST e CLI.

## 📚 Visão Geral

Este projeto implementa um sistema CRUD de livros que exemplifica os princípios e padrões considerados "o caminho do Go" (The Go Way). Ele integra-se com **Turso** (banco de dados distribuído baseado em SQLite) e demonstra como estruturar uma aplicação Go seguindo boas práticas de arquitetura e design.

> **Objetivo**: Servir como referência educacional para desenvolvedores que desejam aprender os padrões recomendados na comunidade Go.

---

## 📑 Índice

- [Visão Geral](#-visão-geral)
- [Conceitos "Go Way" Demonstrados](#-conceitos-go-way-demonstrados)
- [Arquitetura](#-arquitetura)
- [Múltiplas Implementações de Repository](#-múltiplas-implementações-de-repository)
- [Tecnologias](#-tecnologias)
- [Pré-requisitos](#-pré-requisitos)
- [Quick Start](#-quick-start)
- [Instalação e Configuração Detalhada](#-instalação-e-configuração-detalhada)
- [Uso](#-uso)
- [Testes](#-testes)
- [Troubleshooting](#-troubleshooting)
- [Estrutura dos Testes](#-estrutura-dos-testes)
- [Fluxo da Aplicação](#-fluxo-da-aplicação)
- [Principais Padrões Implementados](#-principais-padrões-implementados)
- [Decisões de Design](#-decisões-de-design)
- [Comentários no Código](#-comentários-no-código)
- [Referências e Recursos](#-referências-e-recursos)
- [Schema do Banco de Dados](#-schema-do-banco-de-dados)
- [Próximas Melhorias](#-próximas-melhorias)

---

## 🎓 Conceitos "Go Way" Demonstrados

Este projeto exemplifica os seguintes princípios e padrões:

### 1. **Interfaces Pequenas e Focadas**
```go
type Reader interface {
    Select(ctx context.Context, id int64) (Book, error)
    SelectAll(ctx context.Context) ([]Book, error)
}

type Writer interface {
    Insert(ctx context.Context, book Book) (int64, error)
    Update(ctx context.Context, book Book) error
    Delete(ctx context.Context, id int64) error
}
```
- Interfaces devem ser pequenas e definir **comportamento** (não coisas)
- Permitem composição e flexibilidade

### 2. **Composição de Interfaces**
```go
type Repository interface {
    Reader
    Writer
    Close(ctx context.Context) error
}
```
- Reutilizar interfaces pequenas para criar abstrações mais complexas
- Melhor que herança pesada

### 3. **Value vs Pointer Semantics**
- **Value semantics** para dados (`Book struct`): Sem tags, sem mutabilidade
- **Pointer semantics** para APIs (`Service struct`): Métodos podem ter efeitos colaterais
- **Value semantics** para tipos primitivos (`Category int`)

### 4. **Pacotes que "Fornecem" vs "Contêm"**
- Pacotes devem **fornecer algo** útil (ex: um serviço, abstrações)
- Evitar pacotes auxiliares genéricos (models, utils, helpers) que "contêm" coisas
- Melhora a organização e reduz problemas de dependências

### 5. **Context como Primeiro Parâmetro**
```go
func (s *Service) Create(ctx context.Context, title, author string, category Category) (Book, error)
```
- Context é sempre o primeiro parâmetro em funções que fazem I/O
- Permite cancelamento, timeout e valores compartilhados

### 6. **Error Handling Apropriado**
```go
if err != nil {
    return Book{}, fmt.Errorf("inserting book: %w", err)
}
```
- Usar `%w` para wrapping de erros (Go 1.13+)
- Preservar a cadeia de erros (error chain) para análise com `errors.Is()` e `errors.As()`
- Adicionar contexto sobre o que falhou

### 7. **Separação de DTOs por Camada**
```go
// Camada de domínio
type Book struct {
    ID       int64
    Title    string
    Author   string
    Category Category
}

// Camada HTTP
type bookRequest struct {
    Title    string `json:"title"`
    Author   string `json:"author"`
    Category string `json:"category"`
}
```
- Diferentes representações para diferentes camadas
- Não expor estruturas internas

### 8. **Tipos Customizados para Type Safety**
```go
type Category int

const (
    WantToRead Category = iota + 1
    Reading
    Read
)
```
- Aproveitar o compilador para encontrar erros em tempo de compilação
- Evitar strings ou integers "mágicos"

### 9. **Encapsulamento com `internal/`**
```
internal/
  ├── http/
  └── user/
```
- Arquivos em `internal/` só podem ser importados por pacotes ancestrais
- Cria barreira de acesso efetiva
- Protege implementação interna

### 10. **Testes com Mocks e Subtestes**
```go
t.Run("success", func(t *testing.T) {
    repo := mocks.NewRepository(t)
    repo.On("Insert", ctx, b).Return(int64(1), nil)
    // ...
})
```
- Usar subtestes (`t.Run`) para organizar casos de teste
- Gerar mocks automaticamente com Mockery
- Testar comportamento, não implementação

### 11. **Graceful Shutdown**
- Tratar sinais do SO (SIGHUP, SIGINT, SIGTERM, SIGQUIT)
- Dar tempo para operações em andamento completarem
- Fechar recursos apropriadamente

### 12. **Logging Estruturado**
- Usar structured logging (JSON) ao invés de printf
- Facilita parsing e análise em produção
- Integrado com Chi httplog

---

## 🏗️ Arquitetura

O projeto segue **Clean Architecture** (também conhecida como Hexagonal Architecture):

```
┌─────────────────────────────────────────┐
│     HTTP API (cmd/api)                  │
│     CLI (cmd/cli)                       │
└──────────────────┬──────────────────────┘
                   │
┌──────────────────▼──────────────────────┐
│     Business Logic Layer                │
│  (book/service.go, book/UseCase)       │
└──────────────────┬──────────────────────┘
                   │
┌──────────────────▼──────────────────────┐
│     Repository Interface                │
│  (book/repository.go interfaces)       │
└──────────────────┬──────────────────────┘
                   │
┌──────────────────▼──────────────────────┐
│     Infrastructure                      │
│  (book/turso/repository.go)            │
│  (Turso/LibSQL Database)               │
└─────────────────────────────────────────┘
```

### Fluxo de Dependências
As importações devem ser apenas **para baixo** (verticais):
- `cmd/api` e `cmd/cli` importam `book`
- `book` importa `book/turso`
- Nunca para cima ou horizontalmente

### Estrutura de Diretórios

```
.
├── cmd/                          # Aplicações executáveis
│   ├── api/main.go              # Servidor HTTP REST
│   └── cli/main.go              # Interface CLI
├── book/                        # Domínio de negócio
│   ├── book.go                  # Entidade Book
│   ├── category.go              # Tipo Category e sua lógica
│   ├── service.go               # Casos de uso (UseCase interface)
│   ├── repository.go            # Interfaces Reader/Writer/Repository
│   ├── service_test.go          # Testes da camada de negócio
│   ├── turso/                   # Implementação do repositório SQLite/Turso
│   │   ├── repository.go        # Repository com Turso/SQLite
│   │   ├── repository_test.go   # Testes unitários (SQLite local)
│   │   ├── repository_integration_test.go  # Testes de integração
│   │   └── testhelpers_test.go  # Helpers para testes
│   ├── postgres/                # Implementação do repositório PostgreSQL
│   │   ├── repository.go        # Repository com PostgreSQL
│   │   ├── repository_integration_test.go  # Testes com testcontainers
│   │   └── testhelpers_test.go  # Helpers para containers
│   └── mocks/                   # Mocks gerados automaticamente
├── internal/                    # Código protegido (não importável externamente)
│   ├── http/chi/                # Handlers HTTP
│   │   ├── handlers.go          # Roteamento
│   │   ├── books.go             # Handlers de livros
│   │   └── books_test.go        # Testes dos handlers
│   └── user/                    # Futuro: gerenciamento de usuários
├── config/                      # Configuração da aplicação
│   └── config.go                # Gerenciamento de variáveis de ambiente
├── auth/                        # Autenticação (futuro)
│   └── auth.go                  # Funções de autenticação
├── go.mod                       # Módulo Go e dependências
├── go.sum                       # Checksum das dependências
├── Makefile                     # Comandos úteis
└── README.md                    # Este arquivo
```

---

## 🗄️ Múltiplas Implementações de Repository

O projeto demonstra **duas implementações** da mesma interface `Repository`:

### SQLite/Turso (`book/turso/`)
- **Uso**: Produção (Turso embedded replica)
- **Testes**: SQLite em memória (`:memory:`)
- **Vantagens**: Embarcado, sem dependências externas
- **Testcontainers**: ❌ Não necessário (SQLite é embarcado)

### PostgreSQL (`book/postgres/`)
- **Uso**: Exemplo educacional
- **Testes**: PostgreSQL em container Docker real
- **Vantagens**: Banco completo, suporta concorrência avançada
- **Testcontainers**: ✅ **USO REAL** demonstrado

### Comparação: SQLite vs PostgreSQL

| Aspecto | SQLite (Turso) | PostgreSQL |
|---------|---------------|------------|
| **Deployment** | Embarcado no binário | Servidor separado |
| **Testes** | Arquivo/memória local | Container Docker |
| **Testcontainers** | Desnecessário | Necessário e útil |
| **Placeholders** | `?` | `$1, $2, $3` |
| **Auto Increment** | `AUTOINCREMENT` | `SERIAL` |
| **RETURNING** | Não suportado | `INSERT ... RETURNING id` |
| **Concorrência** | Limitada | Excelente |
| **Setup** | Zero config | Requer server/container |

### Diferenças de Sintaxe SQL

**SQLite:**
```sql
CREATE TABLE books (
  ID INTEGER PRIMARY KEY AUTOINCREMENT,  -- AUTOINCREMENT
  title TEXT
);

INSERT INTO books (title) VALUES (?);    -- ? placeholder
```

**PostgreSQL:**
```sql
CREATE TABLE books (
  id SERIAL PRIMARY KEY,                 -- SERIAL
  title TEXT
);

INSERT INTO books (title) VALUES ($1)    -- $1 placeholder
RETURNING id;                            -- RETURNING clause
```

### Por que duas implementações?

1. **Educacional**: Demonstra adapter pattern
2. **Interface única**: Ambas implementam `book.Repository`
3. **Testcontainers real**: PostgreSQL demonstra uso correto
4. **Flexibilidade**: Trocar banco sem mudar service layer

```go
// Mesma interface, diferentes implementações
var repo book.Repository

// Opção 1: SQLite/Turso
repo, _ = turso.NewRepository(dbName, url, token)

// Opção 2: PostgreSQL
repo, _ = postgres.NewRepository(connStr)

// Service não sabe qual banco está usando!
service := book.NewService(repo)
```

---

## 🔧 Tecnologias

- **Linguagem**: Go 1.24.0
- **Banco de Dados**: Turso/LibSQL (embedded replica)
- **HTTP Router**: Chi v5.2.1
- **Configuração**: Viper v1.20.0
- **Testes Unitários**: Testify v1.10.0
- **Testes de Integração**: Testcontainers v0.39.0
- **Mock Generation**: Mockery v2.53.3
- **Logging**: Chi httplog v0.3.2

---

## 📋 Pré-requisitos

- **Go 1.24.0** ou superior
- **Conta Turso**: Criar em https://turso.tech (gratuito)
- **Turso CLI**: Para gerenciar o banco de dados

---

## ⚡ Quick Start

Para começar rapidamente:

```bash
# 1. Clone e instale dependências
git clone https://github.com/eminetto/post-turso.git
cd post-turso
go mod download

# 2. Configure Turso
curl -sSfL https://get.tur.so/install.sh | bash
turso auth login
turso db create books-db

# 3. Crie o schema
turso db shell books-db "CREATE TABLE IF NOT EXISTS books (
  ID INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL,
  author TEXT NOT NULL,
  category INTEGER NOT NULL
);"

# 4. Configure variáveis de ambiente
cat > .env << EOF
PORT = "8080"
DBNAME = "local.db"
TURSO_DATABASE_URL = "$(turso db show books-db --url)"
TURSO_AUTH_TOKEN = "$(turso db tokens create books-db)"
EOF

# 5. Execute a API
go run cmd/api/main.go
```

**Teste a API**:
```bash
curl http://localhost:8080/v1/books
```

---

## 🚀 Instalação e Configuração Detalhada

### 1. Clonar o Repositório
```bash
git clone https://github.com/eminetto/post-turso.git
cd post-turso
```

### 2. Instalar Dependências
```bash
go mod download
```

### 3. Criar Banco de Dados no Turso

**Instalar Turso CLI**:
```bash
curl -sSfL https://get.tur.so/install.sh | bash
```

**Autenticar e criar banco**:
```bash
# Autenticar
turso auth login

# Criar banco
turso db create books-db

# Obter credenciais
turso db show books-db --url
turso db tokens create books-db
```

**Criar schema (tabela)**:
```bash
turso db shell books-db "CREATE TABLE IF NOT EXISTS books (
  ID INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL,
  author TEXT NOT NULL,
  category INTEGER NOT NULL
);"
```

### 4. Configurar Variáveis de Ambiente
Criar arquivo `.env` na raiz do projeto:

```toml
PORT = "8080"
DBNAME = "local.db"
TURSO_DATABASE_URL = "libsql://seu-db.turso.io"
TURSO_AUTH_TOKEN = "seu-token-de-autenticacao"
```

Substitua `seu-db.turso.io` pela URL obtida no passo anterior e `seu-token-de-autenticacao` pelo token gerado.

---

## 📖 Uso

### Executar o Servidor HTTP

```bash
go run cmd/api/main.go
```

O servidor estará disponível em `http://localhost:8080`

### Executar o CLI

```bash
go run cmd/cli/main.go
```

### Exemplos de API

#### 1. Listar Todos os Livros
```bash
curl http://localhost:8080/v1/books
```

**Resposta**:
```json
[
  {
    "id": 1,
    "title": "Neuromancer",
    "author": "William Gibson",
    "category": "Read"
  }
]
```

#### 2. Obter um Livro por ID
```bash
curl http://localhost:8080/v1/books/1
```

#### 3. Criar um Novo Livro
```bash
curl -X POST http://localhost:8080/v1/books \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Padrões de Arquitetura de Aplicações Distribuídas",
    "author": "Chris Richardson",
    "category": "Want to Read"
  }'
```

**Categorias válidas**:
- `"Want to Read"`
- `"Reading"`
- `"Read"`

#### 4. Atualizar um Livro
```bash
curl -X PUT http://localhost:8080/v1/books/1 \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Neuromancer - 2nd Edition",
    "author": "William Gibson",
    "category": "Reading"
  }'
```

**Resposta**:
```
Status: 200 OK
```

#### 5. Deletar um Livro
```bash
curl -X DELETE http://localhost:8080/v1/books/1
```

**Resposta**:
```
Status: 200 OK
```

### Códigos de Status HTTP

| Endpoint | Método | Sucesso | Erro |
|----------|--------|---------|------|
| `/v1/books` | GET | `200 OK` | `500` Internal Server Error |
| `/v1/books/{id}` | GET | `200 OK` | `400` Bad Request (id inválido)<br>`500` Internal Server Error |
| `/v1/books` | POST | `201 Created` | `400` Bad Request (JSON inválido)<br>`500` Internal Server Error |
| `/v1/books/{id}` | PUT | `200 OK` | `400` Bad Request (id/JSON inválido)<br>`500` Internal Server Error |
| `/v1/books/{id}` | DELETE | `200 OK` | `400` Bad Request (id inválido)<br>`500` Internal Server Error |

### Exemplos de Respostas de Erro

#### Erro 400: ID Inválido
```bash
curl http://localhost:8080/v1/books/abc
```

**Resposta**:
```
strconv.Atoi: parsing "abc": invalid syntax
```
**Status**: `400 Bad Request`

#### Erro 400: JSON Inválido
```bash
curl -X POST http://localhost:8080/v1/books \
  -H "Content-Type: application/json" \
  -d '{ invalid json }'
```

**Resposta**:
```
invalid character 'i' looking for beginning of object key string
```
**Status**: `400 Bad Request`

#### Erro 500: Livro Não Encontrado
```bash
curl http://localhost:8080/v1/books/999
```

**Resposta**:
```
selecting book: not found
```
**Status**: `500 Internal Server Error`

> **Nota**: Idealmente, "not found" deveria retornar `404`, mas atualmente retorna `500`. Isso é uma oportunidade de melhoria.

---

## 🧪 Testes

O projeto implementa **três tipos de testes**:

### Tipos de Testes

#### 1. Testes Unitários (com Mocks)
- **Arquivos**: `book/service_test.go`, `internal/http/chi/books_test.go`
- **Estratégia**: Mocks com mockery
- **Banco**: Nenhum (mocks)
- **Velocidade**: ⚡⚡⚡ Muito rápido (milissegundos)
- **Execução**: `go test ./...`

#### 2. Testes de Integração - SQLite
- **Arquivo**: `book/turso/repository_integration_test.go`
- **Build tag**: `//go:build integration`
- **Banco**: SQLite em memória (`:memory:`)
- **Velocidade**: ⚡ Rápido (~2-5 segundos)
- **Requisitos**: Nenhum (SQLite é embarcado)
- **Testcontainers**: ❌ Não usado (não necessário)
- **Execução**: `go test -tags=integration ./book/turso/...`

#### 3. Testes de Integração - PostgreSQL (Testcontainers REAL)
- **Arquivo**: `book/postgres/repository_integration_test.go`
- **Build tag**: `//go:build integration`
- **Banco**: PostgreSQL 16 em container Docker REAL
- **Velocidade**: 🐢 Lento (~10-30 segundos na primeira execução)
- **Requisitos**:
  - ✅ Docker rodando
  - ✅ Acesso à internet (download da imagem na primeira vez)
- **Testcontainers**: ✅ **USO REAL DEMONSTRADO**
- **Execução**: `go test -tags=integration ./book/postgres/...`

#### Test Helpers
- **Arquivo**: `book/turso/testhelpers_test.go`
- **Propósito**: Funções reutilizáveis para setup/teardown
- **Referência**: [Test Helpers - Elton Minetto](https://eltonminetto.dev/post/2024-02-15-using-test-helpers/)

### Comandos Disponíveis

**Rodar todos os testes (unitários + integração)**:
```bash
make tests
```

**Rodar apenas testes unitários** (rápido):
```bash
make test-unit
# ou
go test ./...
```

**Rodar apenas testes de integração SQLite**:
```bash
go test -tags=integration ./book/turso/...
```

**Rodar apenas testes de integração PostgreSQL** (requer Docker):
```bash
go test -tags=integration ./book/postgres/...
```

**Rodar TODOS os testes de integração**:
```bash
make test-integration
# ou
go test -tags=integration ./...
```

**Gerar/Atualizar Mocks**:
```bash
make generate-mocks
```

Equivalente a:
```bash
go tool mockery --output book/mocks --dir book --all
```

**Testes com Cobertura**:
```bash
go test -cover ./...
go test -tags=integration -cover ./...
```

**Testes Específicos**:
```bash
go test -run TestCreate ./book           # Testes com padrão no nome
go test -run TestCreate/success ./book   # Subtest específico
go test -tags=integration -run TestRepository ./book/turso
```

**Testes Verbosos**:
```bash
go test -v ./...                         # Mostra todos os testes
go test -tags=integration -v ./book/turso
```

### Como Funciona o Testcontainers (PostgreSQL)

**Fluxo de execução dos testes:**

```go
func TestExample(t *testing.T) {
    ctx := context.Background()

    // 1. Testcontainers sobe container PostgreSQL real
    pgContainer, cleanup := SetupPostgresContainer(t, ctx)
    defer cleanup()  // Cleanup destrói o container

    // 2. Cria schema no banco real
    CreateTestSchema(t, ctx, pgContainer.DB)

    // 3. Testa contra banco real!
    repo := CreateTestRepository(t, pgContainer.ConnStr)
    id, err := repo.Insert(ctx, book)

    // 4. Container é destruído automaticamente
}
```

**O que acontece por baixo dos panos:**

1. 📥 Testcontainers baixa imagem `postgres:16-alpine` (se não tiver)
2. 🐳 Cria e inicia container Docker real
3. ⏳ Aguarda PostgreSQL estar pronto (`database system is ready`)
4. 🔗 Retorna connection string para o container
5. ✅ Testes executam contra PostgreSQL real
6. 🧹 Container é destruído (cleanup automático)

**Vantagens:**
- ✅ Testa SQL real, não mocks
- ✅ Detecta problemas de sintaxe específicos do banco
- ✅ Testa comportamento de transações
- ✅ Isolamento completo (cada teste tem seu container)
- ✅ CI/CD friendly (desde que tenha Docker)

**Desvantagens:**
- ⏱️ Mais lento que testes unitários
- 🐳 Requer Docker rodando
- 💾 Consome mais recursos

### Estrutura dos Testes de Integração

Os testes de integração cobrem:

✅ **CRUD Completo**
- Insert (inserção de dados)
- Select (busca por ID)
- SelectAll (listar todos)
- Update (atualização)
- Delete (deleção)

✅ **Casos de Erro**
- Buscar registro não existente
- Banco vazio
- Múltiplas operações

✅ **Concorrência**
- Inserts concorrentes
- Integridade de dados

✅ **PostgreSQL Específico**
- SERIAL auto-increment
- Placeholders $1, $2
- RETURNING clause
- Isolamento de transações

### Boas Práticas de Teste

**Use `t.Run` para subtestes**:
```go
func TestRepository_Insert_Integration(t *testing.T) {
    t.Run("insert single book", func(t *testing.T) {
        // Teste 1
    })
    t.Run("insert multiple books", func(t *testing.T) {
        // Teste 2
    })
}
```

**Use test helpers para setup**:
```go
db := SetupLocalSQLite(t)
defer db.Close()

CreateTestSchema(t, ctx, db)
PopulateSampleData(t, ctx, db)
```

**Use `require` para assertions críticas**:
```go
require.NoError(t, err)      // Para no erro
assert.Equal(t, expected, actual)  // Continua no erro
```

---

## 🔧 Troubleshooting

### Erro: "panic: checked path: $XDG_RUNTIME_DIR" (testes PostgreSQL)

**Problema**: Testcontainers não consegue conectar ao Docker.

**Causas**:
- Docker não está rodando
- Variável `DOCKER_HOST` não configurada (WSL)
- Usuário sem permissão para acessar Docker

**Solução WSL/Linux**:
```bash
# Verificar se Docker está rodando
docker ps

# Se não estiver, iniciar
sudo service docker start

# Ou configurar DOCKER_HOST
export DOCKER_HOST=unix:///var/run/docker.sock
```

**Solução alternativa**: Os testes PostgreSQL são opcionais e educacionais. Use apenas testes SQLite:
```bash
go test -tags=integration ./book/turso/...
```

---

### Erro: "no such table: books"

**Problema**: A tabela não foi criada no banco de dados.

**Solução**:
```bash
turso db shell books-db "CREATE TABLE IF NOT EXISTS books (
  ID INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL,
  author TEXT NOT NULL,
  category INTEGER NOT NULL
);"
```

### Erro: "reading config file: Config File \".env\" Not Found"

**Problema**: Arquivo `.env` não existe ou não está na raiz do projeto.

**Solução**:
```bash
# Crie o arquivo .env na raiz do projeto
cat > .env << EOF
PORT = "8080"
DBNAME = "local.db"
TURSO_DATABASE_URL = "libsql://seu-db.turso.io"
TURSO_AUTH_TOKEN = "seu-token-aqui"
EOF
```

### Erro: "creating connector: TURSO_DATABASE_URL is empty"

**Problema**: Variáveis de ambiente não estão configuradas corretamente.

**Solução**:
1. Verifique se o arquivo `.env` existe
2. Confirme que as variáveis estão no formato TOML correto
3. Obtenha as credenciais corretas:
```bash
turso db show seu-db --url
turso db tokens create seu-db
```

### Erro: "port already in use" ou "bind: address already in use"

**Problema**: Porta 8080 já está sendo usada por outro processo.

**Solução**:
```bash
# Opção 1: Mude a porta no .env
PORT = "8081"

# Opção 2: Encontre e mate o processo usando a porta
lsof -ti:8080 | xargs kill -9  # Linux/Mac
```

### API retorna lista vazia

**Problema**: Banco de dados não tem dados.

**Solução**:
```bash
# Insira um livro de teste
curl -X POST http://localhost:8080/v1/books \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Test Book",
    "author": "Test Author",
    "category": "Read"
  }'
```

### Testes falhando com "panic: runtime error"

**Problema**: Mocks não foram gerados ou estão desatualizados.

**Solução**:
```bash
make generate-mocks
# ou
go tool mockery --output book/mocks --dir book --all
```

---

## 📚 Estrutura dos Testes

O projeto utiliza **subtestes** (`t.Run`) para organizar casos de teste:

```go
func TestCreate(t *testing.T) {
    t.Run("success", func(t *testing.T) {
        // Teste de sucesso
    })
    t.Run("fail", func(t *testing.T) {
        // Teste de falha
    })
}
```

Vantagens:
- Melhor organização
- Fácil identificação de qual caso falhou
- Outputs mais claros

---

## 🔄 Fluxo da Aplicação

### HTTP API

```
HTTP Request
    ↓
Chi Router (internal/http/chi/handlers.go)
    ↓
Handler Function (internal/http/chi/books.go)
    ↓
Request → Struct (bookRequest)
    ↓
BookService.Create/Update/... (book/service.go)
    ↓
Repository Interface (book/repository.go)
    ↓
Turso Repository Implementation (book/turso/repository.go)
    ↓
Turso/LibSQL Database
    ↓
Response ← Struct (bookResponse)
    ↓
JSON Response
```

### CLI

```
main (cmd/cli/main.go)
    ↓
Load Config (config/config.go)
    ↓
Create Turso Repository (book/turso/repository.go)
    ↓
Create Service (book/service.go)
    ↓
Call UseCase (Create, List, Get, etc)
    ↓
Display Result
```

---

## 🎯 Principais Padrões Implementados

### 1. **Dependency Injection**
A aplicação injeta dependências através do construtor:
```go
s := book.NewService(repo)  // Repo é injetado
r := chi.Handlers(ctx, s)   // Service é injetado
```

### 2. **Interface Segregation Principle**
Interfaces são pequenas e específicas:
- `Reader`: apenas leitura
- `Writer`: apenas escrita
- `Repository`: composição das duas

### 3. **Single Responsibility Principle**
Cada package tem uma responsabilidade clara:
- `book`: Entidades e lógica de negócio
- `config`: Apenas configuração
- `internal/http/chi`: Apenas HTTP

### 4. **Graceful Shutdown**
O servidor HTTP trata sinais adequadamente:
```go
ctx, stop := signal.NotifyContext(
    context.Background(),
    syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT,
)
```

---

## 💡 Decisões de Design

### Por que Value Semantics para Book?
- `Book` representa **dados**, não tem identidade mutável
- Value semantics são mais seguras e performáticas
- Reduz alocações no heap e pressão sobre o garbage collector

### Por que Pointer para Service?
- `Service` é uma **API** com métodos
- Precisa compartilhar estado (o repositório)
- Pointer semantics são padrão para tipos com métodos

### Por que Turso?
- Embedded SQLite com replicação automática
- Ideal para desenvolvimento e edge deployments
- Sincronização automática com servidor central

### Por que Chi?
- Router leve e extensível
- Suporte a middleware excelente
- Performance comparável a outros routers populares

---

## 📝 Comentários no Código

O código contém **comentários educacionais** explicando os conceitos. Exemplos:

**book/service.go**:
```go
/*
 * Quando uma struct representa DADOS deveria usar sempre value semantics e não pointer (ex: Book) .
 * Se a struct representa uma API deveria ser pointer (ex: Service).
 * Para tipos primários (int, string) sempre value semantics
 * Para tipos internos (maps, slices) usar value semantics
 */
```

**cmd/api/main.go**:
```go
/* "a porta de entrada e saída da minha aplicação"
 * É no arquivo main.go que vai ser compilado para gerar o executável,
 * onde é feita toda a "amarração" dos demais pacotes.
 */
```

---

## 🔗 Referências e Recursos

### Artigos do Autor (Elton Minetto)
- [Error Handling em CLI Applications Go](https://eltonminetto.dev/post/2022-07-06-error-handling-cli-applications-golang/)
- [Using Go Interfaces](https://eltonminetto.dev/post/2022-06-07-using-go-interfaces/)
- [Test Helpers em Go](https://eltonminetto.dev/post/2024-02-15-using-test-helpers/)

### Go Best Practices
- [Effective Go](https://golang.org/doc/effective_go)
- [Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go Package Naming](https://go.dev/blog/package-names)

### Arquitetura
- [Clean Architecture - Robert C. Martin](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Hexagonal Architecture - Alistair Cockburn](https://alistair.cockburn.us/hexagonal-architecture/)

### Turso/LibSQL
- [Turso Documentation](https://turso.tech/docs)
- [LibSQL Go Driver](https://github.com/tursodatabase/go-libsql)

### Bibliotecas Utilizadas
- [Chi Router](https://github.com/go-chi/chi)
- [Viper Configuration](https://github.com/spf13/viper)
- [Mockery](https://github.com/vektra/mockery)
- [Testify](https://github.com/stretchr/testify)
- [Testcontainers](https://testcontainers.com/)

### Testes
- [Test Helpers em Go - Elton Minetto](https://eltonminetto.dev/post/2024-02-15-using-test-helpers/)
- [Testcontainers - Documentação](https://golang.testcontainers.org/)
- [Go Testing Best Practices](https://go.dev/doc/effective_go#testing)

---

## 📄 Licença

Este projeto é fornecido como material educacional. Consulte o arquivo LICENSE para mais informações.

---

## 🤝 Contribuições

Este é um projeto educacional. Sugestões de melhorias nos comentários e estrutura são bem-vindas através de issues e pull requests.

---

## 📊 Schema do Banco de Dados

### Tabela: books

| Coluna | Tipo | Constraints | Descrição |
|--------|------|-------------|-----------|
| `ID` | INTEGER | PRIMARY KEY, AUTOINCREMENT | Identificador único do livro |
| `title` | TEXT | NOT NULL | Título do livro |
| `author` | TEXT | NOT NULL | Autor do livro |
| `category` | INTEGER | NOT NULL | Categoria (1=Want to Read, 2=Reading, 3=Read) |

### Mapeamento de Categorias

```go
const (
    WantToRead Category = 1
    Reading    Category = 2
    Read       Category = 3
)
```

| Valor | Nome String | Constante Go |
|-------|-------------|--------------|
| 1 | "Want to Read" | `book.WantToRead` |
| 2 | "Reading" | `book.Reading` |
| 3 | "Read" | `book.Read` |

### SQL para Criar a Tabela

```sql
CREATE TABLE IF NOT EXISTS books (
  ID INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL,
  author TEXT NOT NULL,
  category INTEGER NOT NULL
);
```

### Exemplos de Queries

**Inserir**:
```sql
INSERT INTO books (title, author, category)
VALUES ('Neuromancer', 'William Gibson', 3);
```

**Buscar todos**:
```sql
SELECT * FROM books;
```

**Buscar por ID**:
```sql
SELECT * FROM books WHERE id = 1;
```

**Atualizar**:
```sql
UPDATE books
SET title = 'New Title', author = 'New Author', category = 2
WHERE id = 1;
```

**Deletar**:
```sql
DELETE FROM books WHERE id = 1;
```

---

## 🐘 Usando PostgreSQL Localmente

### Iniciar PostgreSQL com Docker Compose

```bash
# Iniciar PostgreSQL container
make run-postgres

# Verificar que está pronto
docker ps | grep postgres

# Parar PostgreSQL
make stop-postgres
```

### Configurar para PostgreSQL

Copie o arquivo de exemplo:

```bash
cp .env.postgres.example .env
```

Edite `.env` com suas credenciais PostgreSQL (já vem com padrão docker-compose):

```toml
POSTGRES_HOST = "localhost"
POSTGRES_PORT = "5432"
POSTGRES_DB = "books"
POSTGRES_USER = "postgres"
POSTGRES_PASSWORD = "postgres"
POSTGRES_SSLMODE = "disable"
PORT = "8080"
```

### Executar CLI PostgreSQL

```bash
# Confirme que PostgreSQL está rodando
make run-postgres

# Execute o CLI
make cli-postgres

# Exemplo de saída:
# 🔗 Connecting to PostgreSQL at localhost:5432...
# ✅ Connected to PostgreSQL!
# 📝 Creating a new book...
# ✅ Book created successfully!
#    ID:       1
#    Title:    The Pragmatic Programmer
#    Author:   Andy Hunt & Dave Thomas
#    Category: Want to Read
```

---

## 🗄️ Database Migrations

As migrations são versionadas em SQL e podem ser aplicadas automaticamente.

### Arquivo de Migrations

Todas as migrations estão em `book/postgres/migrations/`:

```
001_create_books_table.up.sql      # Criar tabela
001_create_books_table.down.sql    # Reverter criação
```

### Aplicar Migrations

**Automático** (ao rodar docker-compose):
```bash
docker-compose up  # Migrations aplicadas automaticamente
```

**Manual** (se PostgreSQL já está rodando):
```bash
make migrate-up

# ou
psql -h localhost -U postgres -d books < book/postgres/migrations/001_create_books_table.up.sql
```

### Reverter Migrations

```bash
make migrate-down

# ou
docker-compose exec -T postgres psql -U postgres -d books -f /docker-entrypoint-initdb.d/001_create_books_table.down.sql
```

Veja `book/postgres/migrations/README.md` para mais detalhes.

---

## 📊 Benchmarks de Performance

Compare a performance entre SQLite e PostgreSQL:

```bash
# Rodar todos os benchmarks PostgreSQL
make benchmark

# Exemplo de saída:
# BenchmarkInsert_Postgres-8           123   9876543 ns/op  1024 B/op  10 allocs/op
# BenchmarkSelect_Postgres-8           456   2345678 ns/op   512 B/op   5 allocs/op
# BenchmarkSelectAll_Postgres-8        789   1234567 ns/op  2048 B/op  15 allocs/op
# BenchmarkUpdate_Postgres-8           234   3456789 ns/op   768 B/op   8 allocs/op
# BenchmarkDelete_Postgres-8           567   2345678 ns/op   256 B/op   3 allocs/op
# BenchmarkCRUD_Cycle-8                 89  11234567 ns/op  5120 B/op  40 allocs/op
```

### Entendendo os Resultados

```
BenchmarkInsert_Postgres-8    123    9876543 ns/op    1024 B/op    10 allocs/op
└─ Nome do benchmark   └─ CPUs └─ Iterações └─ ns/op └─ Bytes/op └─ Alocações/op
```

- **123 iterations**: Quantas vezes o teste rodou
- **9876543 ns/op**: ~9.8ms por operação
- **1024 B/op**: 1KB alocado por operação
- **10 allocs/op**: 10 alocações por operação

Menores valores = melhor performance.

---

## 🧪 Testes Unitários vs Integração

### Testes Unitários PostgreSQL

Rápidos, sem banco de dados real (usam sqlmock):

```bash
# Rodar testes unitários
go test ./book/postgres/...

# Com saída detalhada
go test -v ./book/postgres/...

# Apenas um teste
go test -run TestInsert_Unit ./book/postgres/...
```

Localização: `book/postgres/repository_test.go` (build tag: `!integration`)

### Testes de Integração PostgreSQL

Realistas, com PostgreSQL real via testcontainers:

```bash
# Rodar testes de integração
go test -tags=integration ./book/postgres/...

# Com benchmarks
go test -tags=integration -bench=. ./book/postgres/...

# Apenas testes de integração, sem benchmarks
go test -tags=integration ./book/postgres/repository_integration_test.go
```

Localização: `book/postgres/repository_integration_test.go` (build tag: `integration`)

---

## 📋 Docker Compose

Arquivo: `docker-compose.yml`

Serviços disponíveis:
- **postgres**: PostgreSQL 16 Alpine
  - Porta: 5432
  - Usuário: postgres
  - Senha: postgres
  - Banco: books

Volume: `postgres_data` (para persistência)

Healthcheck: Verifica a cada 10s se PostgreSQL está pronto

---

## 💡 Próximas Melhorias

### ✅ Implementadas
- [x] Testes de integração com testcontainers (PostgreSQL)
- [x] Múltiplas implementações de Repository (SQLite + PostgreSQL)
- [x] CLI PostgreSQL
- [x] Database Migrations (SQL)
- [x] Docker Compose setup
- [x] Benchmarks de performance
- [x] Testes unitários com sqlmock
- [x] Documentação completa (README)

### 🔜 Próximas
- [ ] Implementar autenticação (JWT)
- [ ] Adicionar middlewares de validação
- [ ] Implementar paginação nos endpoints
- [ ] Adicionar buscas e filtros
- [ ] Documentação OpenAPI/Swagger
- [ ] Testes de integração HTTP end-to-end
- [ ] Validação de entrada (request body)
- [ ] Tratamento de erros mais granular (404 para not found)
- [ ] Migration framework automático (goose, sql-migrate)

---

**Criado por**: Elton Minetto
**Objetivo**: Demonstrar as melhores práticas e padrões de desenvolvimento em Go

*"Write code as the Go way. Not the way of other languages."* - Gophers
