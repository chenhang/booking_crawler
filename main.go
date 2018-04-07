package main

import (
	"fmt"
	"time"
	"github.com/gocolly/colly"
	"strings"
	"strconv"
	"encoding/json"
	"io/ioutil"
)

type Hotel struct {
	Id             string   `json:"id"`
	URL            string   `json:"url"`
	Name           string   `json:"name"`
	ReviewStartURL string   `json:"review_start_url"`
	ReviewCount    int64    `json:"review_count"`
	Reviews        []string `json:"reviews"`
}

func main() {
	hotels := make(map[string]*Hotel)

	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/65.0.3325.181 Safari/537.36"),
	)

	c.Limit(&colly.LimitRule{
		//DomainGlob:  ".*booking.*",
		Parallelism: 1,
		Delay:       2 * time.Second,
	})

	detailCollector := c.Clone()
	reviewCollector := c.Clone()

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting: ", r.URL.String())
	})

	c.OnResponse(func(r *colly.Response) {
		fmt.Println("Visited", r.Request.URL)
	})

	detailCollector.OnHTML(`body`, func(ee *colly.HTMLElement) {
		hotelId := ee.Request.Ctx.Get("hotelId")
		hotels[hotelId].ReviewStartURL = ee.Request.AbsoluteURL(ee.ChildAttr(`button[data-reviews-url]`, "data-reviews-url"))
		hotels[hotelId].ReviewCount, _ = strconv.ParseInt(ee.ChildAttr(`#review_filters > option:first-child`, "data-review-filter-number"), 0, 0)
		reviewCollector.Request("GET", hotels[hotelId].ReviewStartURL, nil, ee.Request.Ctx, nil)
	})

	reviewCollector.OnHTML(`body`, func(ee *colly.HTMLElement) {
		hotelId := ee.Request.Ctx.Get("hotelId")
		fmt.Println(ee.Request.URL)

		ee.ForEach(`#comments > li`, func(i int, ce *colly.HTMLElement) {
			hotels[hotelId].Reviews = append(hotels[hotelId].Reviews,
				strings.Join([]string{ce.ChildText(`.comments`), ce.ChildText(`.m-review-summary`)}, "\n"))
		})

		fmt.Println("current count: ", len(hotels[hotelId].Reviews))
		fmt.Println("total count:", hotels[hotelId].ReviewCount)
		ee.ForEach(`.pagenext`, func(i int, pe *colly.HTMLElement) {
			reviewCollector.Request("GET", pe.Request.AbsoluteURL(pe.Attr("data-js-umhcrdap-next")), nil, ee.Request.Ctx, nil)
		})
	})

	c.OnHTML(`body`, func(e *colly.HTMLElement) {
		e.ForEach(`.sr-card`, func(i int, ee *colly.HTMLElement) {
			hotel := Hotel{
				Id:      ee.Attr(`id`),
				URL:     strings.Split(e.Request.AbsoluteURL(ee.ChildAttr(`.sr-card__row`, "href")), "?")[0],
				Name:    ee.ChildText(`.sr-card__name`),
				Reviews: make([]string, 2),
			}

			hotels[hotel.Id] = &hotel
			ee.Request.Ctx.Put("hotelId", hotel.Id)

			detailCollector.Request("GET", hotel.URL, nil, ee.Request.Ctx, nil)
		})

		e.ForEach(`#sr_pagination`, func(i int, ee *colly.HTMLElement) {
			links := ee.ChildAttrs(`.sr-pager__link`, "href")
			if len(links) != 2 {
				c.Request("GET", e.Request.AbsoluteURL(links[len(links)-1]), nil, e.Request.Ctx, nil)
			}
		})

		result, err := json.Marshal(hotels)
		if err != nil {
			fmt.Println(result, err)
		}
		err = ioutil.WriteFile("output.json", result, 0644)
		if err != nil {
			fmt.Println(err)
		}
	})
	// pa: 2610
	// ca: 2279
	c.Visit("https://www.booking.com/searchresults.html?dest_id=2279;dest_type=region&prefer_site_type=mdot")
}
