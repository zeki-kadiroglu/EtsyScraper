package main

import (
	// import Colly
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/debug"
)

type EtsyProduct struct {
	name     string
	comments string
	rate     string
	price    string
	symbol   string
}

var (
	cache     = make(map[string]string)
	cacheLock sync.Mutex
)

func main() {

	// Create a rate limiter with a limit of 1 request per second per domain
	// limiter := ratelimit.NewBucket(time.Second, 1)

	var etsyProducts []EtsyProduct

	numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	for _, value := range numbers {

		URL := fmt.Sprintf("https://www.etsy.com/search?is_merch_library=true&q=anniversary+gifts&ref=pagination&page=%v", value)

		c := colly.NewCollector(
			colly.Async(true),
			// Attach a debugger to the collector
			colly.Debugger(&debug.LogDebugger{}),
		)
		c.IgnoreRobotsTxt = false

		c.OnError(func(_ *colly.Response, err error) {
			log.Println("Something went wrong: ", err)
		})

		c.OnResponse(func(r *colly.Response) {
			fmt.Println("Page visited: ", r.Request.URL)
		})

		c.OnHTML("div.search-listings-group div[data-search-results-container] ol li div.v2-listing-card__info", func(e *colly.HTMLElement) {

			etsyProduct := EtsyProduct{}

			etsyProduct.name = e.ChildText("h3")
			etsyProduct.comments = e.ChildText("div > div > span")
			etsyProduct.rate = etsyProduct.name[len(etsyProduct.name)-4:]
			// fmt.Printf("rate: %v", etsyProduct.rate)
			etsyProduct.price = e.ChildText("div.n-listing-card__price p.wt-text-title-01 span.currency-value")
			etsyProduct.symbol = e.ChildText("div.n-listing-card__price span.currency-symbol")

			etsyProducts = append(etsyProducts, etsyProduct)
		})

		// Set up caching
		c.OnResponse(func(r *colly.Response) {
			// Cache the content for future use
			cachePage(r.Request.URL.String(), string(r.Body))
		})

		err := c.Visit(URL)
		if err != nil {
			log.Fatal(err)
		}

		c.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36"
		c.AllowURLRevisit = true

		c.Limit(&colly.LimitRule{
			DomainGlob:  "*httpbin.*",
			Parallelism: 2,
			//Delay:      5 * time.Second,
		})

		c.Visit(URL)

		c.Wait()

		// opening the CSV file
		file, err := os.Create("products.csv")
		if err != nil {
			log.Fatalln("Failed to create output CSV file", err)
		}
		defer file.Close()

		// initializing a file writer
		writer := csv.NewWriter(file)

		// writing the CSV headers
		headers := []string{
			"name",
			"comments",
			"rate",
			"price",
			"symbol",
		}

		c.OnScraped(func(r *colly.Response) {
			fmt.Println(r.Request.URL, " scraped!")
		})

		writer.Write(headers)

		// writing each etsyProducts as a CSV row
		for _, etsyProduct := range etsyProducts {
			// converting a etsyProduct to an array of strings
			// fmt.Println(etsyProduct)
			record := []string{
				etsyProduct.name,
				etsyProduct.comments,
				etsyProduct.rate,
				etsyProduct.price,
				etsyProduct.symbol,
			}

			// adding a CSV record to the output file
			writer.Write(record)
		}

		defer writer.Flush()
	}
}

func cachePage(url, content string) {
	cacheLock.Lock()
	defer cacheLock.Unlock()
	cache[url] = content
}

func getFromCache(url string) (string, bool) {
	cacheLock.Lock()
	defer cacheLock.Unlock()
	content, found := cache[url]
	return content, found
}
