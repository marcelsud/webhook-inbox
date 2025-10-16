-- Drop indexes
DROP INDEX IF EXISTS idx_books_category;
DROP INDEX IF EXISTS idx_books_title;

-- Drop table
DROP TABLE IF EXISTS books CASCADE;
