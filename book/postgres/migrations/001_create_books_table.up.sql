-- Create books table
CREATE TABLE IF NOT EXISTS books (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    author TEXT NOT NULL,
    category INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index on title for faster searches
CREATE INDEX IF NOT EXISTS idx_books_title ON books(title);

-- Create index on category for filtering
CREATE INDEX IF NOT EXISTS idx_books_category ON books(category);

-- Add comment to table
COMMENT ON TABLE books IS 'Stores book information for the library system';
COMMENT ON COLUMN books.id IS 'Unique identifier for the book';
COMMENT ON COLUMN books.title IS 'Title of the book';
COMMENT ON COLUMN books.author IS 'Author of the book';
COMMENT ON COLUMN books.category IS 'Category: 1=WantToRead, 2=Reading, 3=Read';
