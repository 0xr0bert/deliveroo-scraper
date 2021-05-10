package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/proxy"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})

	log.SetLevel(log.DebugLevel)
}

func main() {
	// Open database
	db, err := sql.Open("postgres", "user=robert dbname=deliveroo-scraping sslmode=disable")

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

	// Create colly collector
	c := colly.NewCollector(
		colly.Async(true),
		colly.MaxDepth(2),
	)
	c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 2, RandomDelay: 500 * time.Millisecond})

	rp, err := proxy.RoundRobinProxySwitcher(
		"https://robertgreener%40protonmail.ch:Repose5-Defrost-Engraving-Gander@uk2150.nordvpn.com:89",
		"https://robertgreener%40protonmail.ch:Repose5-Defrost-Engraving-Gander@uk1784.nordvpn.com:89",
		"https://robertgreener%40protonmail.ch:Repose5-Defrost-Engraving-Gander@uk1894.nordvpn.com:89",
	)

	if err != nil {
		panic(err)
	}
	c.SetProxyFunc(rp)

	c.OnRequest(func(r *colly.Request) {
		log.WithField("url", r.URL.String()).Debug("Visiting")
	})

	c.OnError(func(r *colly.Response, err error) {
		log.WithFields(log.Fields{
			"url":      r.Request.URL.String(),
			"response": r,
			"err":      err,
		}).Error("Failed")
	})

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		if e.Request.Ctx.Get("type") == "restaurant_list" {
			url := e.Attr("href")
			if strings.Contains(url, "/menu/") {
				urlToVisit := fmt.Sprint("https://deliveroo.co.uk", strings.Split(url, "?")[0])

				ctx := colly.NewContext()
				ctx.Put("type", "restaurant_menu")
				ctx.Put("url", urlToVisit)

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

	c.OnHTML("script[data-component-name=\"MenuIndexApp\"]", func(e *colly.HTMLElement) {
		wg.Add(1)
		go func(db *sql.DB, e *colly.HTMLElement) {
			defer wg.Done()
			processMenuBody(db, e)
		}(db, e)
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

	wg.Wait()
	db.Close()
}

type customerToRestaurant struct {
	customerID   string
	restaurantID string
}

func writeCustomersToRestaurants(db *sql.DB, c <-chan customerToRestaurant) {
	stmt, err := db.Prepare("INSERT INTO customers_to_restaurants(customer_id,restaurant_id) VALUES ($1,$2) ON CONFLICT DO NOTHING")

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

func processMenuBody(db *sql.DB, e *colly.HTMLElement) {
	var details restaurantDetails
	err := json.Unmarshal([]byte(e.Text), &details)

	if err != nil {
		panic(err)
	}

	stmt, err := db.Prepare("INSERT INTO restaurants(url, name, avg_rating, address, description) VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING")

	if err != nil {
		panic(err)
	}

	_, err = stmt.Exec(
		e.Request.Ctx.Get("url"),
		details.Restaurant.NameWithBranch,
		details.Rating.Value,
		fmt.Sprintf("%s, %s", details.Restaurant.StreetAddress, details.Restaurant.PostCode),
		details.Restaurant.Description,
	)

	if err != nil {
		panic(err)
	}

	stmt.Close()

	// Tags

	stmt1, err := db.Prepare("INSERT INTO tag_types(name) VALUES ($1) ON CONFLICT DO NOTHING")

	if err != nil {
		panic(err)
	}

	stmt2, err := db.Prepare("INSERT INTO tags(name, tag_type_id) VALUES ($1, $2) ON CONFLICT DO NOTHING")

	if err != nil {
		panic(err)
	}

	stmt3, err := db.Prepare("INSERT INTO tags_restaurants(tag_id, restaurant_id) VALUES ($1, $2) ON CONFLICT DO NOTHING")

	if err != nil {
		panic(err)
	}

	for _, menuTag := range details.Restaurant.Menu.MenuTags {
		_, err = stmt1.Exec(menuTag.Type)

		if err != nil {
			panic(err)
		}

		_, err := stmt2.Exec(menuTag.Name, menuTag.Type)

		if err != nil {
			panic(err)
		}

		_, err = stmt3.Exec(menuTag.Name, e.Request.Ctx.Get("url"))

		if err != nil {
			panic(err)
		}
	}

	stmt1.Close()
	stmt2.Close()
	stmt3.Close()

	// Menu categories

	stmt, err = db.Prepare("INSERT INTO menu_categories(id, name, description, restaurant_id) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING")

	if err != nil {
		panic(err)
	}

	for _, category := range details.Menu.Categories {
		_, err = stmt.Exec(category.ID, category.Name, category.Description, e.Request.Ctx.Get("url"))

		if err != nil {
			panic(err)
		}
	}

	stmt.Close()

	// Items

	stmt, err = db.Prepare("INSERT INTO items(id, name, price, is_popular, menu_category_id) VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING")

	if err != nil {
		panic(err)
	}

	for _, item := range details.Menu.Items {
		_, err = stmt.Exec(item.ID, item.Name, item.RawPrice, item.Popular, item.CategoryID)

		if err != nil {
			panic(err)
		}
	}

	stmt.Close()
}

type restaurantDetails struct {
	CurrencyCode string `json:"currency_code"`
	Urls         struct {
		Homepage                 string `json:"homepage"`
		Back                     string `json:"back"`
		Editions                 string `json:"editions"`
		Checkout                 string `json:"checkout"`
		RestaurantSearch         string `json:"restaurant_search"`
		AddToBasket              string `json:"add_to_basket"`
		AllergyNote              string `json:"allergy_note"`
		Basket                   string `json:"basket"`
		Cookies                  string `json:"cookies"`
		Login                    string `json:"login"`
		LoginPage                string `json:"login_page"`
		Logout                   string `json:"logout"`
		LogoutPage               string `json:"logout_page"`
		Registration             string `json:"registration"`
		PasswordReset            string `json:"password_reset"`
		Root                     string `json:"root"`
		ChangeLocation           string `json:"change_location"`
		Privacy                  string `json:"privacy"`
		Orders                   string `json:"orders"`
		Account                  string `json:"account"`
		Current                  string `json:"current"`
		SharedBaskets            string `json:"shared_baskets"`
		SharedBasketsBasket      string `json:"shared_baskets_basket"`
		SharedBasketsLeave       string `json:"shared_baskets_leave"`
		SharedBasketsUpdateName  string `json:"shared_baskets_update_name"`
		SharedBasketsMerge       string `json:"shared_baskets_merge"`
		SharedBasketsOrderStatus string `json:"shared_baskets_order_status"`
		SharedBasketsCancel      string `json:"shared_baskets_cancel"`
	} `json:"urls"`
	Basket struct {
		RestaurantID          int         `json:"restaurant_id"`
		FulfillmentMethod     string      `json:"fulfillment_method"`
		UserAddressID         interface{} `json:"user_address_id"`
		ScheduledDeliveryDay  string      `json:"scheduled_delivery_day"`
		ScheduledDeliveryTime string      `json:"scheduled_delivery_time"`
		AllergyNote           interface{} `json:"allergy_note"`
		TargetDeliveryTime    string      `json:"target_delivery_time"`
		ChangeLog             struct {
			LastAlteredItemID interface{} `json:"last_altered_item_id"`
			PreviousAction    interface{} `json:"previous_action"`
		} `json:"change_log"`
		CurrencyCode   string `json:"currency_code"`
		CurrencySymbol string `json:"currency_symbol"`
		CountryCode    string `json:"country_code"`
		BasketItems    []struct {
			Name  string        `json:"name"`
			Items []interface{} `json:"items"`
		} `json:"basket_items"`
		Items                  []interface{} `json:"items"`
		RecommendedMenuItemIds []interface{} `json:"recommended_menu_item_ids"`
		UnavailableItems       []interface{} `json:"unavailable_items"`
		IsAsap                 bool          `json:"is_asap"`
		WhiteLabelBrand        string        `json:"white_label_brand"`
		Fee                    float64       `json:"fee"`
		FeeFormatted           string        `json:"fee_formatted"`
		FeeBreakdown           []struct {
			Title                          string      `json:"title"`
			Description                    string      `json:"description"`
			FormattedAmount                string      `json:"formatted_amount"`
			Amount                         int         `json:"amount"`
			TooltipViewedTrackingEventName string      `json:"tooltip_viewed_tracking_event_name"`
			Type                           interface{} `json:"type"`
		} `json:"fee_breakdown"`
		DriverTip                   float64     `json:"driver_tip"`
		DriverTipFormatted          string      `json:"driver_tip_formatted"`
		Subtotal                    float64     `json:"subtotal"`
		SubtotalFormatted           string      `json:"subtotal_formatted"`
		Surcharge                   float64     `json:"surcharge"`
		SurchargeFormatted          string      `json:"surcharge_formatted"`
		SurchargeDifference         string      `json:"surcharge_difference"`
		SurchargeThreshold          float64     `json:"surcharge_threshold"`
		SurchargeThresholdFormatted string      `json:"surcharge_threshold_formatted"`
		Total                       float64     `json:"total"`
		TotalFormatted              string      `json:"total_formatted"`
		ContainsAlcohol             bool        `json:"contains_alcohol"`
		UserConfirmedOver18         interface{} `json:"user_confirmed_over_18"`
		TargetTimeMinutesFromNow    int         `json:"target_time_minutes_from_now"`
		TipPromptThreshold          int         `json:"tip_prompt_threshold"`
		PromotionIncentive          interface{} `json:"promotion_incentive"`
		IsProgressBarEnabled        bool        `json:"is_progress_bar_enabled"`
	} `json:"basket"`
	TipPromptThreshold int `json:"tip_prompt_threshold"`
	FulfillmentTimes   []struct {
		FulfillmentMethodLabel string `json:"fulfillment_method_label"`
		FulfillmentMethod      string `json:"fulfillment_method"`
		Asap                   struct {
			OptionLabel          string `json:"option_label"`
			OptionDisplayLabel   string `json:"option_display_label"`
			SelectedLabel        string `json:"selected_label"`
			SelectedDisplayLabel string `json:"selected_display_label"`
			SelectedTime         struct {
				Day  string `json:"day"`
				Time string `json:"time"`
			} `json:"selected_time"`
			Timestamp string `json:"timestamp"`
		} `json:"asap"`
		Days []struct {
			Day      string `json:"day"`
			DayLabel string `json:"day_label"`
			Times    []struct {
				OptionLabel          string `json:"option_label"`
				OptionDisplayLabel   string `json:"option_display_label"`
				SelectedLabel        string `json:"selected_label"`
				SelectedDisplayLabel string `json:"selected_display_label"`
				SelectedTime         struct {
					Day  string `json:"day"`
					Time string `json:"time"`
				} `json:"selected_time"`
				Timestamp string `json:"timestamp"`
			} `json:"times"`
		} `json:"days"`
		RawDeliveryEstimates struct {
			FulfillmentType string  `json:"fulfillment_type"`
			Minutes         float64 `json:"minutes"`
			MinutesLower    int     `json:"minutes_lower"`
			MinutesUpper    int     `json:"minutes_upper"`
			Range           string  `json:"range"`
		} `json:"raw_delivery_estimates"`
	} `json:"fulfillment_times"`
	DeliveryLocation struct {
		Address                  interface{} `json:"address"`
		ValidateAddressURL       string      `json:"validate_address_url"`
		AddressSearchPlaceholder string      `json:"address_search_placeholder"`
		ExamplePostcode          string      `json:"example_postcode"`
	} `json:"delivery_location"`
	Errors struct {
	} `json:"errors"`
	Banners                    []interface{} `json:"banners"`
	DistancePresentational     string        `json:"distance_presentational"`
	ShowDistancePresentational bool          `json:"show_distance_presentational"`
	CountryHasPostcodes        bool          `json:"country_has_postcodes"`
	DeloverooPrideLogo         bool          `json:"deloveroo_pride_logo"`
	PromoteEditions            bool          `json:"promote_editions"`
	Rating                     struct {
		Value            float64 `json:"value"`
		FormattedCount   string  `json:"formatted_count"`
		RatingsBreakdown []struct {
			RatingValue int `json:"rating_value"`
			Percentage  int `json:"percentage"`
		} `json:"ratings_breakdown"`
		TooltipText    interface{} `json:"tooltip_text"`
		LastUserReview interface{} `json:"last_user_review"`
	} `json:"rating"`
	ShowNewSearch         bool          `json:"show_new_search"`
	ShowNearbyRestaurants bool          `json:"show_nearby_restaurants"`
	ZoneCode              string        `json:"zone_code"`
	UnavailableItems      []interface{} `json:"unavailable_items"`
	TippingEnabled        bool          `json:"tipping_enabled"`
	TippingDetail         interface{}   `json:"tipping_detail"`
	Restaurant            struct {
		ID                          int         `json:"id"`
		Name                        string      `json:"name"`
		CustomerCollectionSupported bool        `json:"customer_collection_supported"`
		SupportedFulfillmentMethods []string    `json:"supported_fulfillment_methods"`
		EnabledFulfillmentMethods   []string    `json:"enabled_fulfillment_methods"`
		NameWithBranch              string      `json:"name_with_branch"`
		BrandName                   string      `json:"brand_name"`
		BrandUname                  string      `json:"brand_uname"`
		Description                 string      `json:"description"`
		NewlyAdded                  bool        `json:"newly_added"`
		Uname                       string      `json:"uname"`
		PriceCategory               interface{} `json:"price_category"`
		CurrencySymbol              string      `json:"currency_symbol"`
		Menu                        struct {
			MenuTags []struct {
				Type string `json:"type"`
				Name string `json:"name"`
			} `json:"menu_tags"`
		} `json:"menu"`
		OpensAt       string `json:"opens_at"`
		ClosesAt      string `json:"closes_at"`
		StreetAddress string `json:"street_address"`
		PostCode      string `json:"post_code"`
		Neighborhood  string `json:"neighborhood"`
		PhoneNumbers  struct {
			Primary   string      `json:"primary"`
			Secondary interface{} `json:"secondary"`
		} `json:"phone_numbers"`
		AcceptsAllergyNotes        bool   `json:"accepts_allergy_notes"`
		City                       string `json:"city"`
		Open                       bool   `json:"open"`
		Image                      string `json:"image"`
		CountrywideAllergyWarnings string `json:"countrywide_allergy_warnings"`
		CompanyInfo                struct {
		} `json:"company_info"`
		ShowCompanyInfo   bool        `json:"show_company_info"`
		AllergyWarnings   string      `json:"allergy_warnings"`
		CustomAllergyNote interface{} `json:"custom_allergy_note"`
	} `json:"restaurant"`
	Menu struct {
		ID    int `json:"id"`
		Offer struct {
			Mov float64 `json:"mov"`
		} `json:"offer"`
		PromotedItemsCarousel interface{} `json:"promoted_items_carousel"`
		Categories            []struct {
			ID          int    `json:"id"`
			Description string `json:"description"`
			Name        string `json:"name"`
			SortOrder   int    `json:"sort_order"`
			TopLevel    bool   `json:"top_level"`
			UniqueID    int    `json:"unique_id"`
		} `json:"categories"`
		OffersVisibilityInformation struct {
		} `json:"offers_visibility_information"`
		ModifierGroups []struct {
			ID                    int    `json:"id"`
			Name                  string `json:"name"`
			Instruction           string `json:"instruction"`
			MinSelectionPoints    int    `json:"min_selection_points"`
			MaxSelectionPoints    int    `json:"max_selection_points"`
			AllowMultipleSameItem bool   `json:"allow_multiple_same_item"`
			PriceStrategy         string `json:"price_strategy"`
			ModifierOptions       []struct {
				ID                 int           `json:"id"`
				UID                int           `json:"uid"`
				Name               string        `json:"name"`
				Description        interface{}   `json:"description"`
				ProductInformation interface{}   `json:"product_information"`
				ModifierGroups     []interface{} `json:"modifier_groups"`
				Price              float64       `json:"price"`
				PriceUnit          string        `json:"price_unit"`
				AltModPrice        float64       `json:"alt_mod_price"`
			} `json:"modifier_options"`
		} `json:"modifier_groups"`
		HideMenuCategoryIds []int `json:"hide_menu_category_ids"`
		Items               []struct {
			ID                  int           `json:"id"`
			UID                 int           `json:"uid"`
			Name                string        `json:"name"`
			Description         interface{}   `json:"description"`
			Price               string        `json:"price"`
			RawPrice            float64       `json:"raw_price"`
			RawDiscountedPrice  float64       `json:"raw_discounted_price"`
			DiscountedPrice     interface{}   `json:"discounted_price"`
			PriceUnit           string        `json:"price_unit"`
			AltModPrice         float64       `json:"alt_mod_price"`
			SortOrder           int           `json:"sort_order"`
			Popular             bool          `json:"popular"`
			Alcohol             bool          `json:"alcohol"`
			ProductInformation  string        `json:"product_information"`
			CategoryID          int           `json:"category_id"`
			Image               interface{}   `json:"image"`
			ModifierGroups      []interface{} `json:"modifier_groups"`
			Available           bool          `json:"available"`
			DiscountTag         interface{}   `json:"discount_tag"`
			ModifierInfoMessage interface{}   `json:"modifier_info_message"`
		} `json:"items"`
		Footnotes []string `json:"footnotes"`
		Carousels []struct {
			ID                      string `json:"id"`
			Style                   string `json:"style"`
			Title                   string `json:"title"`
			Description             string `json:"description"`
			MenuItems               []int  `json:"menu_items"`
			ViewedCarouselEventName string `json:"viewed_carousel_event_name"`
			HideMenuCategoryIds     []int  `json:"hide_menu_category_ids"`
		} `json:"carousels"`
		HasDietaryInfo bool `json:"has_dietary_info"`
	} `json:"menu"`
	SeenCookieMessage            bool `json:"seen_cookie_message"`
	ShouldCheckCookieConsent     bool `json:"should_check_cookie_consent"`
	ShouldUseNewOnetrustWorkflow bool `json:"should_use_new_onetrust_workflow"`
	AppBanner                    struct {
		Ios struct {
			Rating  float64 `json:"rating"`
			Reviews string  `json:"reviews"`
		} `json:"ios"`
		Android struct {
			Rating  float64 `json:"rating"`
			Reviews string  `json:"reviews"`
		} `json:"android"`
	} `json:"app_banner"`
	ShowRatingsBreakdown bool        `json:"show_ratings_breakdown"`
	ShowPricesInline     bool        `json:"show_prices_inline"`
	User                 interface{} `json:"user"`
	CurrentCountry       struct {
		IsoCode string `json:"iso_code"`
		Tld     string `json:"tld"`
	} `json:"current_country"`
	GoogleAPIKey                 string      `json:"google_api_key"`
	UseBrandSearch               bool        `json:"use_brand_search"`
	BrandSearchEnabled           bool        `json:"brand_search_enabled"`
	FulfillmentType              string      `json:"fulfillment_type"`
	RestaurantFulfilledEducation interface{} `json:"restaurant_fulfilled_education"`
	RecyclablePackagingEducation interface{} `json:"recyclable_packaging_education"`
	DisabledButtonText           string      `json:"disabled_button_text"`
	AllergyWarningContent        interface{} `json:"allergy_warning_content"`
	OrderType                    string      `json:"order_type"`
	FulfillmentMethod            string      `json:"fulfillment_method"`
	HygieneContent               struct {
		Heading     string `json:"heading"`
		Description string `json:"description"`
		LinkText    string `json:"link_text"`
		LinkHref    string `json:"link_href"`
	} `json:"hygiene_content"`
	RestaurantInfoBlocks []struct {
		Heading         string      `json:"heading"`
		Description     string      `json:"description"`
		LinkText        string      `json:"link_text"`
		LinkURL         string      `json:"link_url"`
		ImageSrc        interface{} `json:"image_src"`
		ImageDimensions interface{} `json:"image_dimensions"`
		ImageAltText    interface{} `json:"image_alt_text"`
		TrackingEvent   string      `json:"tracking_event"`
	} `json:"restaurant_info_blocks"`
	PlusRestaurantEligibility      interface{} `json:"plus_restaurant_eligibility"`
	IsPlusRebrandEnabled           bool        `json:"is_plus_rebrand_enabled"`
	NewAddressFlowEnabled          bool        `json:"new_address_flow_enabled"`
	ScheduledRangesEnabled         bool        `json:"scheduled_ranges_enabled"`
	ShowEmployeeCwaMenu            bool        `json:"show_employee_cwa_menu"`
	UILineOffersProgressBarEnabled bool        `json:"ui_line_offers_progress_bar_enabled"`
	ActiveCountries                []struct {
		Name          string `json:"name"`
		LocalizedName string `json:"localized_name"`
		Host          string `json:"host"`
		Tld           string `json:"tld"`
	} `json:"active_countries"`
	AvailableLocales         []string      `json:"available_locales"`
	Currentlocale            string        `json:"currentLocale"`
	ShowLogin                bool          `json:"show_login"`
	EnableBrowserGeolocation bool          `json:"enable_browser_geolocation"`
	ClickAndCollectEnabled   bool          `json:"click_and_collect_enabled"`
	DineInEnabled            bool          `json:"dine_in_enabled"`
	GroupOrderingEnabled     bool          `json:"group_ordering_enabled"`
	CacheableMode            bool          `json:"cacheable_mode"`
	CsrfToken                string        `json:"csrf_token"`
	ThemeClass               string        `json:"theme_class"`
	ThemeName                string        `json:"theme_name"`
	WhiteLabelBrand          interface{}   `json:"white_label_brand"`
	WhiteLabelConfig         interface{}   `json:"white_label_config"`
	PastOrders               []interface{} `json:"past_orders"`
	MenuDisabled             bool          `json:"menu_disabled"`
	Payment                  struct {
		Credit struct {
			CurrentAmount      float64       `json:"current_amount"`
			DebitAmount        float64       `json:"debit_amount"`
			RemainingAmount    float64       `json:"remaining_amount"`
			CurrentAmountFmt   string        `json:"current_amount_fmt"`
			DebitAmountFmt     string        `json:"debit_amount_fmt"`
			RemainingAmountFmt string        `json:"remaining_amount_fmt"`
			Credits            []interface{} `json:"credits"`
		} `json:"credit"`
		Allowance struct {
			DebitAmount        float64 `json:"debit_amount"`
			DebitAmountFmt     string  `json:"debit_amount_fmt"`
			RemainingAmount    float64 `json:"remaining_amount"`
			RemainingAmountFmt string  `json:"remaining_amount_fmt"`
		} `json:"allowance"`
		Outstanding struct {
			DebitAmount    float64 `json:"debit_amount"`
			DebitAmountFmt string  `json:"debit_amount_fmt"`
		} `json:"outstanding"`
		AcceptsCash bool `json:"accepts_cash"`
	} `json:"payment"`
}
