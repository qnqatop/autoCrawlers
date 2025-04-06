package crawlers

import (
	"context"
)

const (
	ElectricType = "electric"
	HybridType   = "hybrid"
	HydrogenType = "hydrogen"
	PetrolType   = "petrol"
)

type (
	// Crawlerer осовной интерфейс для краулеров
	Crawlerer interface {
		PageParse(ctx context.Context, task Tasker) error
		ListParse(ctx context.Context, task Tasker) error
		ModelParse(ctx context.Context) error
		BrandParse(ctx context.Context) error
	}

	// Tasker основной интерфейс для тасок краулера
	Tasker interface {
		Model(data interface{}) error
		Byte() []byte
	}
)
