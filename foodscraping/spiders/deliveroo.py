import scrapy
import re
import sqlite3
from foodscraping.utilities import clean_pounds


class DeliverooSpider(scrapy.Spider):
    name = "deliveroo"
    custom_settings = {
        'DB_LOC': 'deliveroo.db'
    }

    def start_requests(self):
        conn = sqlite3.connect(self.settings.get("DB_LOC"))
        cur = conn.cursor()
        cur.execute("SELECT postcode FROM customers")
        postcodes = cur.fetchall()
        cur.close()
        conn.close()

        urls = map(lambda x: "https://deliveroo.co.uk/restaurants/london/" +
                   f"camden?postcode={x[0]}&sort=time", postcodes)

        for url in urls:
            yield scrapy.Request(url=url, callback=self.parse)

    def parse(self, response):
        links = response.css("a::attr(href)").getall()

        # Only get menu links
        links = filter(lambda link: link.startswith("/menu"), links)

        # Get absolute URLS
        links = list(map(response.urljoin, links))
        # Links w/o query params
        clean_links = list(map(lambda link: link.split("?")[0], links))

        # Get postcode
        postcode = re.search(
            r"(?<=(postcode=)).*(?=&)",
            response.request.url
        ).group()

        # Yield the clean links
        yield {
            "restaurants": {
                "urls": clean_links,
                "postcode": postcode
            }
        }

        for link in clean_links:
            yield scrapy.Request(link, callback=self.parse_menu)

    def parse_menu(self, response):
        data = {}

        # Handle URL
        url = response.request.url.split("?")[0]
        if "/menu" not in url:
            return
        data["url"] = url

        # Handle name
        data["name"] = response.css("h1.restaurant__name::text").get()

        # Handle avg rating
        avg_rating = response.css(
            "div.orderweb__61671603 span.ccl-b308a2db3758e3e5::text"
        ).get()

        try:
            avg_rating = float(avg_rating)
        except TypeError:
            avg_rating = None

        data["avg_rating"] = avg_rating

        # Handle tags
        tags = []

        for tag_content in response.css("small.tag"):
            tag = {}
            tag_type = tag_content.xpath("@class").extract()[0].split(" ")[1]
            tag["name"] = tag_content.css("::text").get()
            tag["type"] = tag_type

            tags.append(tag)

        data["tags"] = tags

        # Handle address
        data["street_address"] = response.css("small.address::text").get()

        # Handle description
        data["description"] = response.css(
            "div.restaurant__description *::text"
        ).get()

        # Handle menu categories
        menu_categories = []
        for category_content in \
                response.css("div.menu-index-page__menu-category"):
            category = {}
            category_name = category_content.css("h3::text").get()
            category["name"] = category_name

            # Handle items
            items = []
            for item_content in \
                    category_content.css("li.menu-index-page__item"):
                item = {}
                item["name"] = \
                    item_content.css("h6 *::text")\
                    .get()

                item_price_str =\
                    item_content.css("span.menu-index-page__item-price::text")\
                    .get()
                item["price"] = clean_pounds(item_price_str)

                item["is_popular"] =\
                    item_content.css(
                        "span.menu-index-page__item-popular::text").get()\
                    is not None

                items.append(item)

            category["products"] = items

            menu_categories.append(category)

        data["menu_categories"] = menu_categories

        if data["name"] is None:
            self.logger.error(f"ERROR on: {data['url']}")
            yield {
                "error": {
                    "url": data["url"]
                }
            }
        else:
            yield {"menu": data}
