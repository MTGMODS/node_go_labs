const { MongoLicenseRepository } = require("./mongo-license.repository");
const { PostgresLicenseRepository } = require("./postgres-license.repository");

async function createLicenseRepository(type, { pool, mongoDb }) {
  if (type === "postgres") return new PostgresLicenseRepository(pool);
  if (type === "mongo") {
    const repository = new MongoLicenseRepository(mongoDb);
    await repository.initialize();
    return repository;
  }
  throw new Error(`Unsupported REPO_TYPE: ${type}. Expected postgres or mongo`);
}

module.exports = { createLicenseRepository };
