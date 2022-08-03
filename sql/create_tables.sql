CREATE DATABASE deliveroo;
\c deliveroo

CREATE TABLE tag_types (
    name    text PRIMARY KEY
);

CREATE TABLE tags (
    name    text PRIMARY KEY,
    tag_type_id text REFERENCES tag_types (name) ON UPDATE CASCADE ON DELETE RESTRICT NOT NULL
);

CREATE TABLE restaurants (
    url     text PRIMARY KEY,
    name    text NOT NULL,
    avg_rating  real,
    address text NOT NULL,
    description text
);

CREATE TABLE tags_restaurants (
    id      serial PRIMARY KEY,
    tag_id  text REFERENCES tags (name) ON UPDATE CASCADE ON DELETE CASCADE NOT NULL,
    restaurant_id   text REFERENCES restaurants (url) ON UPDATE CASCADE ON DELETE CASCADE NOT NULL
);

CREATE TABLE customers (
    postcode    varchar(10) PRIMARY KEY
);

CREATE TABLE customers_to_restaurants (
    id      serial PRIMARY KEY,
    customer_id     text REFERENCES customers (postcode) ON UPDATE CASCADE ON DELETE CASCADE NOT NULL,
    restaurant_id   text 
);

CREATE TABLE menu_categories (
    id      serial PRIMARY KEY,
    name    text NOT NULL,
    restaurant_id   text REFERENCES restaurants (url) ON UPDATE CASCADE ON DELETE CASCADE NOT NULL
);

CREATE TABLE items (
    id      serial PRIMARY KEY,
    name    text NOT NULL,
    price   real NOT NULL,
    is_popular  boolean NOT NULL,
    menu_category_id    int REFERENCES menu_categories (id) ON UPDATE CASCADE ON DELETE CASCADE NOT NULL
);
