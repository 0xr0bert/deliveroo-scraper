import got from "got";
import pg from 'pg';
import format from 'pg-format';
import Bottleneck from "bottleneck";
import { MAX_CONCURRENT, MIN_TIME } from "./config";

export interface MenuTag {
    type: string;
    name: string;
}

const limiter = new Bottleneck({
    minTime: MIN_TIME,
    maxConcurrent: MAX_CONCURRENT,
});
const getTagsW = limiter.wrap(getTags);

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

export async function writeTags(restaurantId: number, tags: Array<MenuTag>, client: pg.PoolClient) {
    try {
        const insertData = tags.map(t => [restaurantId, t.name, t.type]);
        if (insertData.length) {
            await client.query(
                format(`INSERT INTO restaurants_to_categories(
                    restaurant_id, category_name, category_type
                ) VALUES %L ON CONFLICT DO NOTHING`, insertData)
            )
        }
    } finally {
        client.release();
    }
}

export async function getAndWriteTags(restaurantId: number, client: pg.PoolClient) {
    const res = await getTagsW(restaurantId);
    await writeTags(restaurantId, res, client);
}

const pool = new pg.Pool({
    user: "postgres",
    host: "db",
    password: "postgres",
    database: "postgres",
    port: 5432
});

const client = await pool.connect();
try {
    const res = await client.query(`
        SELECT id
        FROM restaurants r
        WHERE NOT EXISTS (
            SELECT
            FROM restaurants_to_categories
            WHERE restaurant_id = r.id
        ) 
    `);
    const promises = res.rows.map(async row => getAndWriteTags(row.id, await pool.connect()));
    await Promise.all(promises);
} finally {
    client.release();
}