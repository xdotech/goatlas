-- +goose Up

-- Add repo column to scope files per repository.
ALTER TABLE files ADD COLUMN repo TEXT NOT NULL DEFAULT '';

-- Replace the old unique-on-path constraint with (repo, path).
ALTER TABLE files DROP CONSTRAINT files_path_key;
ALTER TABLE files ADD CONSTRAINT files_repo_path_key UNIQUE (repo, path);

CREATE INDEX idx_files_repo ON files(repo);

-- +goose Down
DROP INDEX idx_files_repo;
ALTER TABLE files DROP CONSTRAINT files_repo_path_key;
ALTER TABLE files ADD CONSTRAINT files_path_key UNIQUE (path);
ALTER TABLE files DROP COLUMN repo;
