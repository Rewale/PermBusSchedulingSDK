package permbusscheduling

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

const (
	searchUrl = "https://www.m.gortransperm.ru/search/?q=%d"
)

type (
	SearchResult struct {
		routeHref string
		RouteName string
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
			}
			isRouteName = t.Data == "h4"

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

// Ищет на сайте расписания определенный номер маршрута и выдает список результатов
func (p *Parser) Search(query int) ([]*SearchResult, error) {
	searchHtml, err := p.getHtmlPage(fmt.Sprintf(searchUrl, query))
	if err != nil {
		return nil, err
	}

	return p.parserResult(searchHtml)
}
