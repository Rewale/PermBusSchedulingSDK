package permbusscheduling

import (
	"fmt"
	"io/ioutil"
	"log"
	"testing"
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
		stopsCount  int
	}{
		{
			name:        "Invalid html",
			html:        "",
			wantResults: nil,
			stopsCount:  20,
		},
		{
			name: "Invalid html",
			html: GetTestHtml("testData/Route80.html"),
			wantResults: []Direction{
				{
					Name: "Детский дом культуры им.Кирова – ул. Милиционера Власова",
				},
				{
					Name: "ул. Милиционера Власова – Детский дом культуры им.Кирова",
				},
			},
			stopsCount: 0,
		},
	}

	for _, ts := range testTable {
		t.Run(ts.name, func(t *testing.T) {
			res, err := parser.parseStops(ts.html)
			if err != nil {
				t.Fatal(err)
			}

			if len(res) != len(ts.wantResults) {
				t.Fatalf("Diff length: want %d, got %d", len(ts.wantResults), len(res))
			}

			for i := range ts.wantResults {
				if ts.wantResults[i].Name != res[i].Name {
					t.Fatalf("Diff names: want %s, got %s", ts.wantResults[i].Name, res[i].Name)
				}
			}

		})

	}

}
