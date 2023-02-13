-- This should be executed after the locations scrape has been done.
insert into restaurants(id)
select distinct restaurant_id as id from locations_to_restaurants
on conflict do nothing;

insert into tags_visited_time(restaurant_id)
select id from restaurants
on conflict do nothing;