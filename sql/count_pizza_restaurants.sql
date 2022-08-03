-- This counts the number of restaurants and pizza restaurants.
SELECT postcode, COUNT(DISTINCT url) as n_restaurants,
COUNT(tags.name) filter (where tags.name = 'Pizza') as n_pizza FROM (
    customers INNER JOIN customers_to_restaurants
     ON customers.postcode = customers_to_restaurants.customer_id INNER JOIN 
    restaurants ON customers_to_restaurants.restaurant_id = restaurants.url
    FULL OUTER JOIN tags_restaurants ON restaurants.url = tags_restaurants.restaurant_id
    FULL OUTER JOIN tags ON tags_restaurants.tag_id = tags.name
) GROUP BY postcode;

-- This counts the number of occurances per tag per postcode.
SELECT postcode, tags.name as tag, COUNT(*) as count
FROM (
    customers INNER JOIN customers_to_restaurants
     ON customers.postcode = customers_to_restaurants.customer_id INNER JOIN 
    restaurants ON customers_to_restaurants.restaurant_id = restaurants.url
    INNER JOIN tags_restaurants ON restaurants.url = tags_restaurants.restaurant_id
    INNER JOIN tags ON tags_restaurants.tag_id = tags.name
) GROUP BY postcode, tags.name
ORDER BY postcode ASC, count DESC;
