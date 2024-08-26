-- backend/db/init/init.sql

CREATE TABLE IF NOT EXISTS items (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    price DOUBLE PRECISION NOT NULL
);

-- Optionally, insert some seed data
INSERT INTO items (name, price) VALUES ('Sample Item 1', 10.0);
INSERT INTO items (name, price) VALUES ('Sample Item 2', 20.0);
