package router

import (
	"backWeb/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
)

// JWT Config and Structs
// ต้องกำหนด Secret Key ที่มีความยาวและซับซ้อน (ดึงจาก ENV หรือใช้ค่าตั้งต้น)
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

func New(app *App) http.Handler {
	// ensure upload dir
	if app.UploadDir == "" {
		app.UploadDir = "uploads/avatars"
	}
	_ = os.MkdirAll(app.UploadDir, 0755)

	r := chi.NewRouter()

	// --- CORS Middleware ---
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			origin := app.AllowOrigin
			if origin == "" {
				origin = "*"
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			if req.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, req)
		})
	})

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// ========= /api =========
	r.Route("/api", func(r chi.Router) {
		// POST /api/register (multipart/form-data)
		r.Post("/register", func(w http.ResponseWriter, r *http.Request) {
			if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB
				writeErr(w, http.StatusBadRequest, "invalid form: "+err.Error())
				return
			}
			username := strings.TrimSpace(r.FormValue("username"))
			email := strings.TrimSpace(r.FormValue("email"))
			password := r.FormValue("password")
			if username == "" || email == "" || password == "" {
				writeErr(w, http.StatusBadRequest, "username/email/password required")
				return
			}

			// ไฟล์รูป (ถ้ามี)
			var avatarURL string
			file, header, err := r.FormFile("avatar")
			if err == nil && header != nil {
				defer file.Close()
				ext := filepath.Ext(header.Filename)
				if ext == "" {
					ext = ".jpg"
				}
				filename := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), slugify(username), ext)
				dstPath := filepath.Join(app.UploadDir, filename)

				dst, err := os.Create(dstPath)
				if err != nil {
					writeErr(w, http.StatusInternalServerError, "cannot save file")
					return
				}
				defer dst.Close()
				if _, err := io.Copy(dst, file); err != nil {
					writeErr(w, http.StatusInternalServerError, "cannot write file")
					return
				}
				avatarURL = "/" + filepath.ToSlash(dstPath)
			}

			u, err := app.Users.Create(r.Context(), username, email, password, avatarURL)
			if err != nil {
				writeErr(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusCreated, u)
		})

		// POST /api/login (application/json) — ใช้ DB อย่างเดียว
		r.Post("/login", func(w http.ResponseWriter, r *http.Request) {
			var creds struct {
				Email    string `json:"email"`
				Password string `json:"password"`
			}

			if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
				writeErr(w, http.StatusBadRequest, "invalid request body")
				return
			}
			if creds.Email == "" || creds.Password == "" {
				writeErr(w, http.StatusBadRequest, "email and password required")
				return
			}

			// 1) ตรวจสอบผู้ใช้จาก DB (ต้องสมัครก่อน)
			user, err := app.Users.Authenticate(r.Context(), creds.Email, creds.Password)
			if err != nil {
				writeErr(w, http.StatusUnauthorized, "login failed: "+err.Error())
				return
			}

			// 2) สร้าง JWT ใส่ Role จาก DB
			expirationTime := time.Now().Add(24 * time.Hour)
			claims := &MyClaims{
				UserID: user.ID,
				Role:   user.Role, // สำคัญ: คืน role จาก DB
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(expirationTime),
					IssuedAt:  jwt.NewNumericDate(time.Now()),
				},
			}

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			tokenString, err := token.SignedString(jwtKey)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, "could not create token")
				return
			}

			// 3) ส่ง AuthResponse กลับไป
			writeJSON(w, http.StatusOK, AuthResponse{
				Token: tokenString,
				Role:  user.Role, // ส่ง role ให้ FE ใช้นำทาง
			})
		})
	})

	return r
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}
func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)
	repl := strings.NewReplacer(" ", "_", "/", "-", "\\", "-", ":", "-", "|", "-")
	return repl.Replace(s)
}

// Helper function envOr ที่ถูกคัดลอกมาจาก main.go เพื่อให้ router.go สามารถใช้งานได้
func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
