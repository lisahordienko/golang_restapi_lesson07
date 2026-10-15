ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'reader';
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
UPDATE users SET role = 'reader' WHERE role = 'member';
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('reader', 'librarian', 'admin'));