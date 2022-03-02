import got from 'got';
import { BASE_URL, MAX_CONCURRENT, MIN_TIME, RESTAURANTS_URL } from './config.js';
import {v4 as uuidv4} from 'uuid';
import pg from 'pg';
import format from 'pg-format';
import Bottleneck from 'bottleneck';
import { promisify } from 'util';
import {CookieJar} from 'tough-cookie';

const limiter = new Bottleneck({
    minTime: MIN_TIME,
    maxConcurrent: MAX_CONCURRENT,
});
const getRestaurantW = limiter.wrap(getRestaurant);

export async function getRestaurant(restaurantId: string): Promise<any> {
    const cookieJar = new CookieJar();
    const setCookie = promisify(cookieJar.setCookie.bind(cookieJar));

    await setCookie(`roo_guid=${uuidv4()}`, BASE_URL);
    await setCookie(`roo_super_properties=eyJBcHBzRmx5ZXJJRCI6IjE2NDMzNzgwMjU2NTYtNjgzNzc5MjQwNzY3MDQ5MjcyNyIsImNvbnRleHQiOnsidXNlckFnZW50IjoiRGVsaXZlcm9vLzMuODUuMCAoR29vZ2xlIHNka19ncGhvbmVfeDg2O0FuZHJvaWQgMTE7ZW4tVVM7cmVsZWFzZUVudiByZWxlYXNlKSIsImlwIjoiMTQxLjk4LjI1Mi4xNjgifSwiUmVxdWVzdGVkIExvY2FsZSI6ImVuIiwiUm9vQnJvd3NlciI6IkdlbmVyaWMgQnJvd3NlciIsIlJvb0Jyb3dzZXJWZXJzaW9uIjoiMCIsIlRMRCI6InVrIiwid2hpdGVfbGFiZWxfYnJhbmQiOiJjb3JlIiwiQXBwIE5hbWVzcGFjZSI6ImNvbS5kZWxpdmVyb28ub3JkZXJhcHAiLCJBcHAgVmVyc2lvbiI6IjMuODUuMCIsIlBsYXRmb3JtIjoiQW5kcm9pZCIsIk9TIFZlcnNpb24iOiIxMSIsIkRldmljZSBNb2RlbCI6Ikdvb2dsZSBzZGtfZ3Bob25lX3g4NiIsIkRldmljZSBUeXBlIjoiUGhvbmUiLCJMb2NhbGUiOiJlbl9VUyIsIklERlYiOiIxYjUzOWNmNzEzMzRlZTJjIiwiSURGQSI6Ijc1MzM3MzIwLTQxZmQtNDE4Mi05YjMxLTkyZDM1ZjU0NjUyNSIsIkdvb2dsZSBQYXkgU3RhdHVzIjoidW5hdmFpbGFibGUiLCJEZXZpY2UgTG9jYWxlIjoiZW5fVVMiLCJEZXZpY2UgTGFuZ3VhZ2UiOiJlbi1VUyIsImZleF80MF9kdXJpbmdfY2hlY2tvdXRfdXBkYXRlIjoiZmVhdHVyZSJ9`, BASE_URL);
    const res = await got.post(
        RESTAURANTS_URL,
        {
            cookieJar,
            json: {
                variables: {
                    options: {
                        restaurant_id: restaurantId,
                        location: {},
                        fulfillment_method: "DELIVERY"
                    },
                    requestUuid: uuidv4(),
                },
                query: `
query GetMenuPage($options: MenuOptionsInput, $requestUuid: String) {
  get_menu_page(options:$options, request_uuid:$requestUuid) {
    meta {
      categories {
        id,
        name,
        description,
        item_ids
      },
      items {
        id,
        uid,
        category_id,
        name,
        description,
        product_information,
        price {
          code,
          fractional,
          formatted,
          presentational
        },
        price_discounted {
          code,
          fractional,
          formatted,
          presentational
        },
        percentage_discounted,
        calories,
        available,
        popular,
        alcohol,
        modifier_group_ids,
        max_selection,
        is_signature_exclusive
      },
      modifier_groups {
        id,
        name,
        description,
        min_selection,
        max_selection,
        repeatable,
        modifier_options {
          id,
          uid,
          name,
          description,
          price {
            code,
            fractional,
            formatted,
            presentational
          },
          price_discounted {
            code,
            fractional,
            formatted,
            presentational
          },
          modifier_group_ids,
          available
        }
      },
      restaurant {
        id,
        drn_id,
        menu_id,
        name,
        branch_type,
        accepts_allergy_notes,
        cash_tipping_message,
        address,
        fulfillment_type,
        currency_code,
        currency_symbol,
        menu_disabled,
        location {
          city_id,
          zone_id,
          address {
            address1,
            post_code,
            neighborhood,
            city,
            country,
            lat,
            lon
          }
        }
      },
      offer {
        offer {
          minimum_order_value {
            code,
            fractional,
            formatted,
            presentational
          }
          __typename
          ... on FullMenuPercentOffOffer {
            alcohol_allowed,
            max_discount {
              code,
              fractional,
              formatted,
              presentational
            },
            percentage_discount
          }
          ... on FreeItemOffer {
            alcohol_allowed,
            max_discount {
              code,
              fractional,
              formatted,
              presentational
            },
            item_ids
          }
          ... on FlashDealOffer {
            alcohol_allowed,
            max_discount {
              code,
              fractional,
              formatted,
              presentational
            },
            percentage_discount
          }
          ... on ItemSpecificPercentOffOffer {
            alcohol_allowed,
            max_discount {
              code,
              fractional,
              formatted,
              presentational
            },
            percentage_discount,
            item_ids
          },
          ... on FreeDeliveryOffer {
            alcohol_allowed
          }
        }
      }
    }
  }
}
                `,
                headers: {
                    "accept": "application/json",
                }
            }
        }
    ).json();
    return res;
}

export async function processSuccessulResult(result: SuccessfulResult, client: pg.PoolClient) {
    try {
        await client.query("BEGIN");
        const restaurant = result.restaurant;
        const location = restaurant.location != undefined ? restaurant.location :
            {
                city_id: null,
                zone_id: null,
                address: null
            };

        const address = location.address != undefined ? location.address :
            {
                address1: null,
                post_code: null,
                neighborhood: null,
                city: null,
                country: null,
                lat: null,
                lon: null
            };

        const insertRestaurantData = [
            restaurant.drn_id,
            restaurant.menu_id,
            restaurant.name,
            restaurant.branch_type,
            restaurant.accepts_allergy_notes,
            restaurant.cash_tipping_message,
            restaurant.fulfillment_type,
            restaurant.currency_code,
            restaurant.currency_symbol,
            restaurant.menu_disabled,
            location.city_id,
            location.zone_id,
            restaurant.address,
            address.address1,
            address.post_code,
            address.neighborhood,
            address.city,
            address.country,
            address.lat,
            address.lon,
            restaurant.id
        ];
        const insertRestaurantQuery = `
      UPDATE restaurants
      SET
        drn_id = $1,
        menu_id = $2,
        name = $3,
        branch_type = $4,
        accepts_allergy_notes = $5,
        cash_tipping_message = $6,
        fulfillment_type = $7,
        currency_code = $8,
        currency_symbol = $9,
        menu_disabled = $10,
        location__city_id = $11,
        location__zone_id = $12,
        address__formatted = $13,
        address__address1 = $14,
        address__postcode = $15,
        address__neighborhood = $16,
        address__city = $17,
        address__country = $18,
        address__lat = $19,
        address__lon = $20
      WHERE id = $21
    `;
        await client.query(insertRestaurantQuery, insertRestaurantData);

        const items = result.items.map(item => [
            item.id,
            restaurant.id,
            item.uid,
            item.category_id,
            item.name,
            item.description,
            item.product_information,
            item.price != undefined ? item.price.code : null,
            item.price != undefined ? item.price.fractional : null,
            item.price != undefined ? item.price.formatted : null,
            item.price != undefined ? item.price.presentational : null,
            item.price_discounted != undefined ? item.price_discounted.code : null,
            item.price_discounted != undefined ? item.price_discounted.fractional : null,
            item.price_discounted != undefined ? item.price_discounted.formatted : null,
            item.price_discounted != undefined ? item.price_discounted.presentational : null,
            item.percentage_discounted,
            item.calories,
            item.available,
            item.popular,
            item.alcohol,
            item.max_selection,
            item.is_signature_exclusive
        ]);

        const insertItemQuery = format!(
            `
      INSERT INTO items(
        id,
        restaurant_id,
        uid,
        category_id,
        name,
        description,
        product_information,
        price__code,
        price__fractional,
        price__formatted,
        price__presentational,
        price_discounted__code,
        price_discounted__fractional,
        price_discounted__formatted,
        price_discounted__presentational,
        percentage_discounted,
        calories,
        available,
        popular,
        alcohol,
        max_selection,
        is_signature_exclusive
      ) VALUES %L ON CONFLICT DO NOTHING
      `, items
        );

        if (items.length !== 0)
            await client.query(insertItemQuery);

        const categoriesData = result.categories.map(c => [
            c.id,
            restaurant.id,
            c.name,
            c.description
        ]);

        const categoriesQuery = format!(
            `
      INSERT INTO categories(
        id,
        restaurant_id,
        name,
        description
      ) VALUES %L ON CONFLICT DO NOTHING
      `, categoriesData
        );

        if (categoriesData.length !== 0)
            await client.query(categoriesQuery);

        const itemsToCategoriesData = result.categories.map(c => c.item_ids.map(
            i => [i, c.id, restaurant.id]
        )).flat();

        const itemsToCatgoriesQuery = format!(
            `INSERT INTO items_to_categories (
        item_id,
        category_id,
        restaurant_id
      ) VALUES %L ON CONFLICT DO NOTHING`,
            itemsToCategoriesData
        );

        if (itemsToCategoriesData.length !== 0)
            await client.query(itemsToCatgoriesQuery);

        const modifierGroupsData = result.modifier_groups.map(g => [
            g.id,
            restaurant.id,
            g.name,
            g.description,
            g.min_selection,
            g.max_selection,
            g.repeatable
        ]);

        const modifierGroupsQuery = format!(
            `INSERT INTO modifier_groups(
        id,
        restaurant_id,
        name,
        description,
        min_selection,
        max_selection,
        repeatable
      ) VALUES %L ON CONFLICT DO NOTHING`, modifierGroupsData
        );

        if (modifierGroupsData.length !== 0)
            await client.query(modifierGroupsQuery);

        const modifierOptionsData = result.modifier_groups.map(g =>
            g.modifier_options.map(o => [
                o.id,
                restaurant.id,
                g.id,
                o.uid,
                o.name,
                o.description,
                o.price ? o.price.code : null,
                o.price ? o.price.fractional : null,
                o.price ? o.price.formatted : null,
                o.price ? o.price.presentational : null,
                o.price_discounted ? o.price_discounted.code : null,
                o.price_discounted ? o.price_discounted.fractional : null,
                o.price_discounted ? o.price_discounted.formatted : null,
                o.price_discounted ? o.price_discounted.presentational : null,
                o.available
            ])
        ).flat();

        const modifierOptionsQuery = format!(
            `INSERT INTO modifier_options (
        id,
        restaurant_id,
        modifier_group_id,
        uid,
        name,
        description,
        price__code,
        price__fractional,
        price__formatted,
        price__presentational,
        price_discounted__code,
        price_discounted__fractional,
        price_discounted__formatted,
        price_discounted__presentational,
        available
      ) VALUES %L ON CONFLICT DO NOTHING`, modifierOptionsData
        );

        if (modifierOptionsData.length !== 0)
            await client.query(modifierOptionsQuery);

        const itemsToModifierGroupsData = result.items.map(i => {
            if (!i.modifier_group_ids)
                return [];
            else
                return i.modifier_group_ids.map(g => [
                    i.id,
                    g,
                    restaurant.id
                ])
        }).flat();

        const itemsToModifierGroupsQuery = format!(
            `INSERT INTO items_to_modifier_groups (
        item_id,
        modifier_group_id,
        restaurant_id
      ) VALUES %L ON CONFLICT DO NOTHING`, itemsToModifierGroupsData
        );

        if (itemsToModifierGroupsData.length !== 0)
            await client.query(itemsToModifierGroupsQuery);

        await client.query("UPDATE restaurants SET visited_time = $1 WHERE id = $2", [new Date(), restaurant.id]);
        await client.query("COMMIT");
    } catch (e) {
        await client.query("ROLLBACK");
        throw e;
    } finally {
        client.release()
    }
}

export async function getAndProcessRestaurant(restaurantId: string, client: pg.PoolClient) {
    const res = await getRestaurantW(restaurantId);
    if (res.data !== null) {
        await processSuccessulResult(res.data.get_menu_page.meta, client);
    }
}

const pool = new pg.Pool({
    user: "postgres",
    host: "localhost",
    password: "postgres",
    database: "postgres",
    port: 15434
});

const client = await pool.connect();
try {
    const res = await client.query("SELECT id FROM restaurants WHERE visited_time IS NULL");
    const promises = res.rows.map(async row => getAndProcessRestaurant(row.id, await pool.connect()));
    await Promise.all(promises);
} finally {
    client.release();
}

// Output types

interface Category {
    id: string;
    name: string;
    description?: string;
    item_ids: [string];
}

interface Currency {
    code: string;
    fractional: number;
    formatted: string;
    presentational?: string;
}

interface Item {
    id: string;
    uid: number;
    category_id: string;
    name: string;
    description?: string;
    product_information?: string;
    price?: Currency;
    price_discounted?: Currency;
    percentage_discounted?: string;
    calories?: string;
    available: boolean;
    popular: boolean;
    alcohol: boolean;
    modifier_group_ids: [string];
    max_selection?: number;
    is_signature_exclusive: boolean
}

interface ModifierGroup {
    id: string;
    name: string;
    description?: string;
    min_selection: number;
    max_selection: number;
    repeatable: boolean;
    modifier_options: [ModifierOptions];
}

interface ModifierOptions {
    id: string;
    uid: number;
    name: string;
    description?: string;
    price?: Currency;
    price_discounted?: Currency;
    modifier_group_ids?: [string];
    available: boolean;
}

interface Restaurant {
    id: string;
    drn_id: string;
    menu_id: string;
    name: string;
    branch_type: string;
    accepts_allergy_notes: boolean;
    cash_tipping_message?: string;
    address?: string,
    fulfillment_type: string;
    currency_code: string;
    currency_symbol: string;
    menu_disabled: boolean;
    location?: {
        city_id?: string;
        zone_id?: string;
        address?: {
            address1?: string;
            post_code?: string;
            neighborhood?: string;
            city?: string;
            country?: string;
            lat?: number;
            lon?: number;
        };
    };
}

interface SuccessfulResult {
    categories: [Category];
    items: [Item];
    modifier_groups: [ModifierGroup];
    restaurant: Restaurant;
    offer?: any;
}
