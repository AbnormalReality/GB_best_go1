package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type CrawlResult struct {
	Err   error
	Title string
	Url   string
}

type Page interface {
	GetTitle() string
	GetLinks() []string
}

type page struct {
	doc *goquery.Document
}

func NewPage(raw io.Reader) (Page, error) {
	doc, err := goquery.NewDocumentFromReader(raw)
	if err != nil {
		return nil, err
	}
	return &page{doc: doc}, nil
}

func (p *page) GetTitle() string {
	return p.doc.Find("title").First().Text()
}

func (p *page) GetLinks() []string {
	var urls []string
	p.doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		url, ok := s.Attr("href")
		if ok {
			urls = append(urls, url)
		}
	})
	return urls
}

type Requester interface {
	Get(ctx context.Context, url string) (Page, error)
}

type requester struct {
	timeout time.Duration
}

func NewRequester(timeout time.Duration) requester {
	return requester{timeout: timeout}
}

func (r requester) Get(ctx context.Context, url string) (Page, error) {
	select {
	case <-ctx.Done():
		return nil, nil
	default:
		cl := &http.Client{
			Timeout: r.timeout,
		}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		body, err := cl.Do(req)
		if err != nil {
			return nil, err
		}
		defer body.Body.Close()
		page, err := NewPage(body.Body)
		if err != nil {
			return nil, err
		}
		return page, nil
	}
}

//Crawler - интерфейс (контракт) краулера
type Crawler interface {
	Scan(ctx context.Context, url string, incNum int64)
	ChanResult() <-chan CrawlResult
	AddDepth()
}

type crawler struct {
	r        Requester
	maxDepth int64
	res      chan CrawlResult
	visited  map[string]struct{}
	mu       sync.RWMutex
}

func NewCrawler(r Requester, depth int64) *crawler {
	return &crawler{
		r:        r,
		maxDepth: depth,
		res:      make(chan CrawlResult),
		visited:  make(map[string]struct{}),
		mu:       sync.RWMutex{},
	}
}

func (c *crawler) Scan(ctx context.Context, url string, incNum int64) {
	time.Sleep(10 * time.Second) // // просто для того, чтобы успевать следить за выводом программы
	if c.maxDepth <= incNum {
		return
	}
	c.mu.RLock()
	_, ok := c.visited[url] //Проверяем, что мы ещё не смотрели эту страницу
	c.mu.RUnlock()
	if ok {
		return
	}
	select {
	case <-ctx.Done(): //Если контекст завершен - прекращаем выполнение
		return
	default:
		page, err := c.r.Get(ctx, url) //Запрашиваем страницу через Requester
		if err != nil {
			c.res <- CrawlResult{Err: err} //Записываем ошибку в канал
			return
		}
		c.mu.Lock()
		c.visited[url] = struct{}{} //Помечаем страницу просмотренной
		c.mu.Unlock()
		c.res <- CrawlResult{ //Отправляем результаты в канал
			Title: page.GetTitle(),
			Url:   url,
		}
		atomic.AddInt64(&incNum, 1)

		for _, link := range page.GetLinks() {
			go c.Scan(ctx, link, incNum) //На все полученные ссылки запускаем новую рутину сборки
		}
	}
}

func (c *crawler) ChanResult() <-chan CrawlResult {
	return c.res
}
func (c *crawler) AddDepth() {
	c.maxDepth += 2
}

//Config - структура для конфигурации
type Config struct {
	MaxDepth   int64
	MaxResults int
	MaxErrors  int
	Url        string
	Timeout    int //in seconds
}

func main() {

	t := flag.Duration("t", 60*time.Second, "total timeout process") // общий таймаут простой
	flag.Parse()

	cfg := Config{
		MaxDepth:   3,
		MaxResults: 10,
		MaxErrors:  5,
		Url:        "https://github.com",
		Timeout:    10,
	}
	var cr Crawler
	var r Requester

	r = NewRequester(time.Duration(cfg.Timeout) * time.Second) // общий таймаут простой
	cr = NewCrawler(r, cfg.MaxDepth)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(*t))
	go cr.Scan(ctx, cfg.Url, 1)            //Запускаем краулер в отдельной рутине
	go processResult(ctx, cancel, cr, cfg) //Обрабатываем результаты в отдельной рутине

	sigCh := make(chan os.Signal)                         //Создаем канал для приема сигналов
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGUSR1) //Подписываемся на сигналы SIGINT и SIGUSR1
	// линтер typecheck вывел ошибку SIGUSR1 not declared by pack age syscall (на винде не исправить,
	//но непонятно почему все линтеры отключены в yml, а typecheck все равно работает)
	for {
		select {
		case <-ctx.Done(): //Если всё завершили - выходим
			return
		case sig := <-sigCh:
			if sig == syscall.SIGINT {
				cancel() //Если пришёл сигнал SigInt - завершаем контекст
			}
			if sig == syscall.SIGUSR1 { //линтер typecheck вывел ошибку SIGUSR1 not declared by pack age syscall (на винде не исправить,
				//но непонятно почему все линтеры отключены в yml, а typecheck все равно работает)
				cr.AddDepth() //Если пришёл сигнал SigUsr1 - увеличиваем глубину поиска на 2
			}

		}
	}
}

func processResult(ctx context.Context, cancel func(), cr Crawler, cfg Config) {
	var maxResult, maxErrors = cfg.MaxResults, cfg.MaxErrors
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-cr.ChanResult():
			if msg.Err != nil {
				maxErrors--
				log.Printf("crawler result return err: %s\n", msg.Err.Error())
				if maxErrors <= 0 {
					cancel()
					return
				}
			} else {
				maxResult--
				log.Printf("crawler result: [url: %s] Title: %s\n", msg.Url, msg.Title)
				if maxResult <= 0 {
					cancel()
					return
				}
			}
		}
	}
}
