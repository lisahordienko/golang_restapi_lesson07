DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS copies;

ALTER TABLE books ADD COLUMN author TEXT;
ALTER TABLE books ADD COLUMN category TEXT;
UPDATE books AS b
SET category = c.name
FROM categories AS c
WHERE c.id = b.category_id;
UPDATE books AS b
SET author = authors.full_names
FROM (
    SELECT ba.book_id, string_agg(a.full_name, ', ' ORDER BY ba.author_order) AS full_names
    FROM book_authors AS ba
    JOIN authors AS a ON a.id = ba.author_id
    GROUP BY ba.book_id
) AS authors
WHERE authors.book_id = b.id;
ALTER TABLE books ALTER COLUMN author SET NOT NULL;
ALTER TABLE books ALTER COLUMN category SET NOT NULL;
DROP TABLE IF EXISTS book_authors;
ALTER TABLE books DROP CONSTRAINT fk_books_category;
ALTER TABLE books DROP COLUMN category_id;

DROP TABLE IF EXISTS authors;
DROP TABLE IF EXISTS categories;