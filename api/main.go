package main

import (
	"backWeb/database"
	"backWeb/models"
	"backWeb/router"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv" // เพิ่ม
)

func main() {
	_ = godotenv.Load() // โหลด .env ถ้ามี

	db := database.MustOpen()
	defer db.Close()

	userModel := &models.UserModel{DB: db}
	app := &router.App{
		Users:       userModel,
		UploadDir:   envOr("UPLOAD_DIR", "uploads/avatars"),
		AllowOrigin: os.Getenv("CORS_ORIGIN"),
	}
	r := router.New(app)

	addr := envOr("HTTP_ADDR", ":8080")
	log.Println("listening on", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
