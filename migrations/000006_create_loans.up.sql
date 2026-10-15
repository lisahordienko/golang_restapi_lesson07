CREATE TABLE loans (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    copy_id BIGINT NOT NULL,
    loaned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    due_at TIMESTAMPTZ NOT NULL,
    returned_at TIMESTAMPTZ,
    CONSTRAINT fk_loans_user FOREIGN KEY (user_id)
        REFERENCES users(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    CONSTRAINT fk_loans_copy FOREIGN KEY (copy_id)
        REFERENCES copies(id) ON UPDATE CASCADE ON DELETE RESTRICT
);
CREATE INDEX idx_loans_user_id ON loans(user_id);
CREATE INDEX idx_loans_copy_id ON loans(copy_id);