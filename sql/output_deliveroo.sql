\c deliveroo
\COPY items TO '/tmp/deliveroo_output/items.csv' WITH (FORMAT CSV, HEADER);
\COPY menu_categories TO '/tmp/deliveroo_output/menu_categories.csv' WITH (FORMAT CSV, HEADER);
\COPY restaurants TO '/tmp/deliveroo_output/restaurants.csv' WITH (FORMAT CSV, HEADER);
\COPY tags TO '/tmp/deliveroo_output/tags.csv' WITH (FORMAT CSV, HEADER);
\COPY tag_types TO '/tmp/deliveroo_output/tag_types.csv' WITH (FORMAT CSV, HEADER);
\COPY customers TO '/tmp/deliveroo_output/customers.csv' WITH (FORMAT CSV, HEADER);
\COPY tags_restaurants TO '/tmp/deliveroo_output/tags_restaurants.csv' WITH (FORMAT CSV, HEADER);
\COPY customers_to_restaurants TO '/tmp/deliveroo_output/customers_to_restaurants.csv' WITH (FORMAT CSV, HEADER);
