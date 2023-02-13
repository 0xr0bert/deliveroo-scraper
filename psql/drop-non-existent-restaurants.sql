-- This script should be executed after the scraper restaurants has been ran many times. If the logs indicate that the
-- remaining restaurants are invalid then we drop them.

begin;
delete from locations_to_restaurants t1
    using restaurants t2
    where t1.restaurant_id = t2.id and t2.visited_time is null;
delete from restaurants
    where visited_time is null;
commit;