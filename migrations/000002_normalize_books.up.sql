CREATE TABLE categories (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE authors (
    id BIGSERIAL PRIMARY KEY,
    full_name TEXT NOT NULL UNIQUE,
    bio TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO categories (name)
SELECT DISTINCT btrim(category)
FROM books
WHERE btrim(category) <> ''
ON CONFLICT (name) DO NOTHING;

INSERT INTO authors (full_name)
SELECT DISTINCT btrim(author_name)
FROM books
CROSS JOIN LATERAL regexp_split_to_table(author, ',') AS author_name
WHERE btrim(author_name) <> ''
ON CONFLICT (full_name) DO NOTHING;

ALTER TABLE books ADD COLUMN category_id BIGINT;
UPDATE books AS b
SET category_id = c.id
FROM categories AS c
WHERE c.name = btrim(b.category);

CREATE TABLE book_authors (
    book_id BIGINT NOT NULL,
    author_id BIGINT NOT NULL,
    author_order INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (book_id, author_id),
    CONSTRAINT fk_book_authors_book FOREIGN KEY (book_id)
        REFERENCES books(id) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT fk_book_authors_author FOREIGN KEY (author_id)
        REFERENCES authors(id) ON UPDATE CASCADE ON DELETE CASCADE
);

INSERT INTO book_authors (book_id, author_id, author_order)
SELECT DISTINCT b.id, a.id, split.author_order
FROM books AS b
CROSS JOIN LATERAL regexp_split_to_table(b.author, ',') WITH ORDINALITY AS split(author_name, author_order)
JOIN authors AS a ON a.full_name = btrim(split.author_name)
ON CONFLICT DO NOTHING;

ALTER TABLE books ALTER COLUMN category_id SET NOT NULL;
ALTER TABLE books ADD CONSTRAINT fk_books_category FOREIGN KEY (category_id)
    REFERENCES categories(id) ON UPDATE CASCADE ON DELETE RESTRICT;
ALTER TABLE books DROP COLUMN author;
ALTER TABLE books DROP COLUMN category;
CREATE INDEX idx_books_category_id ON books(category_id);
CREATE INDEX idx_book_authors_author_id ON book_authors(author_id);

CREATE TABLE copies (
    id BIGSERIAL PRIMARY KEY,
    book_id BIGINT NOT NULL,
    inventory_no TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'available'
        CHECK (status IN ('available', 'borrowed', 'lost')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_copies_book FOREIGN KEY (book_id)
        REFERENCES books(id) ON UPDATE CASCADE ON DELETE CASCADE
);
CREATE INDEX idx_copies_book_id ON copies(book_id);

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    role TEXT NOT NULL DEFAULT 'member'
        CHECK (role IN ('admin', 'member')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);