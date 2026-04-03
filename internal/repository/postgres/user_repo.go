package postgres

import (
	"context"
	"errors"

	"booking-service/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	const q = `
SELECT id, email, password_hash, role, created_at
FROM users
WHERE email = $1
`
	db := dbFromContext(ctx, r.pool)

	var u domain.User
	var passwordHash pgtype.Text
	if err := db.QueryRow(ctx, q, email).Scan(&u.ID, &u.Email, &passwordHash, &u.Role, &u.CreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if passwordHash.Valid {
		v := passwordHash.String
		u.PasswordHash = &v
	}
	return &u, nil
}

func (r *UserRepo) Create(ctx context.Context, user domain.User) error {
	const q = `
INSERT INTO users (id, email, password_hash, role, created_at)
VALUES ($1, $2, $3, $4, $5)
`
	db := dbFromContext(ctx, r.pool)
	_, err := db.Exec(ctx, q, user.ID, user.Email, user.PasswordHash, user.Role, user.CreatedAt)
	if isUsersEmailUniqueViolation(err) {
		// Explicitly mapped to INVALID_REQUEST for /register API contract (duplicate email -> 400).
		return domain.NewDomainError(domain.ErrorInvalidRequest, "email already exists")
	}
	return err
}

func isUsersEmailUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23505" && pgErr.ConstraintName == "users_email_key"
}
