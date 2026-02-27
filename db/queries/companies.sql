-- name: InsertCompany :exec
INSERT INTO companies(name, short_description, size)
VALUES (
  sqlc.arg('name'),
  sqlc.arg('short_description'),
  sqlc.arg('size')
)
ON CONFLICT (name) DO NOTHING;
