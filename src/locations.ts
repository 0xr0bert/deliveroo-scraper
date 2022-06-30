import got from 'got';
import ngeohash from 'ngeohash';
import {load} from 'cheerio';
import Bottleneck from 'bottleneck';
import {MIN_TIME, MAX_CONCURRENT} from './config.js';
import pg from 'pg';
import format from 'pg-format';

/**
 * A successful result from the API.
 */
export interface SuccessfulResult {
  /**
   * The ID of the location.
   */
  id: string;
  /**
   * The restaurant IDs available at the location.
   */
  restaurants: Array<string>;
}

/**
 * A Location.
 */
export interface Location {
  /**
   * The ID of the location.
   */
  id: string;
  /**
   * The longitude of the location.
   */
  longitude: number;
  /**
   * The latitude of the location.
   */
  latitude: number;
}

/**
 * The Bottleneck used.
 */
const limiter: Bottleneck = new Bottleneck({
  minTime: MIN_TIME,
  maxConcurrent: MAX_CONCURRENT,
});

/**
 * The limited getRestaurants.
 */
const getRestaurantsW = limiter.wrap(getRestaurants);

/**
 * Gets the restaurants available at a location.
 * @param {Location} location The Location to scrape.
 * @return {Promise<SuccessfulResult>} The successful result.
 */
export async function getRestaurants(
    location: Location): Promise<SuccessfulResult> {
  const geohash = ngeohash.encode(location.latitude, location.longitude);
  const res = await got.get(`https://deliveroo.co.uk/restaurants/london/canonbury?geohash=${geohash}&collection=all-restaurants`);
  const $ = load(res.body);
  const data = $('script#__NEXT_DATA__').html();
  if (data !== null) {
    const jData = JSON.parse(data);

    const ids = jData
        .props
        .initialState
        .home
        .feed
        .results
        .data
        .map((d: any) => d.blocks.map((b: any) => b.target?.restaurant?.id))
        .flat()
        .filter((e: any) => e !== undefined);
    return {id: location.id, restaurants: ids};
  } else {
    console.log(res);
    return {id: location.id, restaurants: <string[]>[]};
  }
}

/**
 * Process a successful result.
 * @param {SuccessfulResult} result The Successful Result.
 * @param {pg.PoolClient} client The postgres client.
 */
export async function processSuccessfulResult(
    result: SuccessfulResult,
    client: pg.PoolClient) {
  try {
    const insertData = result.restaurants.filter(
        (id) => id).map((id) => [result.id, id],
    );
    await client.query('BEGIN');
    if (insertData.length) {
      await client.query(
          format(`
              INSERT INTO locations_to_restaurants(location_id, restaurant_id)
              VALUES %L
              ON CONFLICT DO NOTHING
              `, insertData),
      );
    }
    await client.query(
        'UPDATE locations SET visited_time = $1 WHERE id = $2',
        [new Date(), result.id],
    );
    await client.query('COMMIT');
  } catch (e) {
    await client.query('ROLLBACK');
    throw e;
  } finally {
    client.release();
  }
}

/**
 * Get and process a location.
 * @param {Location} location The location.
 * @param {pg.PoolClient} client The postgres client.
 */
export async function getAndProcessLocation(
    location: Location,
    client: pg.PoolClient) {
  const res = await getRestaurantsW(location);
  await processSuccessfulResult(res, client);
}

/**
 * The postgres pool.
 */
const pool: pg.Pool = new pg.Pool({
  user: 'postgres',
  host: 'db',
  password: 'postgres',
  database: 'postgres',
  port: 5432,
});

/**
 * The postgres client used to get the unvisited IDs.
 */
const client: pg.PoolClient = await pool.connect();
try {
  /**
   * The results from the SQL database.
   */
  const res: pg.QueryResult<Location> = await client.query(
      `SELECT id, longitude, latitude
       FROM locations WHERE visited_time IS NULL`,
  );
  /**
   * The promises for getAndProcessLocation.
   */
  const promises: Array<Promise<void>> = res.rows.map(
      async (row) => getAndProcessLocation(row, await pool.connect()),
  );
  await Promise.all(promises);
} finally {
  client.release();
}
