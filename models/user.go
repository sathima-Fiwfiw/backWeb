package models

import (
	"context"
	"database/sql"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	AvatarURL string    `json:"avatar_url"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserModel struct {
	DB *sql.DB
}

func (m *UserModel) Create(ctx context.Context, username, email, password, avatarURL string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	res, err := m.DB.ExecContext(ctx, `
		INSERT INTO users (username, email, password_hash, image_profile)
		VALUES (?, ?, ?, ?)`,
		username, email, hash, avatarURL,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return m.GetByID(ctx, int(id))
}

func (m *UserModel) GetByID(ctx context.Context, id int) (*User, error) {
	row := m.DB.QueryRowContext(ctx, `
		SELECT user_id, username, email, IFNULL(image_profile,''), role, created_at, created_at
		FROM users WHERE user_id = ?`, id)

	var u User
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.AvatarURL, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}
