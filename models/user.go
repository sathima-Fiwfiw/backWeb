package models

import (
	"context"
	"database/sql"
	"errors" // เพิ่ม import สำหรับ errors
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
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &u, nil
}

// GetByEmail ใช้สำหรับดึงข้อมูลผู้ใช้ด้วย email และคืนค่า password hash ด้วย
func (m *UserModel) GetByEmail(ctx context.Context, email string) (*User, string, error) {
	row := m.DB.QueryRowContext(ctx, `
		SELECT user_id, username, email, password_hash, IFNULL(image_profile,''), role, created_at, created_at
		FROM users WHERE email = ?`, email)

	var u User
	var passwordHash string

	if err := row.Scan(&u.ID, &u.Username, &u.Email, &passwordHash, &u.AvatarURL, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, "", nil // ไม่พบผู้ใช้ แต่ไม่มี error
		}
		return nil, "", err
	}
	return &u, passwordHash, nil
}

// Authenticate ใช้ตรวจสอบรหัสผ่าน
func (m *UserModel) Authenticate(ctx context.Context, email, password string) (*User, error) {
	u, passwordHash, err := m.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if u == nil {
		// ผู้ใช้ไม่พบ
		return nil, errors.New("invalid credentials")
	}

	// เปรียบเทียบรหัสผ่านที่รับมากับรหัสผ่านที่ถูกแฮชในฐานข้อมูล
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return nil, errors.New("invalid credentials") // รหัสผ่านไม่ตรง
		}
		return nil, err
	}

	// ล็อกอินสำเร็จ
	return u, nil
}
