CREATE TABLE tag_types (
    name    TEXT    PRIMARY KEY
);

CREATE TABLE tags (
    name    TEXT    PRIMARY KEY,
    tag_type_id     TEXT    NOT NULL,
    FOREIGN KEY (tag_type_id)
        REFERENCES tag_types (name)
        ON UPDATE CASCADE
        ON DELETE CASCADE
);

CREATE TABLE restaurants (
    url     TEXT    PRIMARY KEY,
    name    TEXT    NOT NULL,
    avg_rating  REAL,
    address     TEXT NOT NULL,
    description     TEXT
);

CREATE TABLE tags_restaurants (
    tag_id  TEXT    NOT NULL,
    restaurant_id   TEXT NOT NULL,
    PRIMARY KEY (tag_id, restaurant_id),
    FOREIGN KEY (tag_id)
        REFERENCES tags (name)
        ON UPDATE CASCADE
        ON DELETE CASCADE,
    FOREIGN KEY (restaurant_id)
        REFERENCES restaurants (url)
        ON UPDATE CASCADE
        ON DELETE CASCADE
);

CREATE TABLE customers (
    postcode    TEXT   PRIMARY KEY
);

CREATE TABLE customers_to_restaurants (
    customer_id     TEXT    NOT NULL,
    restaurant_id   TEXT    NOT NULL,
    PRIMARY KEY (customer_id, restaurant_id),
    FOREIGN KEY (customer_id)
        REFERENCES customers (postcode)
        ON UPDATE CASCADE
        ON DELETE CASCADE
);

CREATE TABLE menu_categories (
    id  TEXT  PRIMARY KEY,
    name    TEXT    NOT NULL,
    restaurant_id   TEXT    NOT NULL,
    FOREIGN KEY (restaurant_id)
        REFERENCES restaurants (url)
        ON UPDATE CASCADE
        ON DELETE CASCADE
);

CREATE TABLE items (
    id  INTEGER     PRIMARY KEY,
    name    TEXT    NOT NULL,
    price   REAL    NOT NULL,
    is_popular  INTEGER     NOT NULL,
    menu_category_id    TEXT    NOT NULL,
    FOREIGN KEY (menu_category_id)
        REFERENCES menu_categories (id)
        ON UPDATE CASCADE
        ON DELETE CASCADE
);