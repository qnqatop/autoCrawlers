package db

import (
	"fmt"
	"strings"
	"time"
)

type Brand struct {
	ID         int       `pg:"id,pk"`              // Первичный ключ
	Name       string    `pg:"name,notnull"`       // Название бренда
	ExternalID string    `pg:"external_id,unique"` // Внешний ID
	Source     string    `pg:"source"`             // Источник данных
	CreatedAt  time.Time `pg:"created_at"`         // Дата создания
	UpdatedAt  time.Time `pg:"updated_at"`         // Дата обновления
}

type Brands []Brand

func (bb Brands) ToMap() map[int]Brand {
	bbm := make(map[int]Brand)

	for _, b := range bb {
		bbm[b.ID] = b
	}
	return bbm
}

func (bb Brands) NameWithId() map[string]int {
	bbm := make(map[string]int)

	for _, b := range bb {
		bbm[strings.ToLower(b.Name)] = b.ID
	}
	return bbm
}

type Models []Model

func (mm Models) NameWithId() map[string]int {
	mmm := make(map[string]int)

	for _, m := range mm {
		mmm[fmt.Sprintf("%d%s", m.BrandID, strings.ToLower(m.Name))] = m.ID
	}
	return mmm
}

type Model struct {
	ID         int       `pg:"id,pk"`              // Первичный ключ
	Name       string    `pg:"name,notnull"`       // Название модели
	BrandID    int       `pg:"brand_id,notnull"`   // Внешний ключ на brands
	ExternalID string    `pg:"external_id,unique"` // Внешний ID
	CreatedAt  time.Time `pg:"created_at"`         // Дата создания
	UpdatedAt  time.Time `pg:"updated_at"`         // Дата обновления
}

type Car struct {
	ID        int       `pg:"id"`               // Часть составного ключа (id, brand_id)
	BrandID   int       `pg:"brand_id,pk"`      // Часть составного ключа и ключ партиционирования
	ModelID   int       `pg:"model_id,notnull"` // Внешний ключ на models
	Data      string    `pg:"data,type:jsonb"`  // JSONB поле
	IsActive  bool      `pg:"is_active"`
	CreatedAt time.Time `pg:"created_at"`
	UpdatedAt time.Time `pg:"updated_at"`
}
