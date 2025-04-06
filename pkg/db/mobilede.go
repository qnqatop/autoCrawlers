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
		OnConflict("(external_id, name) DO UPDATE").
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

func (mde *MobileDeRepo) AllBrands(ctx context.Context) (Brands, error) {
	var brands Brands
	err := mde.db.ModelContext(ctx, &brands).Select()
	if err != nil {
		return nil, err
	}
	return brands, err
}

func (mde *MobileDeRepo) AllModels(ctx context.Context) (Models, error) {
	var models Models
	err := mde.db.ModelContext(ctx, &models).Select()
	if err != nil {
		return nil, err
	}
	return models, err
}

func (mde *MobileDeRepo) AllMs(ctx context.Context) ([]string, error) {
	var brands Brands
	err := mde.db.ModelContext(ctx, &brands).Select()
	if err != nil {
		return nil, err
	}

	var models []Model
	err = mde.db.ModelContext(ctx, &models).Select()
	if err != nil {
		return nil, err
	}

	bbm := brands.ToMap()

	mss := make([]string, 0, len(models))
	for _, m := range models {
		if res, ok := bbm[m.BrandID]; ok {
			mss = append(mss, fmt.Sprintf("%s;%s;;", res.ExternalID, m.ExternalID))
		}
	}

	return mss, nil
}
