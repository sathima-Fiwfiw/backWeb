module backWeb // ← ให้ตรงกับ import "backWeb/models"

go 1.24.0 // ← ใช้เวอร์ชัน Go ที่คุณติดตั้งอยู่ (แนะนำ 1.22)

require (
	github.com/go-chi/chi/v5 v5.2.3
	github.com/go-sql-driver/mysql v1.9.3
	github.com/golang-jwt/jwt/v5 v5.3.0
	github.com/joho/godotenv v1.5.1
	golang.org/x/crypto v0.42.0
)

require filippo.io/edwards25519 v1.1.0 // indirect