const { Pool } = require("pg");

function createPostgresPool() {
  return new Pool({
    host: process.env.PG_HOST,
    port: Number(process.env.PG_PORT),
    database: process.env.PG_DB,
    user: process.env.PG_USER,
    password: process.env.PG_PASSWORD,
    max: Number(process.env.PG_POOL_MAX || 15),
    idleTimeoutMillis: 30_000,
    connectionTimeoutMillis: 5_000,
  });
}

async function verifyPostgres(pool) {
  await pool.query("SELECT 1");
}

module.exports = { createPostgresPool, verifyPostgres };
