package main

import (
	"backWeb/database"
	"backWeb/models"
	"backWeb/router"
	"log"
	"net/http"
	"os"
)

func main() {
	// ENV ที่ควรตั้ง:
	// MYSQL_DSN="user:pass@tcp(127.0.0.1:3306)/mydb?parseTime=true&charset=utf8mb4"
	// HTTP_ADDR=":8080"
	// CORS_ORIGIN="http://localhost:4200"  (ถ้าทดสอบกับ Angular)
	// UPLOAD_DIR="uploads/avatars"

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
