import got from 'got';
import ngeohash from 'ngeohash';
import { load } from 'cheerio';
import Bottleneck from 'bottleneck';
import { MIN_TIME, MAX_CONCURRENT } from './config.js';
import pg from 'pg';
import format from 'pg-format';

export interface SuccessfulResult {
    id: string;
    restaurants: [string];
}

export interface Location {
    id: string;
    longitude: number;
    latitude: number;
}

const limiter = new Bottleneck({
    minTime: MIN_TIME,
    maxConcurrent: MAX_CONCURRENT,
});

const getRestaurantsW = limiter.wrap(getRestaurants);

export async function getRestaurants(location: Location) {
    const geohash = ngeohash.encode(location.latitude, location.longitude);
    const res = await got.get(`https://deliveroo.co.uk/restaurants/london/canonbury?geohash=${geohash}&collection=all-restaurants`)
    const $ = load(res.body);
    const data = $("script#__NEXT_DATA__").html();
    if (data !== null) {
        const jData = JSON.parse(data);

        const ids = jData.props.initialState.home.feed.results.data[0].blocks.map((b: any) => b.target.restaurant?.id);
        return { id: location.id, restaurants: ids };
    } else {
        console.log(res);
        return { id: location.id, restaurants: [] }
    }
}

export async function processSuccessfulResult(result: SuccessfulResult, client: pg.PoolClient) {
    try {
        const insertData = result.restaurants.filter(id => id).map((id) => [result.id, id]);
        await client.query("BEGIN");
        if (insertData.length) {
            const res = await client.query(
                format(`
                    INSERT INTO locations_to_restaurants(location_id, restaurant_id)
                    VALUES %L
                    ON CONFLICT DO NOTHING
                `, insertData)
            );
        }
        await client.query("UPDATE locations SET visited_time = $1 WHERE id = $2", [new Date(), result.id]);
        await client.query("COMMIT");
    } catch (e) {
        await client.query("ROLLBACK");
        throw e;
    } finally {
        client.release();
    }
}

export async function getAndProcessLocation(location: Location, client: pg.PoolClient) {
    const res = await getRestaurantsW(location);
    await processSuccessfulResult(res, client);
}

const pool = new pg.Pool({
    user: "postgres",
    host: "localhost",
    password: "postgres",
    database: "postgres",
    port: 15434,
});

const client = await pool.connect();
try {
    const res = await client.query(
        "SELECT id, longitude, latitude FROM locations WHERE visited_time IS NULL"
    );
    const promises = res.rows.map(async row => getAndProcessLocation(row, await pool.connect()));
    await Promise.all(promises);
} finally {
    client.release();
}