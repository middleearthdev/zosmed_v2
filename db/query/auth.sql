-- app_user queries for the login + onboarding backend (ADR-003 §2.3).
-- Handlers never write SQL directly — everything routes through
-- apps/api/internal/auth/store.go (SoC §12a-3).

-- name: CreateUser :one
INSERT INTO app_user (email, password_hash) VALUES (@email, @password_hash) RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM app_user WHERE email = @email;

-- name: GetUserByID :one
SELECT * FROM app_user WHERE id = @id;

-- name: SetUserSegment :one
UPDATE app_user SET segment = @segment WHERE id = @id RETURNING *;

-- name: CompleteOnboarding :one
UPDATE app_user SET onboarding_completed_at = now() WHERE id = @id RETURNING *;
