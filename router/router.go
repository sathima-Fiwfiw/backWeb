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
)

type App struct {
	Users      *models.UserModel
	UploadDir  string
	AllowOrigin string // CORS Origin
}

func New(app *App) http.Handler {
	// ensure upload dir
	if app.UploadDir == "" {
		app.UploadDir = "uploads/avatars"
	}
	_ = os.MkdirAll(app.UploadDir, 0755)

	r := chi.NewRouter()

	// --- CORS แบบง่าย ใช้กับ Angular ---
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			origin := app.AllowOrigin
			if origin == "" {
				origin = "*" // ปลอดภัยกว่าคือใส่ http://localhost:4200
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
		// POST /api/register  (multipart/form-data)
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
				// เส้นทาง public (ปรับตามจริง ถ้าเสิร์ฟเป็น static ให้ชี้ URL)
				avatarURL = "/" + filepath.ToSlash(dstPath)
			}

			u, err := app.Users.Create(r.Context(), username, email, password, avatarURL)
			if err != nil {
				writeErr(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusCreated, u)
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
