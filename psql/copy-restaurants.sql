insert into restaurants(id)
select distinct restaurant_id as id from locations_to_restaurants
on conflict do nothing;
