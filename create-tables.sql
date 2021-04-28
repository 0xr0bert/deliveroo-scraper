create table customers_to_restaurants(customer_id varchar(8), restaurant_id text, primary key (customer_id, restaurant_id));
create table restaurants(url text primary key, name text, avg_rating numeric(2,1), address text, description text);
create table tag_types(name text primary key);
create table tags(name text primary key, tag_type_id text);
create table tags_restaurants(tag_id text, restaurant_id text, primary key(tag_id, restaurant_id));
create table menu_categories(id int, name text, description text, restaurant_id text, primary key (id, restaurant_id));
create table items(id int, name text, price money, is_popular boolean, menu_category_id int, primary key (id, menu_category_id));