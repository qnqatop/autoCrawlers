package mobilede

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"qnqa-auto-crawlers/pkg/translate"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"qnqa-auto-crawlers/pkg/crawlers"
	"qnqa-auto-crawlers/pkg/db"
	"qnqa-auto-crawlers/pkg/limitgroup"
	"qnqa-auto-crawlers/pkg/logger"
	"qnqa-auto-crawlers/pkg/proxy"
	"qnqa-auto-crawlers/pkg/rabbitmq"

	"github.com/gocolly/colly/v2"
)

const (
	baseListUrl = "https://m.mobile.de/consumer/api/search/srp/items?page=1&page.size=20&url="
	baseFilter  = "/auto/search.html?lang=en&damageUnrepaired=NO_DAMAGE_UNREPAIRED&q=Unfallfrei&fr=2018:&ml=:20000&ms=%s"
	// countCarUrl = "https://m.mobile.de/consumer/api/search/hit-count?dam=false&fr=2018:&ml=:20000&ms=%s&ref=quickSearch&sb=rel&vc=Car"
)

type Crawler struct {
	logger    logger.Logger
	collector *colly.Collector
	repo      *db.MobileDeRepo
	rabbitmq  *rabbitmq.Client
	balancer  *proxy.Balancer
}

func NewCrawler(logger logger.Logger, repo *db.MobileDeRepo, rmq *rabbitmq.Client) *Crawler {
	collector := colly.NewCollector(
		colly.AllowedDomains("suchen.mobile.de", "m.mobile.de", "www.mobile.de", "mobile.de"),
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36"),
		colly.IgnoreRobotsTxt(),
	)

	_ = collector.Limit(&colly.LimitRule{
		DomainGlob:  "*.mobile.de",
		Parallelism: 10,
		Delay:       100 * time.Millisecond,
		RandomDelay: 50 * time.Millisecond,
	})

	collector.SetRequestTimeout(30 * time.Second)
	c := &Crawler{
		logger:    logger,
		collector: collector,
		repo:      repo,
		rabbitmq:  rmq,
		balancer:  proxy.NewBalancer(),
	}

	proxyCount, err := c.balancer.Load()
	if err != nil {
		c.logger.Errorf("load proxy err=%v", err)
	}

	// Если есть прокси используем их
	if proxyCount != 0 {
		rp, err := c.balancer.RoundRobinProxySwitcher()
		if err == nil {
			c.collector.SetProxyFunc(rp)
		}
	}

	go c.rabbitmq.ConsumeTasks(context.Background(), "list", c.ListParse)
	go c.rabbitmq.ConsumeTasks(context.Background(), "car", c.PageParse)
	return c
}

// BrandParse парсим бренды
func (c *Crawler) BrandParse(ctx context.Context) error {
	collector := c.collector.Clone()
	collector.SetRequestTimeout(time.Second * 30)
	collector.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		r.Headers.Set("Accept-Encoding", "gzip, deflate, br, zstd")
		r.Headers.Set("Accept-Language", "de,en-US;q=0.7,en;q=0.3")
		r.Headers.Set("Origin", "https://www.mobile.de")
		r.Headers.Set("Referer", "https://www.mobile.de/")
		r.Headers.Set("Sec-Ch-Ua", `"Chromium";v="134", "Not:A-Brand";v="24", "Google Chrome";v="134"`)
		r.Headers.Set("Sec-Ch-Ua-Mobile", "?0")
		r.Headers.Set("Sec-Ch-Ua-Platform", `"macOS"`)
		r.Headers.Set("Sec-Fetch-Dest", "document")
		r.Headers.Set("Sec-Fetch-Mode", "navigate")
		r.Headers.Set("Sec-Fetch-Site", "same-origin")
		r.Headers.Set("X-Mobile-Source-Url", "https://www.mobile.de/")
	})
	collector.OnResponse(func(r *colly.Response) {
		c.logger.Printf("Response received status - %d", r.StatusCode)
	})

	// Настраиваем обработчики для конкретной задачи
	collector.OnXML("//*[@id=\"qs-select-make\"]/optgroup[2]/option", func(e *colly.XMLElement) {
		c.logger.Printf("Found brand brand - %s , value - %s", e.Text, e.Attr("value"))
		err := c.repo.SaveBrand(ctx, &db.Brand{
			Name:       e.Text,
			ExternalID: e.Attr("value"),
			Source:     "MDE",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		})
		if err != nil {
			c.logger.Errorf("Save brand failed %v", err)
			return
		}
	})

	collector.OnError(func(r *colly.Response, err error) {
		c.logger.Errorf("Request failed %s , err - %v", r.Request.URL, err)
	})

	// Выполняем запрос
	err := collector.Visit("https://m.mobile.de")
	if err != nil {
		return err
	}
	collector.Wait()

	return nil
}

// ModelParse парсит модели для всех брендов
func (c *Crawler) ModelParse(ctx context.Context) error {
	bb, err := c.repo.AllBrands(ctx)
	if err != nil {
		return err
	}

	lg, _ := limitgroup.New(ctx, 10)
	for _, b := range bb {
		lg.Go(func() error {
			return c.modelParse(ctx, b)
		})
	}

	return lg.Wait()
}

func (c *Crawler) modelParse(ctx context.Context, b db.Brand) error {
	collector := c.collector.Clone()
	collector.OnResponse(func(r *colly.Response) {
		var data ModelsJSON
		err := json.Unmarshal(r.Body, &data)
		if err != nil {
			c.logger.Errorf("Error unmarshalling json - %v", err)
			return
		}
		for item := range data.Data {
			if data.Data[item].OptgroupLabel != "" {
				for innerItem := range data.Data[item].Items {
					if strings.Contains(data.Data[item].Items[innerItem].Label, "alle") || strings.Contains(data.Data[item].Items[innerItem].Label, "All") || strings.Contains(data.Data[item].Items[innerItem].Label, "Other") {
						continue
					} else {
						c.logger.Printf(
							"MODEL-OptgroupLabel %s:%s %s:%s",
							"NAME", data.Data[item].Items[innerItem].Label,
							"SourceExternalId", data.Data[item].Items[innerItem].Value,
						)
						err = c.repo.SaveModel(ctx, &db.Model{
							Name:       data.Data[item].Items[innerItem].Label,
							BrandID:    b.ID,
							ExternalID: data.Data[item].Items[innerItem].Value,
							CreatedAt:  time.Now(),
							UpdatedAt:  time.Now(),
						})
						if err != nil {
							c.logger.Errorf("Save model failed %s-%v", "error", err)
							return
						}
					}
				}
			} else if strings.Contains(data.Data[item].Label, "Other") {
				continue
			}
			c.logger.Printf(
				"MODEL %s:%s %s:%s",
				"NAME", data.Data[item].Label,
				"SourceExternalId", data.Data[item].Value,
			)
			err = c.repo.SaveModel(ctx, &db.Model{
				Name:       data.Data[item].Label,
				BrandID:    b.ID,
				ExternalID: data.Data[item].Value,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			})
			if err != nil {
				c.logger.Errorf("Save model failed %s - %v", "error", err)
				return
			}
		}
	})

	err := collector.Visit(fmt.Sprintf("https://m.mobile.de/consumer/api/search/reference-data/models/%s", b.ExternalID))
	if err != nil {
		return err
	}

	collector.Wait()

	return nil
}

// PageParse парсит машину по прямой ссылке
func (c *Crawler) PageParse(ctx context.Context, tasker crawlers.Tasker) error {
	collector := c.collector.Clone()
	var auto Auto
	var task CarParseTask
	err := tasker.Model(&task)
	if err != nil {
		return err
	}

	u := fmt.Sprintf("https://suchen.mobile.de/fahrzeuge/details.html?id=%d", task.ExternalId)

	//exists, err := config.Redis.Exists(ctx, u).Result()
	//if err != nil {
	//	c.logger.Errorff("Ошибка проверки существования ключа: %v", err)
	//}
	//
	//if exists == 1 {
	//	log.Info("Машина уже была скипнута ранее")
	//	return nil
	//}

	// проверять по extId наличие машины в базе
	//ok := carService.CheckCarLinkInDb(fmt.Sprint(extId))
	//if ok {
	//	log.WithField("id", extId).Info("Машина уже есть в базе)")
	//	return nil
	//}

	collector.OnRequest(func(r *colly.Request) {
		r.Headers.Set("user-agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 Safari/537.36")
		r.Headers.Set("accept-language", "en")
	})

	//ключи по разделу TECH DATEN
	var keyArray []string
	collector.OnXML("//*[@data-testid=\"vip-technical-data-box\"]//dt", func(e *colly.XMLElement) {
		keyArray = append(keyArray, e.Text)
	})

	//значениея по разделу TECH DATEN
	var valueArray []string
	collector.OnXML("//*[@data-testid=\"vip-technical-data-box\"]//dd", func(e *colly.XMLElement) {
		valueArray = append(valueArray, e.Text)
	})

	//цена
	collector.OnXML("//*[@data-testid=\"vip-price-box\"]/section/div/div/span", func(e *colly.XMLElement) {
		reCost := regexp.MustCompile(`\d+.\d+`)
		cost := reCost.FindString(e.Text)
		auto.Price.Value, _ = strconv.Atoi(strings.Replace(cost, ".", "", 1))
		auto.Currency.Symbol = "€"
		auto.Currency.Name = "EUR"

	})

	//улица, номер дома
	collector.OnXML("//*[@id='db-address']", func(e *colly.XMLElement) {
		startIndex := regexp.MustCompile(`\S\S-`).FindString(e.Text)
		stringStreetAndNumber := strings.Split(e.Text, startIndex)[0]

		number := regexp.MustCompile(`\d+`).FindString(stringStreetAndNumber)
		auto.Location.Num = strings.Map(RemoveHiddenChars, number)

		street := regexp.MustCompile(`\D+`).FindString(stringStreetAndNumber)
		auto.Location.Street = strings.Map(RemoveHiddenChars, street)
	})

	//имя диллера
	collector.OnXML("//*[@data-testid=\"vip-dealer-box-content\"]/div/div[1]/div[1]/div[1]", func(e *colly.XMLElement) {
		auto.Dealer.Name = strings.Map(RemoveHiddenChars, e.Text)
	})

	//индекс, город, страна
	collector.OnXML("//*[@data-testid=\"vip-dealer-box-seller-address2\"]", func(e *colly.XMLElement) {
		re := regexp.MustCompile(`\D+`)
		indexTown := re.FindAllString(e.Text, 4)
		reIndex := regexp.MustCompile(`\D+\d+`)
		index := reIndex.FindString(e.Text)
		town := indexTown[1]
		town = strings.Replace(town, " ", "", 1)
		reCountry := regexp.MustCompile(`[A-Z]+`)
		cName := reCountry.FindString(index)

		auto.Location.Zip = strings.Map(RemoveHiddenChars, index)
		auto.Location.City = strings.Map(RemoveHiddenChars, town)
		auto.Country.Name = strings.Map(RemoveHiddenChars, cName)
	})

	collector.OnXML("//title[@data-rh=\"true\"]", func(e *colly.XMLElement) {
		re := regexp.MustCompile(`\S+`)
		brandModel := re.FindAllString(e.Text, 2)

		auto.Brand = brandModel[0]
		auto.Model = brandModel[1]
	})

	//все фото
	var imageArray []string
	collector.OnXML("//*[starts-with(@data-testid, 'thumbnail-image')][@src]", func(e *colly.XMLElement) {
		imageArray = append(imageArray, e.Attr("src"))
	})

	//ошибка парсинка страницы
	collector.OnError(func(r *colly.Response, err error) {
		c.logger.Errorf("Request URL=%s, err=%v", r.Request.URL, err)
	})

	err = collector.Visit(u)
	if err != nil {
		c.logger.Errorf("Visit error=%v", err)
		return nil
	}

	if len(imageArray) <= 14 {
		err = fmt.Errorf("MOBILEDE skip len(carImagesPath) < 15 count (%d) , brandName: %s", len(imageArray), auto.Brand)
		c.logger.Errorf("%v", err)
		return nil
	}

	//создаем мапу по характеристикам
	i := 0
	techData := NewData()
	engData := NewData()
	for _, key := range keyArray {
		engData[translate.Translate(key)] = translate.Translate(valueArray[i])
		techData[key] = valueArray[i]
		i++
	}

	if auto.Price.Value == 0 {
		c.logger.Errorf("MOBILEDE skip May be leasing")
		return nil
	}

	auto.FuelType = techData.FuelType()

	//сохранение цвета кузова и салона
	checkColorSalon := strings.Contains(techData["Innenausstattung"], ", ")

	auto.Colors = make([]Color, 0, 2)

	if checkColorSalon {
		splitColor := strings.Split(techData["Innenausstattung"], ", ")
		// цвет салона и кузова
		auto.Colors = append(auto.Colors,
			Color{Name: splitColor[0], Type: "salon"},
			Color{Name: splitColor[1], Type: "body"},
		)
	} else {
		// цвет кузова
		auto.Colors = append(auto.Colors, Color{Name: techData["Farbe"], Type: "body"})
	}

	//расчет двигателя в литрах

	auto.Mileage, _ = strconv.Atoi(Kilometre(techData["Kilometerstand"]))

	if auto.Mileage <= 1000 {
		auto.OwnersCount = 0
	} else {
		auto.OwnersCount, _ = strconv.Atoi(techData["Anzahl der Fahrzeughalter"])
	}

	auto.LiterEngineVolume = float64(EngineVolume(techData["Hubraum"])) / 1000
	auto.EnginePower, _ = strconv.Atoi(EnginePower(techData["Leistung"]))
	auto.FirstRegDate = FirstRegDate(strings.Replace(techData["Erstzulassung"], "/", "-", 1))
	auto.Year = auto.FirstRegDate.Year()

	auto.TransmissionType = techData["Getriebe"]
	auto.EcoType = EcoType(techData["Umweltplakette"])
	auto.Name = fmt.Sprintf("%s %s", auto.Brand, auto.Model)
	auto.EngineVolume = EngineVolume(techData["Hubraum"])
	if auto.FuelType == crawlers.ElectricType {
		auto.LiterEngineVolume = 0
		auto.EngineVolume = 0
	}
	auto.OtherData = engData

	data, err := json.Marshal(auto)
	if err != nil {
		c.logger.Errorf("marshal json error=%v", err)
		return nil
	}

	err = c.repo.SaveAuto(ctx, &db.Car{
		ExternalId: task.ExternalId,
		Data:       string(data),
		IsActive:   false,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	if err != nil {
		c.logger.Errorf("save car err=%v", err)
	}

	return nil
}

// ListSearch создает таски для парсинга листов машин
func (c *Crawler) ListSearch(ctx context.Context) error {
	mss, err := c.repo.AllMs(ctx)
	if err != nil {
		return err
	}

	lgPub, _ := limitgroup.New(ctx, 2)
	for _, ms := range mss {
		// nolint
		lgPub.Go(func() error {
			err = c.rabbitmq.PublishTask(context.Background(), "list", &rabbitmq.Task{Url: generateTaskUrl(ms)})
			return err
		})
	}

	err = lgPub.Wait()

	return err
}

// ListParse парсит полученный лист с машинами и формирует таски в отдельную очередь для для PageParse
func (c *Crawler) ListParse(ctx context.Context, tasker crawlers.Tasker) error {
	collector := c.collector.Clone()
	var task ListParseTask

	err := tasker.Model(&task)
	if err != nil {
		return err
	}

	collector.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept", "*/*")
		r.Headers.Set("Accept-Encoding", "gzip, deflate, br, zstd")
		r.Headers.Set("Accept-Language", "de")
		r.Headers.Set("Content-Type", "application/json")
		r.Headers.Set("Origin", "https://www.mobile.de")
		r.Headers.Set("Referer", "https://www.mobile.de/")
		r.Headers.Set("Sec-Ch-Ua", `"Google Chrome";v="135", "Not-A.Brand";v="8", "Chromium";v="135"`)
		r.Headers.Set("Sec-Ch-Ua-Mobile", "?0")
		r.Headers.Set("Sec-Ch-Ua-Platform", `"macOS"`)
		r.Headers.Set("Sec-Fetch-Dest", "empty")
		r.Headers.Set("Sec-Fetch-Mode", "cors")
		r.Headers.Set("Sec-Fetch-Site", "same-site")
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36")
		r.Headers.Set("X-Mobile-Device-Type", "DESKTOP")
	})

	collector.OnResponse(func(r *colly.Response) {
		var data ListParseResponse
		err := json.Unmarshal(r.Body, &data)
		if err != nil {
			c.logger.Errorf("listParse mbde err=%v", err)
		}
		if data.HasNextPage {
			up, err := url.Parse(task.Url)
			if err != nil {
				c.logger.Errorf("listParse mbde url err=%v", err)
				return
			}
			qq := up.Query()
			oldPage, _ := strconv.Atoi(qq.Get("page"))
			qq.Set("page", strconv.Itoa(oldPage+1))
			up.RawQuery = qq.Encode()

			err = c.rabbitmq.PublishTask(ctx, "list", &ListParseTask{Url: up.String()})
			if err != nil {
				c.logger.Errorf("listParse mbde err=%v", err)
			}
		}

		for i, item := range data.Items {
			if item.RelativePath == "" {
				continue
			}
			err = c.rabbitmq.PublishTask(ctx, "car", &CarParseTask{RelativePath: data.Items[i].RelativePath, ExternalId: data.Items[i].Id})
			if err != nil {
				c.logger.Errorf("listParse mbde err=%v", err)
			}
		}
	})

	collector.OnError(func(r *colly.Response, err error) {
		c.logger.Errorf("Request failed %s , err - %v", r.Request.URL, err)
	})

	// Выполняем запрос
	err = collector.Visit(task.Url)
	if err != nil {
		return err
	}
	collector.Wait()

	return nil
}

// find interesting url https://m.mobile.de/consumer/api/search/reference-data/filters/Car

func generateTaskUrl(ms string) string {
	urlParams := fmt.Sprintf(baseFilter, ms)

	encodedUrlParams := url.QueryEscape(urlParams)

	return baseListUrl + encodedUrlParams
}

func RemoveHiddenChars(r rune) rune {
	if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || r == '\uFEFF' {
		return -1
	}
	return r
}

func Kilometre(str string) string {
	reKm := regexp.MustCompile(`\d+(\.\d+)?`)
	km := reKm.FindString(str)
	return strings.Replace(km, ".", "", 1)
}

func EnginePower(st string) string {
	re := regexp.MustCompile(`\d+`)
	eng := re.FindAllString(st, 2)
	if len(eng) < 2 {
		return ""
	}
	return eng[1]
}

func EngineVolume(str string) int {
	if str == "" {
		engV := 0
		return engV
	}
	eng := strings.Replace(str, "\xa0", "", 2)
	re := regexp.MustCompile(`\d+.\d+`)
	engVolume := re.FindAllString(eng, 2)
	if engVolume == nil {
		engV := 0
		return engV
	}
	if len(engVolume[0]) > 3 {
		words := strings.Split(engVolume[0], ".")
		lNumber, _ := strconv.Atoi(words[0])
		rNumber, _ := strconv.Atoi(words[1])
		engV := lNumber*1000 + rNumber
		return engV
	}
	engV, _ := strconv.Atoi(engVolume[0])
	return engV
}

func FirstRegDate(st string) time.Time {
	dt := st
	if dt == "" {
		return time.Now()
	}
	firstReg, _ := time.Parse("01-2006", dt)

	return firstReg
}

func EcoType(str string) string {
	reEt := regexp.MustCompile(`\d+`)
	eco := reEt.FindString(str)
	return eco
}
