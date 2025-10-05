package database

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func MustOpen() *sql.DB {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		log.Fatal("missing env MYSQL_DSN")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("open DB error: ", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(60 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatal("db ping error: ", err)
	}
	return db
}
