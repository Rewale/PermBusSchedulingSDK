package permbusscheduling

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const (
	searchUrl = "https://www.m.gortransperm.ru/search/?q=%d"
	baseUrl   = "https://www.m.gortransperm.ru%s"
)

type (
	SearchResult struct {
		routeHref string
		RouteName string
	}
	Stop struct {
		Name          string
		RouteName     string
		Scheduling    []time.Time
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
	body, err := ioutil.ReadAll(resp.Body)
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
			return result, nil
		case html.StartTagToken:
			t := tkn.Token()
			if t.Data == "h3" {
				direction = new(Direction)
				isDirection = true
			} else if direction != nil && t.Data == "a" {
				href := ""
				for _, attr := range t.Attr {
					if attr.Key == "href" {
						href = attr.Val
					}
					if attr.Key == "class" && attr.Val == "ui-btn ui-btn-icon-right ui-icon-carat-r" {
						isStops = true
						stop = Stop{
							schedulingUrl: href,
						}
						log.Println("Stop found")
					}
				}
			} else if isStops {
				isStops = false
				result = append(result, direction)
				direction = nil
			}

		case html.TextToken:
			if isDirection {
				direction.Name = strings.TrimSpace(tkn.Token().Data)
				result = append(result, direction)
				isDirection = false
			}
			if direction != nil && isStops {
				if direction.Stops == nil {
					direction.Stops = make([]Stop, 10)
				}
				stop.Name = tkn.Token().Data
				// TODO: parse stop scheduling
				direction.Stops = append(direction.Stops, stop)
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

func (p *Parser) Stops(search SearchResult) ([]*Direction, error) {
	stopsHtml, err := p.getHtmlPage(fmt.Sprintf(baseUrl, search.routeHref))
	if err != nil {
		return nil, err
	}

	return p.parseStops(stopsHtml)
}
