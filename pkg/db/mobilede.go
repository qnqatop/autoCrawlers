package db

import (
	"context"
	"fmt"
)

type MobileDeRepo struct {
	db *DB
}

func NewMobileDERepo(db *DB) *MobileDeRepo {
	return &MobileDeRepo{db}
}

func (mde *MobileDeRepo) SaveBrand(ctx context.Context, brand *Brand) error {
	_, err := mde.db.ModelContext(ctx, brand).
		OnConflict("(source, external_id, name) DO UPDATE").
		Set("updated_at = NOW()"). // Обновляем только updated_at, если конфликт
		Insert()
	return err
}

func (mde *MobileDeRepo) SaveModel(ctx context.Context, model *Model) error {
	_, err := mde.db.ModelContext(ctx, model).
		Insert()
	return err
}

func (mde *MobileDeRepo) SaveAuto(ctx context.Context, car *Car) error {
	// Сохраняем автомобиль
	_, err := mde.db.Model(car).Insert()
	if err != nil {
		return fmt.Errorf("failed to save car: %w", err)
	}
	return nil
}

func (mde *MobileDeRepo) AllBrands(ctx context.Context) ([]*Brand, error) {
	var brands []*Brand
	err := mde.db.ModelContext(ctx, &brands).Select()
	if err != nil {
		return nil, err
	}
	return brands, err
}
