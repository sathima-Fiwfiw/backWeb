package router

import (
	"backWeb/models"
	"context" // ต้องใช้ context เพื่อส่ง claims ผ่าน middleware
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	// "time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
)

// === 1. Context Key & Helper Functions ===

// contextKey ใช้สำหรับ Key ใน context เพื่อหลีกเลี่ยงการชนกัน
type contextKey string

const ContextKeyClaims contextKey = "claims"

// envOr: ฟังก์ชันสำหรับอ่าน Environment Variable หรือใช้ค่า Default
func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// writeJSON: Helper สำหรับเขียน Response ในรูปแบบ JSON
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeErr: Helper สำหรับเขียน Error Response ในรูปแบบ JSON
func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// === 2. JWT Config and Structs ===

var jwtKey = []byte(envOr("JWT_SECRET", "a-very-secret-key-that-must-be-changed-in-production"))

// MyClaims กำหนดข้อมูลที่จะใส่ใน JWT Payload
type MyClaims struct {
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// AuthResponse คือโครงสร้างที่ Frontend คาดหวัง
type AuthResponse struct {
	Token string `json:"token"`
	Role  string `json:"role"`
}

type App struct {
	Users       *models.UserModel
	UploadDir   string
	AllowOrigin string // CORS Origin
}

// ProfileUpdateRequest คือโครงสร้างที่ใช้รับข้อมูล PUT จาก Frontend
type ProfileUpdateRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"` // อาจว่างเปล่า
}

// === 3. Middleware Implementations ===

// corsMiddleware: Middleware สำหรับจัดการ CORS
func corsMiddleware(allowOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// authMiddleware: Middleware สำหรับตรวจสอบ JWT และนำ Claims ไปใส่ใน Context
func (app *App) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			writeErr(w, http.StatusUnauthorized, "Missing or invalid token format")
			return
		}

		tokenStr := authHeader[7:]
		claims := &MyClaims{}

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Method)
			}
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			writeErr(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		// ใส่ claims ลงใน context เพื่อส่งต่อไปยัง handler
		ctx := context.WithValue(r.Context(), ContextKeyClaims, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// === 4. Placeholders for Auth Handlers (เพื่อไม่ให้โค้ดแดง) ===

func (app *App) handleRegister(w http.ResponseWriter, r *http.Request) {
	// ต้องมีการ implement logic การลงทะเบียน
	writeErr(w, http.StatusNotImplemented, "Registration endpoint not implemented yet.")
}

func (app *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	// ต้องมีการ implement logic การเข้าสู่ระบบ
	writeErr(w, http.StatusNotImplemented, "Login endpoint not implemented yet.")
}

// === 5. Router Setup and Profile Handlers ===

func New(app *App) http.Handler {
	r := chi.NewRouter()

	// ... ensure upload dir ... (โค้ดที่ถูกละไว้)

	// ใช้ middleware CORS ที่ถูกเพิ่มเข้ามา
	r.Use(corsMiddleware(app.AllowOrigin))

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", app.handleRegister)
			r.Post("/login", app.handleLogin)
		})

		// -------------------------
		// Secured Routes (ต้องมี JWT)
		// -------------------------
		r.Route("/user", func(r chi.Router) {
			r.Use(app.authMiddleware) // ใช้ middleware ในทุก route ย่อย

			// GET /api/v1/user/profile - ดึงข้อมูลโปรไฟล์
			r.Get("/profile", app.handleGetProfile)

			// PUT /api/v1/user/profile - อัปเดตข้อมูลโปรไฟล์
			r.Put("/profile", app.handleUpdateProfile)
		})
	})
	// ... existing static/upload routes ... (โค้ดที่ถูกละไว้)
	return r
}

// handleGetProfile: ดึงข้อมูลโปรไฟล์ของ user ที่ล็อคอินอยู่
func (app *App) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(ContextKeyClaims).(*MyClaims)

	// ดึงข้อมูลผู้ใช้จาก DB โดยใช้ ID จาก JWT
	user, err := app.Users.GetByID(r.Context(), claims.UserID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Failed to fetch user data")
		return
	}

	// ส่งข้อมูล User กลับไป
	writeJSON(w, http.StatusOK, user)
}

// handleUpdateProfile: อัปเดตข้อมูลโปรไฟล์
func (app *App) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(ContextKeyClaims).(*MyClaims)

	// 1. อ่านและ Parse Request Body
	var req ProfileUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// 2. เรียก Model เพื่ออัปเดตข้อมูลใน DB
	err := app.Users.UpdateProfile(
		r.Context(),
		claims.UserID,
		req.Username,
		req.Email,
		req.Password,
	)

	if err != nil {
		// จัดการ Error เช่น Username/Email ซ้ำ
		if strings.Contains(err.Error(), "Duplicate entry") {
			writeErr(w, http.StatusConflict, "Username or Email already taken.")
			return
		}
		log.Println("Database update error:", err)
		writeErr(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	// 3. ส่ง response สำเร็จกลับไป
	writeJSON(w, http.StatusOK, map[string]string{"message": "Profile updated successfully"})
}
