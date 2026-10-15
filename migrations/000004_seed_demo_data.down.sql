DELETE FROM users WHERE email IN ('librarian@example.test', 'reader@example.test');
DELETE FROM copies WHERE inventory_no LIKE 'SEED-%';
DELETE FROM book_authors WHERE book_id IN (
    SELECT id FROM books WHERE isbn LIKE '97800000000%'
);
DELETE FROM books WHERE isbn LIKE '97800000000%';
DELETE FROM authors AS a
WHERE a.full_name IN (
    'Ada Lovelace', 'Grace Hopper', 'James Baldwin', 'Rachel Carson', 'Ursula K. Le Guin'
)
AND NOT EXISTS (
    SELECT 1 FROM book_authors AS ba WHERE ba.author_id = a.id
);
DELETE FROM categories AS c
WHERE c.name IN ('Programming', 'History', 'Science')
AND NOT EXISTS (
    SELECT 1 FROM books AS b WHERE b.category_id = c.id
);