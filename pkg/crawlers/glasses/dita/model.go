package dita

import (
	"encoding/json"
	"time"
)

// Task представляет задачу для краулера
type Task struct {
	URL string
}

// Model реализует интерфейс Tasker
func (t *Task) Model(v interface{}) error {
	if url, ok := v.(*string); ok {
		*url = t.URL
		return nil
	}
	return nil
}

type PageResponse struct {
	Product Product
	Details ProductDetails
	Url     string
}

// Product представляет структуру продукта из JSON API
type Product struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Price       int       `json:"price"`
	Variants    []Variant `json:"variants"`
}

// Variant представляет вариант продукта
type Variant struct {
	ID        int    `json:"id"`
	SKU       string `json:"sku"`
	Available bool   `json:"available"`
	Option1   string `json:"option1"` // Цвет
	Option2   string `json:"option2"` // Линзы
}

// ProductDetails представляет детали продукта с сайта
type ProductDetails struct {
	FrameDimensions string
	FrameWidth      string
	FrameHeight     string
	Temple          string
	Bridge          string
	FrameMaterial   string
	LensWidth       string
	LensHeight      string
	LensCurve       string
	LensMaterial    string
	Polarized       string
	AntiReflective  string
	Images          []string
}

// CSVProduct представляет структуру продукта для CSV файла
type CSVProduct struct {
	Title           string    `csv:"Название"`
	Price           float64   `csv:"Цена"`
	Color           string    `csv:"Цвет"`
	Lenses          string    `csv:"Линзы"`
	SKU             string    `csv:"Артикул (SKU)"`
	Available       string    `csv:"Доступность"`
	Description     string    `csv:"Описание"`
	ImageURL        string    `csv:"Ссылка на изображение"`
	FrameDimensions string    `csv:"Frame Dimensions"`
	FrameWidth      string    `csv:"Frame Width"`
	FrameHeight     string    `csv:"Frame Height"`
	Temple          string    `csv:"Temple"`
	Bridge          string    `csv:"Bridge"`
	FrameMaterial   string    `csv:"Frame Material"`
	LensWidth       string    `csv:"Lens Width"`
	LensHeight      string    `csv:"Lens Height"`
	LensCurve       string    `csv:"Lens Curve"`
	LensMaterial    string    `csv:"Lens Material"`
	Polarized       string    `csv:"Polarized"`
	AntiReflective  string    `csv:"Anti-Reflective"`
	URL             string    `csv:"Ссылка на товар"`
	CreatedAt       time.Time `csv:"created_at"`
}

type ListParse struct {
	Url   string `json:"url"`
	Pages int    `json:"page"`
}

func (lpt *ListParse) Model(data interface{}) error {
	b, err := json.Marshal(lpt)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &data)
	return err
}

func (lpt *ListParse) Byte() []byte {
	b, err := json.Marshal(lpt)
	if err != nil {
		return nil
	}
	return b
}

type BaseParse struct {
	Url string `json:"url"`
}

func (lpt *BaseParse) Model(data interface{}) error {
	b, err := json.Marshal(lpt)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &data)
	return err
}

func (lpt *BaseParse) Byte() []byte {
	b, err := json.Marshal(lpt)
	if err != nil {
		return nil
	}
	return b
}

type ProductLinksParse struct {
	Urls []string `json:"urls"`
}

func (lpt *ProductLinksParse) Model(data interface{}) error {
	b, err := json.Marshal(lpt)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &data)
	return err
}

func (lpt *ProductLinksParse) Byte() []byte {
	b, err := json.Marshal(lpt)
	if err != nil {
		return nil
	}
	return b
}
