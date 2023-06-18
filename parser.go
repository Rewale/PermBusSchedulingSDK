package permbusscheduling

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

const (
	searchUrl = "https://www.m.gortransperm.ru/search/?q=%d"
	baseUrl   = "https://www.m.gortransperm.ru%s"
)

type (
	Scheduling   []time.Time
	SearchResult struct {
		routeHref string
		RouteName string
	}
	Stop struct {
		Name          string
		schedulingUrl string
	}
	Direction struct {
		Name  string
		Stops []Stop
	}

	Parser struct {
		client *http.Client
	}
)

func NewParser(client *http.Client) *Parser {
	return &Parser{
		client: client,
	}
}

func (p *Parser) getHtmlPage(webPage string) (string, error) {
	resp, err := p.client.Get(webPage)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (p *Parser) parserResult(text string) ([]*SearchResult, error) {
	var search *SearchResult
	var isRouteName bool
	result := make([]*SearchResult, 0)
	tkn := html.NewTokenizer(strings.NewReader(text))

	for {
		tt := tkn.Next()
		switch tt {
		case html.ErrorToken:
			return result, nil
		case html.StartTagToken:
			t := tkn.Token()
			if t.Data == "a" {
				for _, attr := range t.Attr {
					if attr.Key == "href" {
						search = new(SearchResult)
						search.routeHref = attr.Val
					}
				}
			} else {
				isRouteName = t.Data == "h4"
			}

		case html.TextToken:
			if isRouteName && search != nil {
				search.RouteName = tkn.Token().Data
				result = append(result, search)
				search = nil
				isRouteName = false
			}
		}
	}
}
func (p *Parser) parseStops(text string) ([]*Direction, error) {
	var wg sync.WaitGroup

	var direction *Direction
	var stop Stop
	var isDirection bool
	var isStops bool
	result := make([]*Direction, 0)
	tkn := html.NewTokenizer(strings.NewReader(text))

	for {
		tt := tkn.Next()
		switch tt {
		case html.ErrorToken:
			wg.Wait()
			return result, nil
		case html.StartTagToken:
			t := tkn.Token()
			if t.Data == "h3" {
				direction = new(Direction)
				isDirection = true
			} else if direction != nil && t.Data == "a" {
				for _, attr := range t.Attr {
					if attr.Key == "href" {
						href := attr.Val
						if !strings.HasPrefix(href, "/time-table") {
							isStops = false
							continue
						}
						isStops = true
						stop = Stop{
							schedulingUrl: href,
						}
					}
					//<a href="/time-table/80/1701">
					//Детский дом культуры им.Кирова
					//</a>
				}
			}
		case html.TextToken:
			if isDirection {
				direction.Name = strings.TrimSpace(tkn.Token().Data)
				result = append(result, direction)
				isDirection = false
			}
			if direction != nil && isStops {
				if direction.Stops == nil {
					direction.Stops = make([]Stop, 0)
				}
				name := strings.TrimSpace(tkn.Token().Data)
				if name != "" {
					stop.Name = name
					direction.Stops = append(direction.Stops, stop)
				}
			}

		}
	}
}

// Расписание прибытия транспорта на остановку
func (p *Parser) StopScheduling(s Stop) (Scheduling, error) {
	schedulingHtml, err := p.getHtmlPage(fmt.Sprintf(baseUrl, s.schedulingUrl))
	if err != nil {
		return nil, err
	}

	return p.parseStopSchedulingHtml(schedulingHtml)
}

func (p *Parser) parseStopSchedulingHtml(text string) ([]time.Time, error) {
	var result []time.Time
	isHour := false
	isMinute := false
	var hour int

	tkn := html.NewTokenizer(strings.NewReader(text))

	for {
		tt := tkn.Next()
		switch tt {
		case html.ErrorToken:
			return result, nil
		case html.StartTagToken:
			t := tkn.Token()
			if t.Data == "div" {
				for _, attr := range t.Attr {
					if attr.Key == "class" && attr.Val == "hour" {
						isHour = true
					}
					if attr.Key == "class" && attr.Val == "minute trip-with-note" && isHour {
						isMinute = true
					}
				}
			}
			if t.Data == "li" {
				isMinute = false
				isHour = false
			}
		case html.TextToken:
			t := tkn.Token()
			if isHour && !isMinute {
				data := strings.TrimSpace(t.Data)
				if data == "" {
					continue
				}
				hourTmp, err := strconv.Atoi(data)
				if err != nil {
					return nil, fmt.Errorf("cant parse hour: %#v", t.Data)
				}
				hour = hourTmp
			}

			if isMinute {
				data := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(t.Data, "*", ""), "\n", ""))
				minute, err := strconv.Atoi(data)
				if err != nil {
					continue
				}
				newTime := time.Date(time.Now().Year(),
					time.Now().Month(), time.Now().Day(),
					hour, minute, 0, 0, time.Now().Location())
				result = append(result, newTime)
			}
		}
	}
}

// Ищет на сайте расписания определенный номер маршрута и выдает список результатов
func (p *Parser) Search(query int) ([]*SearchResult, error) {
	searchHtml, err := p.getHtmlPage(fmt.Sprintf(searchUrl, query))
	if err != nil {
		return nil, err
	}

	return p.parserResult(searchHtml)
}

// Выдает информацию о маршруте: его направления и остановки
func (p *Parser) Stops(search *SearchResult) ([]*Direction, error) {
	stopsHtml, err := p.getHtmlPage(fmt.Sprintf(baseUrl, search.routeHref))
	if err != nil {
		return nil, err
	}

	return p.parseStops(stopsHtml)
}
