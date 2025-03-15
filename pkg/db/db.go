package db

import (
	"hash/crc64"

	"qnqa-auto-crawlers/pkg/logger"

	"github.com/go-pg/pg/v10"
)

type DB struct {
	*pg.DB
	logger.Logger

	crcTable *crc64.Table
}

func New(db *pg.DB, log logger.Logger) *DB {
	return &DB{db, log, crc64.MakeTable(crc64.ECMA)}
}
