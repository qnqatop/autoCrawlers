package mobilede

import (
	"encoding/json"
	"strings"
	"time"

	"qnqa-auto-crawlers/pkg/crawlers"
)

type ModelsJSON struct {
	Data []DataItem `json:"data"`
}

type DataItem struct {
	Value         string `json:"value"`
	Label         string `json:"label"`
	OptgroupLabel string `json:"optgroupLabel"`
	Items         []InternalItem
}
type InternalItem struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type ListParseResponse struct {
	HasNextPage bool   `json:"hasNextPage"`
	Items       []Item `json:"items"`
}

type Item struct {
	IsEyeCatcher bool   `json:"isEyeCatcher"`
	NumImages    int    `json:"numImages"`
	RelativePath string `json:"relativeUrl"`
	Id           int    `json:"id"`
}

// Response представляет стандартный ответ API
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type ListParseTask struct {
	Url string `json:"url"`
}

func (lpt *ListParseTask) Model(data interface{}) error {
	b, err := json.Marshal(lpt)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &data)
	return err
}

func (lpt *ListParseTask) Byte() []byte {
	b, err := json.Marshal(lpt)
	if err != nil {
		return nil
	}
	return b
}

type CarParseTask struct {
	RelativePath string `json:"url"`
	ExternalId   int    `json:"externalId"`
}

func (cpt *CarParseTask) Model(data interface{}) error {
	b, err := json.Marshal(cpt)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &data)
	return err
}

func (cpt *CarParseTask) Byte() []byte {
	b, err := json.Marshal(cpt)
	if err != nil {
		return nil
	}
	return b
}

type Auto struct {
	ExternalID        string    `json:"externalId"`
	Year              int       `json:"year"`
	OwnersCount       int       `json:"ownersCount"`
	Mileage           int       `json:"mileage"`
	AdLink            string    `json:"adLink"`
	FuelType          string    `json:"fuelType"`
	TransmissionType  string    `json:"transmissionType"`
	EcoType           string    `json:"eco_type"`
	EnginePower       int       `json:"engine_power"`
	EngineVolume      int       `json:"engine_volume"`
	Warranty          int       `json:"warranty"`
	Dealer            Dealer    `json:"dealer"`
	Country           Country   `json:"country"`
	Price             Price     `json:"price"`
	Currency          Currency  `json:"currency"`
	Model             string    `json:"model_name"`
	Brand             string    `json:"brand_name"`
	Location          Location  `json:"location"`
	Name              string    `json:"name"`
	FirstRegDate      time.Time `json:"first_reg_date"`
	LiterEngineVolume float64   `json:"liter_engine_volume"`
	Age               int       `json:"age"`
	Vin               *string   `json:"vin"`
	ExternalUrl       string    `json:"externalUrl"`
	Colors            []Color   `json:"colors"`
	OtherData         Data      `json:"otherData"`
}

type Color struct {
	Type string `json:"type"`
	Name string `json:"name"`
}
type Location struct {
	City               string `json:"city"`
	Street             string `json:"street"`
	Num                string `json:"num"`
	Zip                string `json:"zip"`
	CountryID          string `json:"country"`
	Floor              string `json:"floor"`
	Comment            string `json:"comment"`
	IsDefaultInCountry bool   `json:"is_default_in_country"`
}

type Country struct {
	Name          string `json:"name"`
	IsCustomUnion bool   `json:"is_custom_union"`
	IsCreateAllow bool   `json:"is_create_allow"`
	FullNameRu    string `json:"full_name_ru"`
	FullNameEn    string `json:"full_name_en"`
}

type Dealer struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Type    string `json:"type"`
}

type Price struct {
	Value      int  `json:"value"`
	CurrencyId uint `json:"currency_id"`
	IsMain     bool `json:"is_main"`
}

type Currency struct {
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
}

type Data map[string]string

func NewData() Data {
	return make(Data)
}

func (k Data) FuelType() string {
	switch {
	case k.isElectric():
		return crawlers.ElectricType
	case k.isHybridType():
		return crawlers.HybridType
	case k.isHydrogenType():
		return crawlers.HydrogenType
	default:
		return crawlers.PetrolType
	}
}

func (k Data) isElectric() bool {
	if k["Anderer Energieträger"] == "Strom" && k["Antriebsart"] == "Elektromotor" && !strings.Contains(k["Kraftstoffart"], "Hybrid") {
		return true
	}
	return false
}

func (k Data) isHybridType() bool {
	return strings.Contains(k["Kraftstoffart"], "Hybrid")
}

func (k Data) isHydrogenType() bool {
	return strings.Contains(k["Anderer Energieträger"], "Wasserstoff")
}
