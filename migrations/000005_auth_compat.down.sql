ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
UPDATE users SET role = 'member' WHERE role IN ('reader', 'librarian');
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'member';
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('admin', 'member'));
ALTER TABLE users DROP COLUMN IF EXISTS password_hash;