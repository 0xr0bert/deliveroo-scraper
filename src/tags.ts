import got from 'got';
import pg from 'pg';
import format from 'pg-format';
import Bottleneck from 'bottleneck';
import {MAX_CONCURRENT, MIN_TIME} from './config.js';

/**
 * The MenuTag returned by the API.
 */
export interface MenuTag {
  /**
   * The type of the MenuTag.
   */
  type: string;
  /**
   * The name of the MenuTag.
   */
  name: string;
}

/**
 * The bottleneck used to limit the requests to the API.
 */
const limiter: Bottleneck = new Bottleneck({
  minTime: MIN_TIME,
  maxConcurrent: MAX_CONCURRENT,
});

/**
 * getTags wrapped with a limiter.
 */
const getTagsW = limiter.wrap(getTags);

/**
 * Gets the tags from the Deliveroo API.
 * @param {number} restaurantId The ID of the restaurant.
 * @return {Promise<Array<MenuTag>>} The MenuTags for thee restaurant.
 */
export async function getTags(restaurantId: number): Promise<Array<MenuTag>> {
  try {
    const res = (await got.get(`https://consumer-ow-api.deliveroo.com/orderapp/v1/restaurants/${restaurantId}`).json()) as any;
    const returnValue = res.menu?.menu_tags;
    if (returnValue !== undefined) {
      return returnValue;
    } else {
      return [];
    }
  } catch (e) {
    return [];
  }
}

/**
 * Write the tags to the postgres database.
 * @param {number} restaurantId The ID of the restaurant.
 * @param {Array<MenuTag>} tags The MenuTags.
 * @param {pg.PoolClient} client The postgres client.
 */
export async function writeTags(
    restaurantId: number,
    tags: Array<MenuTag>,
    client: pg.PoolClient) {
  try {
    const insertData = tags.map((t) => [restaurantId, t.name, t.type]);
    if (insertData.length) {
      await client.query(
          format(`INSERT INTO restaurants_to_categories(
                    restaurant_id, category_name, category_type
                ) VALUES %L ON CONFLICT DO NOTHING`, insertData),
      );
    }
  } finally {
    client.release();
  }
}

/**
 * Gets and writes the tags to the postgres database.
 * @param {number} restaurantId The ID of the restaurant.
 * @param {pg.PoolClient} client The postgres client.
 */
export async function getAndWriteTags(
    restaurantId: number,
    client: pg.PoolClient) {
  const res = await getTagsW(restaurantId);
  await writeTags(restaurantId, res, client);
}

/**
 * The Postgres pool.
 */
const pool: pg.Pool = new pg.Pool({
  user: 'postgres',
  host: 'db',
  password: 'postgres',
  database: 'postgres',
  port: 5432,
});

/**
 * The postgres client for getting IDs.
 */
const client: pg.PoolClient = await pool.connect();
try {
  /**
   * The IDs with no categories yet.
   */
  const res: pg.QueryResult<SQLResult> = await client.query(`
        SELECT id
        FROM restaurants r
        WHERE NOT EXISTS (
            SELECT
            FROM restaurants_to_categories
            WHERE restaurant_id = r.id
        ) 
    `);

  /**
   * An array of promises for getting and writing tags.
   */
  const promises: Array<Promise<void>> = res.rows.map(
      async (row) => getAndWriteTags(row.id, await pool.connect()),
  );
  await Promise.all(promises);
} finally {
  client.release();
}

/**
 * An SQL result.
 */
export interface SQLResult {
  /**
   * The ID of a restaurant.
   */
  id: number;
}
