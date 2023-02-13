FROM docker.io/postgres:15.1-alpine

COPY copy-restaurants.sql /copy-restaurants.sql
COPY create-tables.sql /docker-entrypoint-initdb.d/
COPY load-locations.sql /docker-entrypoint-initdb.d/
COPY locations.csv /locations.csv