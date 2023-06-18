package permbusscheduling

import (
	"fmt"
	"io/ioutil"
	"log"
	"testing"
	"time"
)

func GetTestHtml(path string) string {

	bs, err := ioutil.ReadFile(path)

	if err != nil {
		log.Fatalln(err)
	}

	return string(bs)
}
func TestParseSearchResult(t *testing.T) {
	parser := NewParser(nil)
	testTable := []struct {
		name        string
		html        string
		wantResults []SearchResult
		wantError   bool
	}{
		{
			name:        "Invalid html",
			html:        "",
			wantError:   false,
			wantResults: nil,
		},
		{
			name:        "Single search result",
			html:        GetTestHtml("testData/SingleSearchResult.html"),
			wantError:   false,
			wantResults: []SearchResult{{routeHref: "/route/80/", RouteName: "Автобус «80, ДДК им. Кирова - ул. Милиционера Власова»"}},
		},
		{
			name:      "Three search results",
			html:      GetTestHtml("testData/ThreeSearchResult.html"),
			wantError: false,
			wantResults: []SearchResult{
				{routeHref: "/route/80/", RouteName: "Автобус «80, ДДК им. Кирова - ул. Милиционера Власова»"},
				{routeHref: "/route/79/", RouteName: "Автобус «79, Test test»"},
				{routeHref: "/route/7988/", RouteName: "Трамвай «12, Тест тест теееест»"},
			},
		},
	}
	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			res, err := parser.parserResult(testCase.html)

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
				if res[i].routeHref != testCase.wantResults[i].routeHref || res[i].RouteName != testCase.wantResults[i].RouteName {
					t.Errorf("Diff results: got %s, want %s", res[i], testCase.wantResults[i])
				}
			}

		})

	}
}

func TestStops(t *testing.T) {
	parser := NewParser(nil)
	res, _ := parser.parseStops(GetTestHtml("testData/Route80.html"))

	fmt.Printf("%#v", res)
	testTable := []struct {
		name        string
		html        string
		wantResults []Direction
		stopsCount  []int
	}{
		{
			name:        "Invalid html",
			html:        "",
			wantResults: nil,
			stopsCount:  nil,
		},
		{
			name: "Correct html",
			html: GetTestHtml("testData/Route80.html"),
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
			res, err := parser.parseStops(ts.html)
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
			html: GetTestHtml("testData/Stop80_1701.html"),
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
