package permbusscheduling

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func getTestHtml(path string) string {

	bs, err := os.ReadFile(path)

	if err != nil {
		log.Fatalln(err)
	}

	return string(bs)
}

type roundTripFunc func(r *http.Request) (*http.Response, error)

func (s roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return s(r)
}

func newTestParser(statusCode int, body string) *Parser {
	testClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {

			return &http.Response{
				StatusCode: statusCode,
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		}),
	}

	return NewParser(testClient)
}

func TestParseSearchResult(t *testing.T) {
	testTable := []struct {
		name        string
		html        string
		wantResults []Route
		wantError   bool
		Search      string
	}{
		{
			Search:    "80",
			name:      "Single search result",
			html:      getTestHtml("testData/SingleSearchResult.html"),
			wantError: false,
			wantResults: []Route{{RouteUrl: "/route/80/", Number: 80, Literal: "",
				RouteName: "ДДК им. Кирова - ул. Милиционера Власова", Type: Bus}},
		},
		{
			Search:    "7Т",
			name:      "Number with literal",
			html:      getTestHtml("testData/7T.html"),
			wantError: false,
			wantResults: []Route{{RouteUrl: "/route/207/", Number: 7,
				Literal:   "Т",
				RouteName: "Н.Крым - Центральный рынок", Type: Taxi}},
		},
		{
			Search:    "7т",
			name:      "Number with low literal",
			html:      getTestHtml("testData/7T.html"),
			wantError: false,
			wantResults: []Route{{RouteUrl: "/route/207/", Number: 7,
				Literal:   "Т",
				RouteName: "Н.Крым - Центральный рынок", Type: Taxi}},
		},
		{
			Search:    "12",
			name:      "Multiple result",
			html:      getTestHtml("testData/12.html"),
			wantError: false,
			wantResults: []Route{
				{
					RouteUrl: "/route/812/", Number: 12,
					Literal:   "",
					RouteName: "Школа № 107 - Разгуляй", Type: Tram,
				},
				{
					RouteUrl: "/route/12/", Number: 12,
					Literal:   "",
					RouteName: "Дворец культуры им. Гагарина - ОАО \"ПЗСП\"", Type: Bus,
				},
			},
		},
	}
	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			var res []*Route
			var err error
			if testing.Short() {
				res, err = newTestParser(200, testCase.html).Search(testCase.Search)
			} else {
				res, err = NewParser(&http.Client{}).Search(testCase.Search)
			}

			if err == nil && testCase.wantError {
				t.Error("need error, but err == nil")
			}

			if err != nil {
				t.Error(err)
			}

			if len(testCase.wantResults) != len(res) {
				t.Fatalf("Diff lengths:\ngot %d\nwant %d", len(res), len(testCase.wantResults))
			}

			for i := range testCase.wantResults {
				if res[i].RouteUrl != testCase.wantResults[i].RouteUrl ||
					res[i].RouteName != testCase.wantResults[i].RouteName ||
					res[i].Type != testCase.wantResults[i].Type ||
					res[i].Number != testCase.wantResults[i].Number {
					t.Errorf("Diff routes: got %v, want %v", res[i], testCase.wantResults[i])
				}
			}

		})

	}
}

func TestAllRoutes(t *testing.T) {
	parser := NewParser(nil)
	wantType := Bus
	wantCount := 71
	text := getTestHtml("testData/AllBusRoutes.html")

	res, err := parser.parserResult(text, &wantType)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != wantCount {
		t.Fatalf("Wrong length. got %d want %d", len(res), wantCount)
		return
	}
	for _, r := range res {
		if r.Type != wantType {
			t.Fatalf("Wrong type. want %d got %d", wantType, r.Type)
		}
		if r.RouteName == "" {
			t.Fatal("Empty route name")
		}
		if r.RouteUrl == "" {
			t.Fatal("Empty url")
		}
	}
}

func TestStops(t *testing.T) {
	var parser *Parser
	if testing.Short() {
		parser = NewParser(nil)
	} else {
		parser = NewParser(&http.Client{})
	}
	res, _ := parser.parseStops(getTestHtml("testData/Route80.html"))

	fmt.Printf("%#v", res)
	testTable := []struct {
		name        string
		html        string
		wantResults []Direction
		stopsCount  []int
		route       *Route
	}{
		{
			route: &Route{RouteUrl: "/route/80/", Number: 80,
				RouteName: "80, ДДК им. Кирова - ул. Милиционера Власова", Type: Bus},
			name: "Correct html",
			html: getTestHtml("testData/Route80.html"),
			wantResults: []Direction{
				{
					Name: "Детский дом культуры им.Кирова – ул. Милиционера Власова",
				},
				{
					Name: "ул. Милиционера Власова – Детский дом культуры им.Кирова",
				},
			},
			stopsCount: []int{26, 20},
		},
	}

	for _, ts := range testTable {
		t.Run(ts.name, func(t *testing.T) {
			var res []*Direction
			var err error
			if testing.Short() {
				res, err = parser.parseStops(ts.html)
			} else {
				res, err = parser.Stops(ts.route.RouteUrl)
			}

			if err != nil {
				t.Fatal(err)
			}

			if len(res) != len(ts.wantResults) {
				fmt.Println("Got:")
				for _, r := range res {
					fmt.Printf("Name: %s Stops: %#v\n", r.Name, nil)
				}
				fmt.Printf("want %#v", ts.wantResults)
				t.Fatalf("Diff length: want %d, got %d", len(ts.wantResults), len(res))
			}

			for i := range ts.wantResults {
				printStops(res[i].Stops)
				if ts.wantResults[i].Name != res[i].Name {
					t.Fatalf("Diff names: want %s, got %s", ts.wantResults[i].Name, res[i].Name)
				}
				if ts.stopsCount[i] != len(res[i].Stops) {
					t.Fatalf("[%d] Diff stops length: want %d, got %d", i, ts.stopsCount[i], len(res[i].Stops))
				}
			}

		})

	}

}

func TestParseScheduling(t *testing.T) {
	parser := NewParser(nil)
	testTable := []struct {
		name           string
		wantScheduling []time.Time
		html           string
	}{
		{
			html: getTestHtml("testData/Stop80_1701.html"),
			wantScheduling: []time.Time{
				newTime(5, 50),
				newTime(6, 16),
				newTime(6, 38),
				newTime(7, 01),
				newTime(7, 23),
				newTime(7, 45),

				newTime(8, 07),
				newTime(8, 29),
				newTime(8, 51),

				newTime(9, 14),
				newTime(9, 37),
			},
			name: "Correct stops html",
		},
		{
			html:           "",
			wantScheduling: []time.Time{},
			name:           "Incorrect stops html",
		},
	}

	for _, ts := range testTable {
		t.Run(ts.name, func(t *testing.T) {
			res, err := parser.parseStopSchedulingHtml(ts.html)
			if err != nil {
				t.Fatal(err)
			}

			if len(res) != len(ts.wantScheduling) {
				t.Fatalf("Wrong length. Want %d got %d", len(res), len(ts.wantScheduling))
			}

			for i := range res {
				if res[i].Hour() != ts.wantScheduling[i].Hour() {
					t.Fatalf("Wrong hour. got %d, want %d", res[i].Hour(), ts.wantScheduling[i].Hour())
				}
				if res[i].Minute() != ts.wantScheduling[i].Minute() {
					t.Fatalf("Wrong minutes. got %d, want %d", res[i].Minute(), ts.wantScheduling[i].Minute())
				}
			}

		})
	}
}

func printStops(stops []Stop) {
	if testing.Short() {
		return
	}
	fmt.Println("Stops:")
	for i, s := range stops {
		fmt.Printf("\t%d - %s - %s\n", i+1, s.Name, s.schedulingUrl)
	}
	fmt.Println("end")
}

func newTime(hour, minute int) time.Time {
	return time.Date(time.Now().Year(),
		time.Now().Month(), time.Now().Day(),
		hour, minute, 0, 0, time.Now().Location())
}
