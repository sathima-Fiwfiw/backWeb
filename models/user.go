package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
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

// Create: สมัครผู้ใช้ใหม่... (โค้ดเดิม)

// GetByID: ดึงข้อมูลผู้ใช้ด้วย ID (สำหรับ handleGetProfile)
func (m *UserModel) GetByID(ctx context.Context, userID int) (*User, error) {
	row := m.DB.QueryRowContext(ctx, `
		-- แก้: ดึง updated_at มาด้วย แต่ใช้ created_at เป็นค่า fallback
		SELECT user_id, username, email, role, IFNULL(image_profile,''), created_at, IFNULL(updated_at, created_at)
		FROM users
		WHERE user_id = ?
	`, userID)

	var (
		u         User
		imageProf sql.NullString
		created   time.Time
		updated   time.Time
	)

	if err := row.Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.Role,
		&imageProf,
		&created,
		&updated,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	u.AvatarURL = imageProf.String
	u.CreatedAt = created
	u.UpdatedAt = updated

	return &u, nil
}

// GetByEmail: ดึงข้อมูลผู้ใช้ด้วย email... (โค้ดเดิม)
// Authenticate: ตรวจสอบอีเมล/รหัสผ่าน... (โค้ดเดิม)

// UpdateProfile: อัปเดตข้อมูลโปรไฟล์ของผู้ใช้ (Username, Email, Password)
// *หมายเหตุ: ลบคอลัมน์ updated_at ออกจาก query เพื่อแก้ Error 1054
func (m *UserModel) UpdateProfile(ctx context.Context, userID int, username, email, password string) error {
	// 1. ตรวจสอบว่าต้องอัปเดตรหัสผ่านหรือไม่
	var pwHash []byte
	var err error
	if password != "" {
		pwHash, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
	}

	// 2. สร้าง Query แบบ Dynamic เพื่ออัปเดตเฉพาะฟิลด์ที่มีการเปลี่ยนแปลง
	var fields []string
	var args []interface{}

	if username != "" {
		fields = append(fields, "username = ?")
		args = append(args, username)
	}
	if email != "" {
		fields = append(fields, "email = ?")
		args = append(args, email)
	}
	if len(pwHash) > 0 {
		fields = append(fields, "password_hash = ?")
		args = append(args, pwHash)
	}

	if len(fields) == 0 {
		return errors.New("no fields provided for update")
	}

	// *** [แก้ไข]: ลบบรรทัดนี้ออกเพื่อแก้ปัญหา Unknown column 'updated_at' ***
	// fields = append(fields, "updated_at = NOW()")
	args = append(args, userID) // userID เป็นตัวสุดท้ายสำหรับ WHERE

	query := fmt.Sprintf("UPDATE users SET %s WHERE user_id = ?", strings.Join(fields, ", "))

	// 3. Execute Query
	result, err := m.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("user not found or no new changes made")
	}

	return nil
}
