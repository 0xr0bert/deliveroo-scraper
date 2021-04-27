package main

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Open database
	db, err := sql.Open("sqlite3", "./results.db")
	if err != nil {
		panic(err)
	}

	// Create tables
	_, err = db.Exec(
		"CREATE TABLE IF NOT EXISTS customers_to_restaurants(customer_id VARCHAR(8), restaurant_id TEXT, PRIMARY KEY(customer_id, restaurant_id));",
	)

	if err != nil {
		panic(err)
	}

	_, err = db.Exec(
		"CREATE TABLE IF NOT EXISTS restaurants(url TEXT PRIMARY KEY, name TEXT, avg_rating REAL, address TEXT, description TEXT);",
	)

	if err != nil {
		panic(err)
	}

	// Create processors

	var wg sync.WaitGroup

	customerToRestaurantChan := make(chan customerToRestaurant, 1000)

	wg.Add(1)

	go func() {
		defer wg.Done()
		writeCustomersToRestaurants(db, customerToRestaurantChan)
	}()

	restaurantNameChan := make(chan restaurantName, 1000)

	wg.Add(1)

	go func() {
		defer wg.Done()
		writeRestaurantName(db, restaurantNameChan)
	}()

	// Create colly collector
	c := colly.NewCollector(
		colly.Async(true),
		colly.MaxDepth(2),
	)
	c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 2, RandomDelay: 1 * time.Second})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		if e.Request.Ctx.Get("type") == "restaurant_list" {
			url := e.Attr("href")
			if strings.Contains(url, "/menu/") {
				urlToVisit := fmt.Sprint("https://deliveroo.co.uk", strings.Split(url, "?")[0])

				ctx := colly.NewContext()
				ctx.Put("type", "restaurant_menu")

				customerToRestaurantChan <- customerToRestaurant{customerID: e.Request.Ctx.Get("postcode"), restaurantID: urlToVisit}

				c.Request(
					"GET",
					urlToVisit,
					nil,
					ctx,
					nil,
				)
			}
		}
	})

	c.OnHTML("div.restaurant__details h1", func(e *colly.HTMLElement) {
		if e.Request.Ctx.Get("type") == "restaurant_menu" {
			restaurantNameChan <- restaurantName{url: e.Request.URL.String(), name: e.Text}
		}
	})

	// For each postcode
	rows, err := db.Query("SELECT * FROM customers")
	if err != nil {
		panic(err)
	}
	var postcode string

	for rows.Next() {
		err = rows.Scan(&postcode)
		if err != nil {
			panic(err)
		}
		ctx := colly.NewContext()
		ctx.Put("postcode", postcode)
		ctx.Put("type", "restaurant_list")

		c.Request(
			"GET",
			fmt.Sprintf(
				"https://deliveroo.co.uk/restaurants/london/camden?postcode=%s&collection=all-restaurants",
				strings.ReplaceAll(postcode, " ", "+"),
			),
			nil,
			ctx,
			nil,
		)
	}
	rows.Close()
	c.Wait()
	close(customerToRestaurantChan)
	close(restaurantNameChan)

	wg.Wait()
	db.Close()
}

type customerToRestaurant struct {
	customerID   string
	restaurantID string
}

func writeCustomersToRestaurants(db *sql.DB, c <-chan customerToRestaurant) {
	stmt, err := db.Prepare("INSERT OR IGNORE INTO customers_to_restaurants(customer_id, restaurant_id) VALUES (?, ?)")

	if err != nil {
		panic(err)
	}

	for custRest := range c {
		_, err := stmt.Exec(custRest.customerID, custRest.restaurantID)

		if err != nil {
			panic(err)
		}
	}

	stmt.Close()
}

type restaurantName struct {
	url  string
	name string
}

func writeRestaurantName(db *sql.DB, c <-chan restaurantName) {
	stmt1, err := db.Prepare("INSERT OR IGNORE INTO restaurants(url, name) VALUES (?,?)")

	if err != nil {
		panic(err)
	}

	stmt2, err := db.Prepare("UPDATE restaurants SET name = ? WHERE url = ?")

	if err != nil {
		panic(err)
	}

	for i := range c {
		_, err := stmt1.Exec(i.url, i.name)

		if err != nil {
			panic(err)
		}

		_, err = stmt2.Exec(i.name, i.url)

		if err != nil {
			panic(err)
		}
	}
	stmt1.Close()
	stmt2.Close()
}
