package main

import (
	// import Colly
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/gocolly/colly"
)

type EtsyProduct struct {
	name     string
	comments string
	rate     string
	price    string
	symbol   string
}

type RateLimiter struct {
	mu           sync.Mutex
	tokens       int
	maxTokens    int
	refillRate   time.Duration
	lastRefill   time.Time
	refillAmount int
}

func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:       maxTokens,
		maxTokens:    maxTokens,
		refillRate:   refillRate,
		lastRefill:   time.Now(),
		refillAmount: maxTokens,
	}
}

func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.lastRefill)

	// Refill the tokens if enough time has passed
	if elapsed >= r.refillRate {
		r.tokens = r.maxTokens
		r.lastRefill = now
	} else {
		// Calculate the number of tokens to refill based on the elapsed time
		refillTokens := int(elapsed / (r.refillRate / time.Duration(r.maxTokens)))
		r.tokens = min(r.tokens+refillTokens, r.maxTokens)
	}

	// Consume a token if available
	if r.tokens > 0 {
		r.tokens--
		return true
	}

	return false
}

func main() {

	// Create a new rate limiter with a limit of 5 requests per second
	rateLimiter := NewRateLimiter(5, time.Second)

	var etsyProducts []EtsyProduct

	URL := "https://www.etsy.com/search?is_merch_library=true&q=anniversary+gifts&ref=pagination&page=1"

	c := colly.NewCollector()
	c.IgnoreRobotsTxt = false

	// Set up a callback to be called before making a request
	c.OnRequest(func(r *colly.Request) {
		// Check if it's allowed to make the request
		if !rateLimiter.Allow() {
			fmt.Println("Rate limit exceeded. Waiting...")
			// Sleep or wait before making the next request
			time.Sleep(rateLimiter.refillRate)
		}
	})

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
		etsyProduct.price = e.ChildText("div.n-listing-card__price p.wt-text-title-01 span.currency-value")
		etsyProduct.symbol = e.ChildText("div.n-listing-card__price span.currency-symbol")

		etsyProducts = append(etsyProducts, etsyProduct)
	})
	c.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3"
	c.AllowURLRevisit = true
	c.Visit(URL)

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
		fmt.Println(etsyProduct)
		record := []string{
			etsyProduct.name,
			etsyProduct.comments,
			etsyProduct.price,
			etsyProduct.symbol,
		}

		// adding a CSV record to the output file
		writer.Write(record)
	}

	defer writer.Flush()
}
