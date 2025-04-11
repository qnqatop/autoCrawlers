package dita

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"qnqa-auto-crawlers/pkg/crawlers"
	"qnqa-auto-crawlers/pkg/limitgroup"
	"qnqa-auto-crawlers/pkg/logger"
	"qnqa-auto-crawlers/pkg/proxy"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
)

const (
	baseURL         = "https://dita.com"
	sunglassesURL   = baseURL + "/en-ru/collections/sunglasses/"
	opticalURL      = baseURL + "/en-ru/collections/optical"
	sunglassesPages = 3
	opticalPages    = 2
	batchSize       = 50
)

// Crawler представляет краулер для Dita
type Crawler struct {
	collector *colly.Collector
	client    *http.Client
	logger    logger.Logger
	chPD      chan PageResponse
	chLP      chan []string
	balancer  *proxy.Balancer
}

// NewCrawler создает новый экземпляр краулера
func NewCrawler(logger logger.Logger) *Crawler {
	c := colly.NewCollector(
		colly.AllowedDomains("dita.com"),
		//colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
		colly.MaxDepth(1),
		colly.DetectCharset(),
		colly.Async(true),
		colly.AllowURLRevisit(),
	)
	c.SetRequestTimeout(15 * time.Second)

	extensions.RandomUserAgent(c)
	extensions.Referer(c)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 10,
		Delay:       100 * time.Millisecond,
		RandomDelay: 300 * time.Millisecond,
	})

	cr := &Crawler{
		collector: c,
		logger:    logger,
		client:    &http.Client{Timeout: 30 * time.Second},
		chPD:      make(chan PageResponse, 400),
		chLP:      make(chan []string, 20),
		balancer:  proxy.NewBalancer(),
	}

	proxyCount, err := cr.balancer.Load("socks5")
	if err != nil {
		cr.logger.Errorf("load proxy err=%v", err)
	}

	// Если есть прокси используем их
	if proxyCount != 0 {
		rp, err := cr.balancer.RoundRobinProxySwitcher()
		if err == nil {
			cr.collector.SetProxyFunc(rp)
		}
	}

	return cr
}

func (c *Crawler) Start(ctx context.Context, tasker crawlers.Tasker) error {
	var wg sync.WaitGroup
	var productLinks []string
	var err error

	c.logger.Printf("Start parse dita")
	wg.Add(2)
	go func() {
		defer wg.Done()
		c.logger.Printf("list parse - %s", sunglassesURL)
		err = c.ListParse(ctx, &ListParse{
			Url:   sunglassesURL,
			Pages: sunglassesPages,
		})
	}()

	go func() {
		defer wg.Done()
		c.logger.Printf("list parse - %s", opticalURL)
		err = c.ListParse(ctx, &ListParse{
			Url:   opticalURL,
			Pages: opticalPages,
		})
	}()

	if err != nil {
		return fmt.Errorf("error getting optical links: %v", err)
	}

	wg.Wait()
	close(c.chLP)

	for l := range c.chLP {
		productLinks = append(productLinks, l...)
	}

	if err != nil {
		return fmt.Errorf("error getting product links: %v", err)
	}

	// Создаем CSV файл
	if err := c.Save(ctx, &ProductLinksParse{Urls: productLinks}); err != nil {
		return fmt.Errorf("error saving to CSV: %v", err)
	}

	c.logger.Printf("End parse")
	return nil
}

func (c *Crawler) ListParse(ctx context.Context, tasker crawlers.Tasker) error {
	var links []string
	var mu sync.Mutex
	var task ListParse

	err := tasker.Model(&task)
	if err != nil {
		return err
	}

	collector := c.collector.Clone()

	lg, _ := limitgroup.New(context.Background(), 3)
	for page := 1; page <= task.Pages; page++ {
		lg.Go(func() error {
			collector.OnXML("//*[contains(@class,'product-item__image-link')]", func(e *colly.XMLElement) {
				link := e.Attr("href")
				if !strings.HasPrefix(link, "http") {
					link = baseURL + link + ".js"
				}
				mu.Lock()
				defer mu.Unlock()
				links = append(links, link)
			})

			pageURL := fmt.Sprintf("%s?page=%d", task.Url, page)
			if err := collector.Visit(pageURL); err != nil {
				return fmt.Errorf("error visiting page %d: %v", page, err)
			}
			collector.Wait()
			return nil
		})
	}

	err = lg.Wait()
	if err != nil {
		return fmt.Errorf("error visiting page %d: %v", task.Pages, err)
	}

	c.chLP <- links
	return nil
}

func (c *Crawler) PageParse(ctx context.Context, tasker crawlers.Tasker) error {
	var err error
	var product Product
	var details ProductDetails
	var task BaseParse

	err = tasker.Model(&task)
	if err != nil {
		return err
	}

	// Добавляем искусственную задержку перед HTTP-запросом (100-300 мс)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(100+rand.Intn(200)) * time.Millisecond):
	}

	productCollector := c.collector.Clone()
	productCollector.OnResponse(func(r *colly.Response) {
		err = json.Unmarshal(r.Body, &product)
		if err != nil {
			err = fmt.Errorf("error unmarshalling product links: %v", err)
		}
	})
	err = productCollector.Visit(task.Url)
	if err != nil {
		return fmt.Errorf("error visiting task %s: %v", task.Url, err)
	}
	productCollector.Wait()

	collector := c.collector.Clone()
	collector.OnResponse(func(r *colly.Response) {
		if r.StatusCode >= 299 {
			_ = r.Request.Retry()
		}
	})

	// Устанавливаем обработчики для деталей продукта
	collector.OnHTML("span:contains('Frame Dimensions') + span", func(e *colly.HTMLElement) {
		details.FrameDimensions = strings.TrimSpace(e.Text)
	})
	collector.OnHTML("span:contains('Frame Width') + span", func(e *colly.HTMLElement) {
		details.FrameWidth = strings.TrimSpace(e.Text)
	})
	collector.OnHTML("span:contains('Frame Height') + span", func(e *colly.HTMLElement) {
		details.FrameHeight = strings.TrimSpace(e.Text)
	})
	collector.OnHTML("span:contains('Temple') + span", func(e *colly.HTMLElement) {
		details.Temple = strings.TrimSpace(e.Text)
	})
	collector.OnHTML("span:contains('Bridge') + span", func(e *colly.HTMLElement) {
		details.Bridge = strings.TrimSpace(e.Text)
	})
	collector.OnHTML("span:contains('Frame Material') + span", func(e *colly.HTMLElement) {
		details.FrameMaterial = strings.TrimSpace(e.Text)
	})
	collector.OnHTML("span:contains('Lens Width') + span", func(e *colly.HTMLElement) {
		details.LensWidth = strings.TrimSpace(e.Text)
	})
	collector.OnHTML("span:contains('Lens Height') + span", func(e *colly.HTMLElement) {
		details.LensHeight = strings.TrimSpace(e.Text)
	})
	collector.OnHTML("span:contains('Lens Curve') + span", func(e *colly.HTMLElement) {
		details.LensCurve = strings.TrimSpace(e.Text)
	})
	collector.OnHTML("span:contains('Lens Material') + span", func(e *colly.HTMLElement) {
		details.LensMaterial = strings.TrimSpace(e.Text)
	})
	collector.OnHTML("span:contains('Polarized') + span", func(e *colly.HTMLElement) {
		details.Polarized = strings.TrimSpace(e.Text)
	})
	collector.OnHTML("span:contains('Anti-Reflective') + span", func(e *colly.HTMLElement) {
		details.AntiReflective = strings.TrimSpace(e.Text)
	})

	// Получаем изображения
	collector.OnHTML("button.product__thumbnail-item.hidden-pocket div img", func(e *colly.HTMLElement) {
		if src := e.Attr("src"); src != "" {
			details.Images = append(details.Images, "https:"+src)
		}
	})

	// Добавляем искусственную задержку перед HTML-запросом (200-500 мс)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(200+rand.Intn(300)) * time.Millisecond):
	}

	// Посещаем страницу
	err = collector.Visit(strings.TrimSuffix(task.Url, ".js"))

	collector.Wait()

	if err != nil {
		return err
	}

	c.chPD <- PageResponse{Product: product, Details: details, Url: task.Url}
	return nil
}

func (c *Crawler) Save(ctx context.Context, tasker crawlers.Tasker) error {
	var task ProductLinksParse
	err := tasker.Model(&task)
	if err != nil {
		return err
	}
	// Создаем директорию для файлов, если она не существует
	dir := "data/dita"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating directory: %v", err)
	}

	// Создаем имя файла с текущей датой
	filename := filepath.Join(dir, fmt.Sprintf("products_%s.csv", time.Now().Format("2006-01-02")))

	// Создаем или открываем файл
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer file.Close()

	// Создаем writer для CSV
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Записываем заголовки
	headers := []string{
		"Название", "Цена", "Цвет", "Линзы", "Артикул (SKU)", "Доступность",
		"Описание", "Ссылка на изображение", "Frame Dimensions", "Frame Width",
		"Frame Height", "Temple", "Bridge", "Frame Material", "Lens Width",
		"Lens Height", "Lens Curve", "Lens Material", "Polarized", "Anti-Reflective",
		"Ссылка на товар",
	}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("error writing headers: %v", err)
	}

	records := make([][]string, 0, batchSize)

	// Функция для записи буфера
	writeRecords := func() error {
		if len(records) == 0 {
			return nil
		}
		if err := writer.WriteAll(records); err != nil {
			return fmt.Errorf("error writing batch of %d records: %v", len(records), err)
		}
		records = records[:0] // Очищаем буфер
		return nil
	}

	c.logger.Printf("Stat PageParse for loop - %d", len(task.Urls))
	// Обрабатываем каждый продукт
	var wg sync.WaitGroup
	for _, link := range task.Urls {
		go func() {
			wg.Add(1)
			defer wg.Done()
			task := BaseParse{Url: link}
			err = c.PageParse(context.Background(), &task)
			if err != nil {
				c.logger.Printf("Error getting product details for %s: %v", link, err)
			}
		}()
	}

	go func() {
		wg.Wait()
		defer close(c.chPD)
	}()

	for ch := range c.chPD {
		product := ch.Product
		details := ch.Details
		// Обрабатываем каждый вариант
		for _, variant := range product.Variants {
			// Получаем изображения для варианта
			variantURL := fmt.Sprintf("%s/en-ru/variants/%d", baseURL, variant.ID)

			variantImages, err := c.ImageParse(ctx, &BaseParse{Url: variantURL})
			if err != nil {
				c.logger.Printf("Error getting variant images for %s: %v", variantURL, err)
				return nil
			}

			// Создаем запись для CSV
			record := []string{
				product.Title,
				fmt.Sprintf("%.2f", float64(product.Price)/100),
				variant.Option1,
				variant.Option2,
				variant.SKU,
				func() string {
					if variant.Available {
						return "Доступен"
					}
					return "Не доступен"
				}(),
				product.Description,
				strings.Join(variantImages, "\n"),
				details.FrameDimensions,
				details.FrameWidth,
				details.FrameHeight,
				details.Temple,
				details.Bridge,
				details.FrameMaterial,
				details.LensWidth,
				details.LensHeight,
				details.LensCurve,
				details.LensMaterial,
				details.Polarized,
				details.AntiReflective,
				strings.TrimSuffix(ch.Url, ".js"),
			}

			records = append(records, record)

			if len(records) >= batchSize {
				if err = writeRecords(); err != nil {
					c.logger.Printf("Error writing batch: %v", err)
					return nil
				}
			}
			if err = writer.Write(record); err != nil {
				c.logger.Printf("Error writing record for %s: %v", ch.Url, err)
				return nil
			}
			return nil
		}
	}

	if err = writeRecords(); err != nil {
		return err
	}

	return err
}

func (c *Crawler) ImageParse(ctx context.Context, tasker crawlers.Tasker) ([]string, error) {
	var task BaseParse
	err := tasker.Model(&task)
	if err != nil {
		return nil, err
	}

	var images []string
	collector := c.collector.Clone()

	collector.OnHTML("button.product__thumbnail-item.hidden-pocket div img", func(e *colly.HTMLElement) {
		if src := e.Attr("src"); src != "" {
			images = append(images, "https:"+src)
		}
	})

	if err := collector.Visit(task.Url); err != nil {
		return nil, fmt.Errorf("error visiting variant page: %v", err)
	}
	collector.Wait()

	return images, nil
}
