package mobilede

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

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

func (c *Crawler) modelParse(ctx context.Context, b *db.Brand) error {
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
//func (c *Crawler) PageParse(task rabbitmq.Task) error {
//	//const op = "app.crawlers.MobileDe.itemCrawler.ItemParse"
//	//var imageArray []string
//	//var splitColor []string
//	//var valueArray []string
//	//var keyArray []string
//	//keyValueTechDaten := make(map[string]string)
//	//auto := make(map[string]string)
//	//car := models.Auto{}
//	//address := models.Location{}
//	//country := models.Country{}
//	//dealer := models.Dealer{}
//	//price := models.Price{}
//	//currency := models.Currency{}
//	//
//	//u := "https://m.mobile.de" + task.Parse.Path
//	//
//	//lockOk, lockWhen, meta := carService2.MobileDEIdLockMap.TryLockKeyMeta(fmt.Sprint(extId))
//	//if !lockOk {
//	//	dt := time.Since(lockWhen)
//	//	log.WithFields(log.Fields{
//	//		"adLink": path,
//	//		"dt":     dt,
//	//	}).Warn("Машина уже парсится в другом потоке")
//	//}
//	//defer carService2.MobileDEIdLockMap.UnlockKey(fmt.Sprint(extId))
//	//_ = meta
//	//// meta.Phase = "можно расписывать фазы парсинга"
//	//
//	//exists, err := config.Redis.Exists(ctx, u).Result()
//	//if err != nil {
//	//	c.logger.Errorff("Ошибка проверки существования ключа: %v", err)
//	//}
//	//
//	//if exists == 1 {
//	//	log.Info("Машина уже была скипнута ранее")
//	//	return nil
//	//}
//	//ok := carService.CheckCarLinkInDb(fmt.Sprint(extId))
//	//if ok {
//	//	log.WithField("id", extId).Info("Машина уже есть в базе)")
//	//	return nil
//	//}
//	//
//	//px := utils.GetArrayProxy()
//	//
//	//c := colly.NewCollector(
//	//	colly.AllowedDomains("suchen.mobile.de", "m.mobile.de", "www.mobile.de"),
//	//	colly.UserAgent("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 Safari/537.36"),
//	//	colly.IgnoreRobotsTxt(),
//	//)
//	//
//	//c.SetProxyFunc(func(r *http.Request) (*url.URL, error) {
//	//	proxy := fmt.Sprintf("http://%s", px[rand.Intn(len(px))])
//	//	return url.Parse(proxy)
//	//})
//	//
//	//c.OnRequest(func(r *colly.Request) {
//	//	r.Headers.Set("user-agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 Safari/537.36")
//	//	r.Headers.Set("accept-language", "en")
//	//})
//	//
//	//c.OnResponse(func(r *colly.Response) {
//	//	log.Debug("Status:", r.StatusCode)
//	//})
//	//
//	////ключи по разделу TECH DATEN
//	//c.OnXML("//*[@data-testid=\"vip-technical-data-box\"]//dt", func(e *colly.XMLElement) {
//	//	keyArray = append(keyArray, e.Text)
//	//})
//	//
//	////значениея по разделу TECH DATEN
//	//c.OnXML("//*[@data-testid=\"vip-technical-data-box\"]//dd", func(e *colly.XMLElement) {
//	//	valueArray = append(valueArray, e.Text)
//	//})
//	//
//	////цена
//	//c.OnXML("//*[@data-testid=\"vip-price-box\"]/section/div/div/span", func(e *colly.XMLElement) {
//	//	re_cost := regexp.MustCompile(`\d+.\d+`)
//	//	cost := re_cost.FindString(e.Text)
//	//	price.Value, _ = strconv.Atoi(strings.Replace(cost, ".", "", 1))
//	//	currency.Symbol = "€"
//	//	currency.Name = "EUR"
//	//
//	//})
//	//
//	////улица, номер дома
//	//c.OnXML("//*[@id='db-address']", func(e *colly.XMLElement) {
//	//	a := e.Text
//	//	re_index := regexp.MustCompile(`\S\S-`)
//	//	start_index := re_index.FindString(a)
//	//	stringStreetAndNumber := strings.Split(a, start_index)[0]
//	//
//	//	re_number := regexp.MustCompile(`\d+`)
//	//	number := re_number.FindString(stringStreetAndNumber)
//	//	re_street := regexp.MustCompile(`\D+`)
//	//	street := re_street.FindString(stringStreetAndNumber)
//	//	address.Street = strings.Map(RemoveHiddenChars, street)
//	//	address.Num = strings.Map(RemoveHiddenChars, number)
//	//})
//	//
//	////имя диллера
//	//c.OnXML("//*[@data-testid=\"vip-dealer-box-content\"]/div/div[1]/div[1]/div[1]", func(e *colly.XMLElement) {
//	//	dealer.Name = strings.Map(RemoveHiddenChars, e.Text)
//	//})
//	//
//	////индекс, город, страна
//	//c.OnXML("//*[@data-testid=\"vip-dealer-box-seller-address2\"]", func(e *colly.XMLElement) {
//	//	re := regexp.MustCompile(`\D+`)
//	//	indexTown := re.FindAllString(e.Text, 4)
//	//	reIndex := regexp.MustCompile(`\D+\d+`)
//	//	index := reIndex.FindString(e.Text)
//	//	town := indexTown[1]
//	//	town = strings.Replace(town, " ", "", 1)
//	//	reCountry := regexp.MustCompile(`[A-Z]+`)
//	//	cName := reCountry.FindString(index)
//	//
//	//	address.Zip = strings.Map(RemoveHiddenChars, index)
//	//	address.City = strings.Map(RemoveHiddenChars, town)
//	//	country.Name = strings.Map(RemoveHiddenChars, cName)
//	//})
//	//
//	//c.OnXML("//title[@data-rh=\"true\"]", func(e *colly.XMLElement) {
//	//	re := regexp.MustCompile(`\S+`)
//	//	brandModel := re.FindAllString(e.Text, 2)
//	//
//	//	auto["brand_name"] = brandModel[0]
//	//	auto["model_name"] = brandModel[1]
//	//})
//	//
//	////все фото
//	//c.OnXML("//*[starts-with(@data-testid, 'thumbnail-image')][@src]", func(e *colly.XMLElement) {
//	//
//	//	a := e.Attr("src")
//	//	imageArray = append(imageArray, a)
//	//})
//	//
//	////ошибка парсинка страницы
//	//c.OnError(func(r *colly.Response, err error) {
//	//	log.Debug("Request URL:", r.Request.URL, "\nError:", err)
//	//})
//	//
//	////показывает на какой странице остановился
//	//c.OnScraped(func(r *colly.Response) {
//	//	log.Debug("Finished ", r.Request.URL)
//	//})
//	//
//	//err = c.Visit(u)
//	//
//	//logCtx := log.WithFields(log.Fields{
//	//	"brand": auto["brand_name"],
//	//	"model": auto["model_name"],
//	//	"url":   u,
//	//	"id":    extId,
//	//	"func":  op,
//	//})
//	//
//	//if !containsCountry(country.Name) {
//	//	return nil
//	//}
//	//if err != nil {
//	//	e := config.Redis.Set(ctx, u, "err", 0).Err()
//	//	if e != nil {
//	//		logCtx.WithError(err).Error(e)
//	//	}
//	//	logCtx.WithError(err).Error("MOBILEDE skip")
//	//	return nil
//	//}
//	//
//	//if len(imageArray) <= 14 {
//	//	err = fmt.Errorf("MOBILEDE skip len(carImagesPath) < 15 count (%d) , brandName: %s", len(imageArray), auto["brand_name"])
//	//	e := config.Redis.Set(ctx, u, "err", 0).Err()
//	//	if e != nil {
//	//		logCtx.WithError(err).Error(e)
//	//	}
//	//	logCtx.WithError(err).Error(err)
//	//	return nil
//	//}
//	//
//	////создаем мапу по характеристикам
//	//i := 0
//	//for _, key := range keyArray {
//	//	keyValueTechDaten[key] = valueArray[i]
//	//	i++
//	//}
//	//
//	//if price.Value == 0 {
//	//	e := config.Redis.Set(ctx, u, "err", 0).Err()
//	//	if e != nil {
//	//		logCtx.WithError(err).Error(e)
//	//	}
//	//	logCtx.WithError(err).Error("MOBILEDE skip May be leasing")
//	//	return nil
//	//}
//	//
//	//isElectro := isElektroType(keyValueTechDaten)
//	//isHybrid := isHybridType(keyValueTechDaten)
//	//isHydrogen := isHydrogenType(keyValueTechDaten)
//	//if !isElectro {
//	//	if EnginePower(keyValueTechDaten["Leistung"]) == "" && FuelType(keyValueTechDaten["Kraftstoffart"]) != "3" {
//	//		err = fmt.Errorf("MOBILEDE skipEnginePower == 0 ")
//	//		e := config.Redis.Set(ctx, u, "err", 0).Err()
//	//		if e != nil {
//	//			logCtx.WithError(err).Error(e)
//	//		}
//	//		logCtx.WithError(err).Error(err)
//	//		return nil
//	//	}
//	//	if FuelType(keyValueTechDaten["Kraftstoffart"]) == "7" {
//	//		e := config.Redis.Set(ctx, u, "MOBILEDE skip This type of fuel is not suitable for machine parsing", 0).Err()
//	//		if e != nil {
//	//			logCtx.WithError(err).Error(e)
//	//		}
//	//		logCtx.WithError(err).Error("MOBILEDE skip This type of fuel is not suitable for machine parsing")
//	//		return nil
//	//	}
//	//	if EngineVolume(keyValueTechDaten["Hubraum"]) == 0 && FuelType(keyValueTechDaten["Kraftstoffart"]) != "3" {
//	//		err = fmt.Errorf("MOBILEDE skip EngineVolume == 0 ")
//	//		e := config.Redis.Set(ctx, u, "err", 0).Err()
//	//		if e != nil {
//	//			logCtx.WithError(err).Error(e)
//	//		}
//	//		logCtx.WithError(err).Error(err)
//	//		return nil
//	//	}
//	//}
//	//
//	//mileage, _ := strconv.Atoi(Kilometre(keyValueTechDaten["Kilometerstand"]))
//	//if mileage >= 70000 {
//	//	e := config.Redis.Set(ctx, u, "MOBILEDE skip car mobilede mileage > 70000", 0).Err()
//	//	if e != nil {
//	//		logCtx.WithError(err).Error(e)
//	//	}
//	//	logCtx.WithError(err).Error("MOBILEDE skip car mobilede mileage > 70000")
//	//	return nil
//	//}
//	//err = d.Builder.SetBrandId(auto["brand_name"])
//	//if err != nil {
//	//	e := config.Redis.Set(ctx, u, "MOBILEDE skip Нет бренда в базе для машины", 0).Err()
//	//	if e != nil {
//	//		logCtx.WithError(err).Error(e)
//	//	}
//	//	logCtx.WithError(err).Error("MOBILEDE skip Нет бренда в базе для машины")
//	//	return nil
//	//}
//	//err = d.Builder.SetModel(auto["model_name"])
//	//if err != nil {
//	//	e := config.Redis.Set(ctx, u, "MOBILEDE skip %s \nНет бренда в базе для машины url", 0).Err()
//	//	if e != nil {
//	//		logCtx.WithError(err).Error(e)
//	//	}
//	//	logCtx.WithError(err).Error("MOBILEDE skip Нет бренда в базе для машины ")
//	//	return nil
//	//}
//	//err = d.Builder.SetConfiguration(strings.ToUpper(keyValueTechDaten["Kategorie"]))
//	//if err != nil {
//	//	e := config.Redis.Set(ctx, u, "MOBILEDE skip SetConfiguration", 0).Err()
//	//	if e != nil {
//	//		logCtx.WithError(err).Error(e)
//	//	}
//	//	logCtx.WithError(err).Error("MOBILEDE skip SetConfiguration")
//	//	return nil
//	//}
//	//
//	//err = d.Builder.SetCurrency(currency.Name)
//	//if err != nil {
//	//	e := config.Redis.Set(ctx, u, "MOBILEDE skip SetCurrency", 0).Err()
//	//	if e != nil {
//	//		logCtx.WithError(err).Error(e)
//	//	}
//	//	logCtx.WithError(err).Error("MOBILEDE skip SetCurrency")
//	//	return nil
//	//}
//	//err = d.Builder.SetPrice(price.Value)
//	//if err != nil {
//	//	e := config.Redis.Set(ctx, u, "MOBILEDE skip SetPrice", 0).Err()
//	//	if e != nil {
//	//		logCtx.WithError(err).Error(e)
//	//	}
//	//	logCtx.WithError(err).Error("MOBILEDE skip SetPrice")
//	//	return nil
//	//}
//	//
//	////сохранение цвета кузова и салона
//	//checkColorSalon := strings.Contains(keyValueTechDaten["Innenausstattung"], ", ")
//	//
//	//if checkColorSalon {
//	//	splitColor = strings.Split(keyValueTechDaten["Innenausstattung"], ", ")
//	//
//	//	err = d.Builder.SetColor(splitColor[1], 0)
//	//	if err != nil {
//	//		e := config.Redis.Set(ctx, u, "MOBILEDE skip SetColor", 0).Err()
//	//		if e != nil {
//	//			logCtx.WithError(err).Error(e)
//	//		}
//	//		logCtx.WithError(err).Error("MOBILEDE skip SetColor")
//	//		return nil
//	//	}
//	//	err = d.Builder.SetColor(keyValueTechDaten["Farbe"], 1)
//	//	if err != nil {
//	//		e := config.Redis.Set(ctx, u, "MOBILEDE skip SetColor", 0).Err()
//	//		if e != nil {
//	//			logCtx.WithError(err).Error(e)
//	//		}
//	//		logCtx.WithError(err).Error("MOBILEDE skip SetColor")
//	//		return nil
//	//	}
//	//} else {
//	//	err = d.Builder.SetColor(keyValueTechDaten["Farbe"], 1)
//	//	if err != nil {
//	//		e := config.Redis.Set(ctx, u, "MOBILEDE skip SetColor", 0).Err()
//	//		if e != nil {
//	//			logCtx.WithError(err).Error(e)
//	//		}
//	//		logCtx.WithError(err).Error("MOBILEDE skip SetColor")
//	//		return nil
//	//	}
//	//}
//	//
//	//err = d.Builder.SetAddress(address.City, country.Name)
//	//if err != nil {
//	//	e := config.Redis.Set(ctx, u, "MOBILEDE skip SetAddress", 0).Err()
//	//	if e != nil {
//	//		logCtx.WithError(err).Error(e)
//	//	}
//	//	logCtx.WithError(err).Error("MOBILEDE skip SetAddress")
//	//	return nil
//	//}
//	//
//	//err = d.Builder.SetDealer(dealer.Name)
//	//if err != nil {
//	//	e := config.Redis.Set(ctx, u, "MOBILEDE skip SetDealer", 0).Err()
//	//	if e != nil {
//	//		logCtx.WithError(err).Error(e)
//	//	}
//	//	logCtx.WithError(err).Error("MOBILEDE skip SetDealer")
//	//	return nil
//	//}
//	//
//	////расчет двигателя в литрах
//	//
//	//if mileage <= 1000 {
//	//	car.Mileage = mileage
//	//	car.OwnersCount = 0
//	//	car.Type = 0
//	//} else {
//	//	car.Mileage = mileage
//	//	car.OwnersCount, _ = strconv.Atoi(keyValueTechDaten["Anzahl der Fahrzeughalter"])
//	//	car.Type = 1
//	//}
//	//
//	//literEngineVolume := float64(EngineVolume(keyValueTechDaten["Hubraum"])) / 1000
//	//car.EnginePower, _ = strconv.Atoi(EnginePower(keyValueTechDaten["Leistung"]))
//	//car.FirstRegDate = FirstRegDate(strings.Replace(keyValueTechDaten["Erstzulassung"], "/", "-", 1))
//	//car.Year = car.FirstRegDate.Year()
//	//
//	//tType, _ := strconv.Atoi(TransmissionType(keyValueTechDaten["Getriebe"]))
//	//fType, _ := strconv.Atoi(FuelType(keyValueTechDaten["Kraftstoffart"]))
//	//eType, _ := strconv.Atoi(EcoType(keyValueTechDaten["Umweltplakette"]))
//	//name := auto["brand_name"] + " " + auto["model_name"]
//	//engineVol := EngineVolume(keyValueTechDaten["Hubraum"])
//	//if isElectro {
//	//	fType = 3
//	//	literEngineVolume = 0
//	//	engineVol = 0
//	//}
//	//if isHybrid {
//	//	fType = 2
//	//}
//	//if isHydrogen && !isHybrid {
//	//	fType = 6
//	//}
//	//
//	//err = d.Builder.SetImages(imageArray)
//	//if err != nil {
//	//	logCtx.WithError(err).Error("MOBILEDE skip SetImages")
//	//	return nil
//	//}
//	//err = d.Builder.CreateCar(
//	//	u, name, strconv.Itoa(extId), Round(literEngineVolume, 1), engineVol, car.Year,
//	//	tType, fType, mileage, car.OwnersCount, car.Type, eType, car.EnginePower, "", car.FirstRegDate,
//	//)
//	//
//	//if err != nil {
//	//	logCtx.WithError(err).Error("MOBILEDE skip CreateCar")
//	//	return nil
//	//}
//	//
//	//return nil
//	return nil
//}

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
