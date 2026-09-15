CREATE TABLE IF NOT EXISTS licenses (
    id SERIAL PRIMARY KEY,
    key TEXT NOT NULL UNIQUE,
    product TEXT NOT NULL,
    owner TEXT NOT NULL,
    status TEXT NOT NULL,
    duration_days INTEGER NOT NULL,
    max_devices INTEGER NOT NULL
);

INSERT INTO licenses (key, product, owner, status, duration_days, max_devices)
VALUES ('MTGM-VIP1-AAAA-0001', 'MTG MODS VIP', 'bogdan', 'ACTIVE', 30, 2)
ON CONFLICT (key) DO NOTHING;
