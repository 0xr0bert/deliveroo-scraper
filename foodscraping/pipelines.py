import sqlite3
import uuid


class DeliverooPipeline:
    def __init__(self, sqlite_db_loc):
        self.sqlite_db_loc = sqlite_db_loc

    @classmethod
    def from_crawler(cls, crawler):
        return cls(sqlite_db_loc=crawler.settings.get('DB_LOC'))

    def open_spider(self, spider):
        self.conn = sqlite3.connect(self.sqlite_db_loc)

    def close_spider(self, spider):
        self.conn.close()

    def process_item(self, item, spider):
        cur = self.conn.cursor()
        if "restaurants" in item:
            postcode = item["restaurants"]["postcode"]
            urls = item["restaurants"]["urls"]
            insert_values = list(
                zip((postcode for i in range(len(urls))), urls)
            )

            # Insert into DB
            cur.executemany(
                """INSERT INTO customers_to_restaurants
                (customer_id, restaurant_id) VALUES (?, ?)""",
                insert_values
            )

        elif "menu" in item:
            menu_data = item["menu"]
            restaurant_table_data = {}
            restaurant_table_data["url"] = menu_data["url"]
            restaurant_table_data["name"] = menu_data["name"]
            restaurant_table_data["avg_rating"] = menu_data["avg_rating"]
            restaurant_table_data["address"] = menu_data["street_address"]
            restaurant_table_data["description"] = menu_data["description"]

            cur.execute(
                """
                INSERT INTO restaurants (url, name, avg_rating, address,
                description) VALUES (:url, :name, :avg_rating, :address,
                :description)
                """,
                restaurant_table_data
            )

            # Add menu categories
            for category in menu_data["menu_categories"]:
                category_id = str(uuid.uuid4())
                cur.execute(
                    """
                    INSERT INTO menu_categories (id, name, restaurant_id)
                    VALUES (?, ?, ?)
                    """,
                    (category_id, category["name"],
                     restaurant_table_data["url"])
                )

                # Add items
                items = []
                for product_item in category["products"]:
                    item_data = {}
                    item_data["name"] = product_item["name"]
                    item_data["price"] = product_item["price"]
                    item_data["is_popular"] = product_item["is_popular"]
                    item_data["menu_category_id"] = category_id
                    items.append(item_data)

                cur.executemany(
                    """
                    INSERT INTO items (name, price, is_popular,
                    menu_category_id) VALUES (:name, :price, :is_popular,
                    :menu_category_id)
                    """,
                    items
                )

            # Handle tags
            tag_types = set((t["type"],) for t in menu_data["tags"])
            cur.executemany(
                "INSERT OR IGNORE INTO tag_types (name) VALUES (?)",
                tag_types
            )

            cur.executemany(
                """
                INSERT OR IGNORE INTO tags (name, tag_type_id) VALUES (:name,
                :type)""",
                menu_data["tags"]
            )

            tag_names = set(t["name"] for t in menu_data["tags"])
            insert_values = list(
                zip((restaurant_table_data["url"] for i in range(
                    len(tag_names))),
                    tag_names)
            )
            cur.executemany(
                """
                INSERT INTO tags_restaurants (restaurant_id, tag_id) VALUES
                (?, ?)
                """,
                insert_values
            )

        self.conn.commit()
        cur.close()

        return item
