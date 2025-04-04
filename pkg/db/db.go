package db

import (
	"context"
	"fmt"
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

func (db *DB) CheckPartitions(ctx context.Context) error {
	// Проверяем наличие брендов
	var count int
	if _, err := db.QueryOne(&count, "SELECT COUNT(*) FROM brands"); err != nil {
		return fmt.Errorf("count brnad err=%w", err)
	}

	if count == 0 {
		return fmt.Errorf("no brands found")
	}

	// Проверяем существование партиций
	exists, err := db.checkPartitionsExist(ctx)
	if err != nil {
		return fmt.Errorf("failed to check partitions err=%w", err)
	}

	if exists {
		return nil
	}

	// Создаем партиции
	if err = db.createPartitions(ctx); err != nil {
		return fmt.Errorf("failed to create partitions err=%w", err)
	}

	return nil
}

func (db *DB) checkPartitionsExist(ctx context.Context) (bool, error) {
	var count int
	_, err := db.QueryOneContext(ctx, &count, `
		SELECT COUNT(*)
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind = 'p'
		AND n.nspname = 'public'
		AND c.relname = 'cars'
	`)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (db *DB) createPartitions(ctx context.Context) error {
	var brands []Brand
	if err := db.Model(&brands).Select(); err != nil {
		return err
	}

	for _, brand := range brands {
		partitionName := fmt.Sprintf("cars_%s", brand.ExternalID)
		_, err := db.ExecOneContext(ctx, `
			CREATE TABLE IF NOT EXISTS ? PARTITION OF cars
			FOR VALUES IN (?)
		`, pg.Ident(partitionName), brand.ExternalID)
		if err != nil {
			return err
		}
	}

	return nil
}
