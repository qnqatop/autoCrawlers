package crawlers

import "context"

type (
	// Crawler осовной интерфейс для краулеров
	Crawler interface {
		PageParse(ctx context.Context, task Task) error
		ListParse(ctx context.Context) error
		ModelParse(ctx context.Context, brandId string) error
		BrandParse(ctx context.Context) error
	}

	Task struct {
		Parse *Parse
		Err   error
	}
	Parse struct {
		Source string
		Path   string
		ID     string
		Brand  string
	}
)
