CREATE TABLE IF NOT EXISTS licenses (
    id            SERIAL PRIMARY KEY,
    license_key   VARCHAR(128) NOT NULL UNIQUE,
    product       VARCHAR(100) NOT NULL,
    owner         VARCHAR(255) NOT NULL,
    status        VARCHAR(32) NOT NULL,
    duration_days INTEGER NOT NULL CHECK (duration_days > 0),
    max_devices   INTEGER NOT NULL CHECK (max_devices > 0),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS accounts (
    id      SERIAL PRIMARY KEY,
    owner   VARCHAR(100) NOT NULL,
    balance NUMERIC(12,2) NOT NULL DEFAULT 0.00 CHECK (balance >= 0)
);

INSERT INTO accounts (owner, balance)
SELECT 'Primary license account', 1000.00
WHERE NOT EXISTS (SELECT 1 FROM accounts WHERE id = 1);

INSERT INTO accounts (owner, balance)
SELECT 'Secondary license account', 250.00
WHERE NOT EXISTS (SELECT 1 FROM accounts WHERE id = 2);
