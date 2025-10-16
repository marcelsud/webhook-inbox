# Database Migrations

Migrations SQL para PostgreSQL usando padrão **up/down**.

## Convenção de Nomes

- `NNN_description.up.sql` - Aplicar migration
- `NNN_description.down.sql` - Reverter migration

Exemplo: `001_create_books_table.up.sql` / `001_create_books_table.down.sql`

## Como Aplicar Migrations

### Opção 1: Usando psql CLI

```bash
# Conectar ao banco e aplicar migration
psql -h localhost -U postgres -d books -f 001_create_books_table.up.sql

# Reverter migration
psql -h localhost -U postgres -d books -f 001_create_books_table.down.sql
```

### Opção 2: Usando Docker Compose

```bash
# Aplicar migrations automaticamente (ao iniciar)
docker-compose up

# Aplicar migration manual
docker-compose exec -T postgres psql -U postgres -d books -f /docker-entrypoint-initdb.d/001_create_books_table.up.sql
```

### Opção 3: Usando Makefile

```bash
# Iniciar PostgreSQL
make run-postgres

# Aplicar migrations
make migrate-up

# Reverter migrations
make migrate-down
```

## Estrutura das Migrations

### UP Migration (001_create_books_table.up.sql)

```sql
-- 1. Create table
CREATE TABLE IF NOT EXISTS books (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    author TEXT NOT NULL,
    category INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. Create indexes
CREATE INDEX IF NOT EXISTS idx_books_title ON books(title);
CREATE INDEX IF NOT EXISTS idx_books_category ON books(category);

-- 3. Add comments (documentação)
COMMENT ON TABLE books IS 'Stores book information';
```

### DOWN Migration (001_create_books_table.down.sql)

```sql
-- Reverse order: drop indexes first, then table
DROP INDEX IF EXISTS idx_books_category;
DROP INDEX IF EXISTS idx_books_title;
DROP TABLE IF EXISTS books CASCADE;
```

## Boas Práticas

✅ **DO's**:
- Use `IF NOT EXISTS` / `IF EXISTS` para idempotência
- Sempre crie indexes para foreign keys
- Adicione comments para documentar
- Uma mudança lógica por migration
- Reverter completamente em DOWN

❌ **DON'Ts**:
- Não use `DROP TABLE` sem verificar
- Não mude estrutura de tabela em DOWN (apenas DROP)
- Não execute migrations manualmente em produção
- Não esqueça de fazer backup antes

## Próximas Migrations

Exemplos de outras migrations que poderiam ser adicionadas:

```sql
-- 002_add_isbn_to_books.up.sql
ALTER TABLE books ADD COLUMN isbn VARCHAR(13) UNIQUE;

-- 003_create_users_table.up.sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    name TEXT NOT NULL
);

-- 004_add_user_id_to_books.up.sql
ALTER TABLE books ADD COLUMN user_id INTEGER REFERENCES users(id);
```

## Referências

- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Migration Best Practices](https://wiki.postgresql.org/wiki/Versioning)
