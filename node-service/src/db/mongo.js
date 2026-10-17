const { MongoClient } = require("mongodb");

async function connectMongo() {
  const client = new MongoClient(process.env.MONGO_URI, {
    maxPoolSize: Number(process.env.MONGO_POOL_MAX || 15),
  });
  await client.connect();
  const db = client.db(process.env.MONGO_DB);
  await db.command({ ping: 1 });
  return { client, db };
}

module.exports = { connectMongo };
