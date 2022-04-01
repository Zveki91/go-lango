package main

import (
	"database/sql"
	"github.com/hako/branca"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"social/internal/handlers"
	"social/internal/services"
)

func main() {
	var (
		port        = env("PORT", "3005")
		databaseURL = env("DATABASE_URL", "host=localhost port=5432 user=postgres password=postgres dbname=nakama sslmode=disable")
		origin      = env("ORIGIN", "http://localhost:"+port)
		brancaKey   = env("BRANCA_KEY", "YEk9b2KT7Hv6bYuthSzckXKkqkYZawhq")
	)

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("could not connect to db : %s", err)
		return
	}

	defer db.Close()
	if err = db.Ping(); err != nil {
		log.Fatalf("could not connect to db : %s", err)
		return
	}

	codec := branca.NewBranca(brancaKey)
	codec.SetTTL(uint32(services.TokenLifeSpan.Seconds()))
	s := services.New(db, codec, origin)

	h := handlers.New(s)
	log.Printf("app running on port %s", port)
	if err := http.ListenAndServe(":"+port, h); err != nil {
		log.Fatalf("could not start app")
	}
}

func env(key, fallbackValue string) string {
	s := os.Getenv(key)
	if s == "" {
		return fallbackValue
	}

	return s
}
