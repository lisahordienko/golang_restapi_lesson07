INSERT INTO categories (name) VALUES
    ('Programming'), ('History'), ('Science')
ON CONFLICT (name) DO NOTHING;

INSERT INTO authors (full_name, bio)
SELECT seed.full_name, seed.bio
FROM (VALUES
    ('Ada Lovelace', 'Mathematician and writer.'),
    ('Grace Hopper', 'Computer scientist and naval officer.'),
    ('James Baldwin', 'American novelist and essayist.'),
    ('Rachel Carson', 'American marine biologist and author.'),
    ('Ursula K. Le Guin', 'American author of speculative fiction.')
) AS seed(full_name, bio)
WHERE NOT EXISTS (
    SELECT 1 FROM authors AS existing WHERE existing.full_name = seed.full_name
);

INSERT INTO books (title, isbn, category_id) VALUES
    ('Computing Foundations', '9780000000001', (SELECT id FROM categories WHERE name = 'Programming')),
    ('The Analytical Engine', '9780000000002', (SELECT id FROM categories WHERE name = 'Programming')),
    ('Compiler Design Notes', '9780000000003', (SELECT id FROM categories WHERE name = 'Programming')),
    ('Stories of the Atlantic', '9780000000004', (SELECT id FROM categories WHERE name = 'History')),
    ('The Long View', '9780000000005', (SELECT id FROM categories WHERE name = 'History')),
    ('Silent Spring', '9780000000006', (SELECT id FROM categories WHERE name = 'Science')),
    ('Ocean Life', '9780000000007', (SELECT id FROM categories WHERE name = 'Science')),
    ('Earth and Sky', '9780000000008', (SELECT id FROM categories WHERE name = 'Science')),
    ('Algorithms at Work', '9780000000009', (SELECT id FROM categories WHERE name = 'Programming')),
    ('A Natural History', '9780000000010', (SELECT id FROM categories WHERE name = 'Science'))
ON CONFLICT (isbn) DO NOTHING;

INSERT INTO book_authors (book_id, author_id, author_order)
SELECT b.id, a.id, seed.author_order
FROM (VALUES
    ('9780000000001', 'Ada Lovelace', 1), ('9780000000001', 'Grace Hopper', 2),
    ('9780000000002', 'Ada Lovelace', 1), ('9780000000002', 'Ursula K. Le Guin', 2),
    ('9780000000003', 'Grace Hopper', 1), ('9780000000003', 'Ursula K. Le Guin', 2),
    ('9780000000004', 'James Baldwin', 1), ('9780000000004', 'Ursula K. Le Guin', 2),
    ('9780000000005', 'James Baldwin', 1),
    ('9780000000006', 'Rachel Carson', 1), ('9780000000006', 'Ada Lovelace', 2),
    ('9780000000007', 'Rachel Carson', 1), ('9780000000008', 'Rachel Carson', 1),
    ('9780000000008', 'Ursula K. Le Guin', 2),
    ('9780000000009', 'Grace Hopper', 1), ('9780000000009', 'Ada Lovelace', 2),
    ('9780000000010', 'Rachel Carson', 1), ('9780000000010', 'James Baldwin', 2)
) AS seed(isbn, full_name, author_order)
JOIN books AS b ON b.isbn = seed.isbn
JOIN authors AS a ON a.full_name = seed.full_name
ON CONFLICT DO NOTHING;

INSERT INTO copies (book_id, inventory_no, status)
SELECT books.id, 'SEED-' || books.id || '-' || copy_no,
       CASE WHEN copy_no = 3 THEN 'borrowed' ELSE 'available' END
FROM books
CROSS JOIN generate_series(1, 3) AS copy_no
WHERE books.isbn LIKE '97800000000%'
ON CONFLICT (inventory_no) DO NOTHING;

INSERT INTO users (email, password_hash, role) VALUES
    ('admin@example.test', '$2a$10$Touuttgkd3jbaP8oenep0u/C9Qdeyxka7CKJWUz66M8.fAU6QVdDW', 'admin'),
    ('librarian@example.test', '$2a$10$Touuttgkd3jbaP8oenep0u/C9Qdeyxka7CKJWUz66M8.fAU6QVdDW', 'librarian'),
    ('reader@example.test', '$2a$10$Touuttgkd3jbaP8oenep0u/C9Qdeyxka7CKJWUz66M8.fAU6QVdDW', 'reader')
ON CONFLICT (email) DO NOTHING;