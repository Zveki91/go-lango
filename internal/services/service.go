package services

import (
	"database/sql"
	"github.com/hako/branca"
)

// Service contains core logic
type Service struct {
	Db     *sql.DB
	Codec  *branca.Branca
	Origin string
}

func New(db *sql.DB, cdc *branca.Branca, origin string) *Service {
	return &Service{
		db,
		cdc,
		origin,
	}
}
