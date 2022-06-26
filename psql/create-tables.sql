create table if not exists locations (
    id varchar(10) primary key,
    name text not null,
    region text not null,
    latitude double precision not null,
    longitude double precision not null,
    visited_time timestamptz
);
create table if not exists locations_to_restaurants(
    location_id varchar(10) not null,
    restaurant_id varchar(10) not null,
    primary key(location_id, restaurant_id)
);
create table if not exists restaurants_to_categories(
    restaurant_id varchar(10) not null,
    category_name text not null,
    category_type text not null,
    primary key(restaurant_id, category_name, category_type)
);
create table if not exists restaurants(
    id varchar(10) primary key,
    visited_time timestamptz,
    drn_id uuid,
    menu_id text,
    name text,
    branch_type text,
    accepts_allergy_notes boolean,
    cash_tipping_message text,
    fulfillment_type text,
    currency_code text,
    currency_symbol text,
    menu_disabled boolean,
    location__city_id int,
    location__zone_id int,
    address__formatted text,
    address__address1 text,
    address__postcode text,
    address__neighborhood text,
    address__city text,
    address__country text,
    address__lat double precision,
    address__lon double precision
);
create table if not exists modifier_groups(
    id varchar(20) primary key,
    restaurant_id varchar(10) not null,
    name text,
    description text,
    min_selection int,
    max_selection int,
    repeatable boolean,
    constraint fk_restaurant_id foreign key(restaurant_id) references restaurants(id) on delete cascade
);
create table if not exists modifier_options(
    id varchar(20) primary key,
    restaurant_id varchar(10) not null,
    modifier_group_id varchar(20),
    uid int,
    name text,
    description text,
    price__code text,
    price__fractional int,
    price__formatted text,
    price__presentational text,
    price_discounted__code text,
    price_discounted__fractional int,
    price_discounted__formatted text,
    price_discounted__presentational text,
    available boolean,
    constraint fk_restaurant_id foreign key(restaurant_id) references restaurants(id) on delete cascade,
    constraint fk_modifier_group_id foreign key(modifier_group_id) references modifier_groups(id) on delete cascade
);
create table if not exists items(
    id varchar(20) primary key,
    restaurant_id varchar(10) not null,
    uid int,
    category_id varchar(20),
    name text,
    description text,
    product_information text,
    price__code text,
    price__fractional int,
    price__formatted text,
    price__presentational text,
    price_discounted__code text,
    price_discounted__fractional int,
    price_discounted__formatted text,
    price_discounted__presentational text,
    percentage_discounted text,
    calories text,
    available boolean,
    popular boolean,
    alcohol boolean,
    max_selection int,
    is_signature_exclusive boolean,
    nutritional_info__energy int,
    nutritional_info__energy_formatted text,
    constraint fk_restaurant_id foreign key(restaurant_id) references restaurants(id) on delete cascade
);
create table if not exists items_to_modifier_groups(
    item_id varchar(20) not null,
    modifier_group_id varchar(20) not null,
    restaurant_id varchar(10) not null,
    primary key(item_id, modifier_group_id, restaurant_id),
    constraint fk_item_id foreign key(item_id) references items(id) on delete cascade,
    constraint fk_restaurant_id foreign key(restaurant_id) references restaurants(id) on delete cascade,
    constraint fk_modifier_group_id foreign key(modifier_group_id) references modifier_groups(id) on delete cascade
);
create table if not exists categories(
    id varchar(20) primary key,
    restaurant_id varchar(10) not null,
    name text,
    description text,
    constraint fk_restaurant_id foreign key(restaurant_id) references restaurants(id) on delete cascade
);
create table if not exists items_to_categories(
    item_id varchar(20) not null,
    category_id varchar(20) not null,
    restaurant_id varchar(10) not null,
    primary key(item_id, category_id, restaurant_id),
    constraint fk_item_id foreign key(item_id) references items(id) on delete cascade,
    constraint fk_restaurant_id foreign key(restaurant_id) references restaurants(id) on delete cascade,
    constraint fk_category_id foreign key(category_id) references categories(id) on delete cascade
);