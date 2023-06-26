package permbusscheduling

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

type RouteType int

const (
	Bus RouteType = iota + 1
	TrolleyBus
	Tram
	Taxi
)

const (
	searchUrl = "https://www.m.gortransperm.ru/search/?q=%s"
	baseUrl   = "https://www.m.gortransperm.ru%s"
	bus       = "Автобус"
	tram      = "Трамвай"
	taxi      = "Маршрутное такси"
)

type (
	Scheduling []time.Time
	Route      struct {
		RouteUrl  string
		RouteName string
		Type      RouteType
		Number    int
		Literal   string
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
		client          *http.Client
		reNumber        *regexp.Regexp
		reNumberLiteral *regexp.Regexp
	}
)

func NewParser(client *http.Client) *Parser {
	return &Parser{
		client:          client,
		reNumber:        regexp.MustCompile("[0-9]+"),
		reNumberLiteral: regexp.MustCompile("[0-9]+[а-яА-Я]"),
	}
}

func (p *Parser) getHtmlPage(webPage string) (string, error) {
	req, err := http.NewRequest("GET", webPage, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Accept-Charset", "utf-8")

	resp, err := p.client.Do(req)
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

func (p *Parser) parserResult(text string, routeType *RouteType) ([]*Route, error) {
	var search *Route
	var isRouteName bool
	result := make([]*Route, 0)
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
					if attr.Key == "href" && strings.HasPrefix(attr.Val, "/route") {
						search = new(Route)
						search.RouteUrl = attr.Val
						if routeType != nil {
							isRouteName = true
						}
					}
				}
			} else if routeType == nil {
				isRouteName = t.Data == "h4"
			}

		case html.TextToken:
			if isRouteName && search != nil {
				search.RouteName = tkn.Token().Data
				if routeType == nil {
					if strings.HasPrefix(search.RouteName, bus) {
						search.Type = Bus
						search.RouteName = strings.Replace(search.RouteName, bus, "", 1)
					} else if strings.HasPrefix(search.RouteName, tram) {
						search.Type = Tram
						search.RouteName = strings.Replace(search.RouteName, tram, "", 1)
					} else if strings.HasPrefix(search.RouteName, taxi) {
						search.Type = Taxi
						search.RouteName = strings.Replace(search.RouteName, taxi, "", 1)
					} else {
						continue
					}
				} else {
					search.Type = *routeType
				}
				search.RouteName = strings.TrimSpace(search.RouteName)
				search.RouteName = p.removeQuotes(search.RouteName)
				p.extractNumberFromRouteName(search)
				result = append(result, search)
				search = nil
				isRouteName = false
			}
		}
	}
}

func (p *Parser) extractNumberFromRouteName(r *Route) {
	n := p.reNumberLiteral.FindString(r.RouteName)
	if n != "" {
		runeN := []rune(n)
		r.Number, _ = strconv.Atoi(string(runeN[:len(runeN)-1]))
		r.Literal = strings.ToUpper(string(runeN[len(runeN)-1]))
	} else {
		n = p.reNumber.FindString(r.RouteName)
		r.Number, _ = strconv.Atoi(n)
	}
	r.RouteName = strings.ReplaceAll(r.RouteName, fmt.Sprintf("%s, ", n), "")
}

func (p *Parser) removeQuotes(s string) string {
	res := strings.ReplaceAll(s, "«", "")
	res = strings.ReplaceAll(res, "»", "")
	return res
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

// StopScheduling расписание прибытия транспорта на остановку
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

// Search ищет на сайте расписания определенный номер маршрута c литералом и выдает список результатов
func (p *Parser) Search(number string) ([]*Route, error) {
	searchHtml, err := p.getHtmlPage(fmt.Sprintf(searchUrl, url.QueryEscape(number)))
	if err != nil {
		return nil, err
	}

	return p.parserResult(searchHtml, nil)
}

// Stops выдает информацию о маршруте: его направления и остановки
func (p *Parser) Stops(routeUrl string) ([]*Direction, error) {
	stopsHtml, err := p.getHtmlPage(fmt.Sprintf(baseUrl, routeUrl))
	if err != nil {
		return nil, err
	}

	return p.parseStops(stopsHtml)
}

func (p *Parser) AllRoutes(t RouteType) ([]*Route, error) {
	const url = "https://www.m.gortransperm.ru/routes-list/%d/"
	text, err := p.getHtmlPage(fmt.Sprintf(url, int(t)))
	if err != nil {
		return nil, err
	}

	return p.parserResult(text, &t)
}
