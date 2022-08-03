\c deliveroo

DELETE FROM customers_to_restaurants
WHERE restaurant_id IN (
    SELECT t1.restaurant_id
    FROM customers_to_restaurants t1
        LEFT JOIN restaurants t2 ON t1.restaurant_id = t2.url
    WHERE t2.url IS NULL
);

ALTER TABLE customers_to_restaurants 
ADD CONSTRAINT cust_rest_fk FOREIGN KEY (restaurant_id) REFERENCES restaurants (url)
ON UPDATE CASCADE ON DELETE CASCADE;
