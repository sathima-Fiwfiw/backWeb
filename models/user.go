package models

import (
	"context"
	"database/sql"
<<<<<<< HEAD
	"errors"
=======
	"errors" // เพิ่ม import สำหรับ errors
>>>>>>> 3538e17ad9e714499c6c65b68497e2fdaeb0071d
	"time"

	"golang.org/x/crypto/bcrypt"
)

// โครงสร้างผู้ใช้ (ให้ชื่อฟิลด์/ชนิดตรงกับที่ FE ใช้)
type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	AvatarURL string    `json:"avatar_url"` // map จากคอลัมน์ image_profile
	Role      string    `json:"role"`       // user / admin (มาจาก DB)
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ตัวโมเดลที่ผูกกับ *sql.DB
type UserModel struct {
	DB *sql.DB
}

// Create: สมัครผู้ใช้ใหม่
// - แฮชรหัสผ่านด้วย bcrypt
// - ใส่รูปโปรไฟล์ได้ (อาจว่าง)
// - role ปล่อยให้เป็น DEFAULT ของตาราง (เช่น 'user')
func (m *UserModel) Create(ctx context.Context, username, email, password, avatarURL string) (*User, error) {
	pwHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// แทรกข้อมูล (ให้คอลัมน์ตรงกับตาราง: users)
	// ตารางอ้างอิง: users(user_id, username, email, password_hash, role, image_profile, created_at)
	res, err := m.DB.ExecContext(ctx, `
		INSERT INTO users (username, email, password_hash, image_profile, created_at)
		VALUES (?, ?, ?, ?, NOW())
	`, username, email, pwHash, avatarURL)
	if err != nil {
		return nil, err
	}

<<<<<<< HEAD
	id64, err := res.LastInsertId()
	if err != nil {
=======
func (m *UserModel) GetByID(ctx context.Context, id int) (*User, error) {
	row := m.DB.QueryRowContext(ctx, `
		SELECT user_id, username, email, IFNULL(image_profile,''), role, created_at, created_at
		FROM users WHERE user_id = ?`, id)

	var u User
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.AvatarURL, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
>>>>>>> 3538e17ad9e714499c6c65b68497e2fdaeb0071d
		return nil, err
	}

	u := &User{
		ID:        int(id64),
		Username:  username,
		Email:     email,
		AvatarURL: avatarURL,
		Role:      "user",     // ค่าเริ่มต้น (ตรงกับ DEFAULT ของ DB)
		CreatedAt: time.Now(), // อ้างอิงเวลาปัจจุบัน (ถ้าต้องการอ่านจาก DB จริงค่อย SELECT อีกครั้ง)
		UpdatedAt: time.Now(),
	}
	return u, nil
}

// GetByEmail: ดึงข้อมูลผู้ใช้ด้วย email พร้อม password_hash (เพื่อนำไป verify)
// เลือกคอลัมน์ role มาด้วยเสมอ!
func (m *UserModel) GetByEmail(ctx context.Context, email string) (*User, []byte, error) {
	row := m.DB.QueryRowContext(ctx, `
		SELECT user_id, username, email, password_hash, role, IFNULL(image_profile,''), created_at
		FROM users
		WHERE email = ?
	`, email)

	var (
		u         User
		hash      []byte
		imageProf sql.NullString
		created   time.Time
	)

	if err := row.Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&hash,   // password_hash (VARBINARY / VARCHAR ก็ scan เป็น []byte ได้)
		&u.Role, // ← สำคัญ: เอา role จาก DB
		&imageProf,
		&created,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, errors.New("user not found")
		}
		return nil, nil, err
	}

	u.AvatarURL = imageProf.String
	u.CreatedAt = created
	u.UpdatedAt = created // ถ้ายังไม่มี updated_at แยกในตาราง ก็อิง created ไปก่อน

	return &u, hash, nil
}

// Authenticate: ตรวจสอบอีเมล/รหัสผ่าน แล้วคืน User (role จาก DB)
func (m *UserModel) Authenticate(ctx context.Context, email, password string) (*User, error) {
	u, hash, err := m.GetByEmail(ctx, email)
	if err != nil {
		// รวม error message เป็น invalid credentials เพื่อความปลอดภัย
		return nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword(hash, []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	return u, nil
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
