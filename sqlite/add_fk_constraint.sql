DELETE FROM customers_to_restaurants
WHERE restaurant_id IN (
    SELECT t1.restaurant_id
    FROM customers_to_restaurants t1
        LEFT JOIN restaurants t2 ON t1.restaurant_id = t2.url
    WHERE t2.url IS NULL
);

BEGIN TRANSACTION;
CREATE TEMPORARY TABLE c_to_r_bk(customer_id, restaurant_id);
INSERT INTO c_to_r_bk SELECT customer_id, restaurant_id FROM customers_to_restaurants;
DROP TABLE customers_to_restaurants;

CREATE TABLE customers_to_restaurants (
    customer_id     TEXT    NOT NULL,
    restaurant_id   TEXT    NOT NULL,
    PRIMARY KEY (customer_id, restaurant_id),
    FOREIGN KEY (customer_id)
        REFERENCES customers (postcode)
        ON UPDATE CASCADE
        ON DELETE CASCADE,
    FOREIGN KEY (restaurant_id)
        REFERENCES restaurants (url)
        ON UPDATE CASCADE
        ON DELETE CASCADE
);

INSERT INTO customers_to_restaurants SELECT customer_id, restaurant_id FROM c_to_r_bk;
DROP TABLE c_to_r_bk;
COMMIT;