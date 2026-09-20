const express = require("express");
const { Store } = require("./store");
const { createLicensesRouter } = require("./licenses");
const { createWorkloadsRouter } = require("./workloads");

const port = Number(process.env.PORT) || 3000;
const databaseUrl = process.env.DATABASE_URL;

if (!databaseUrl) {
  console.error("DATABASE_URL is required");
  process.exit(1);
}

async function main() {
  const store = new Store(databaseUrl);
  await store.ping();

  const app = express();
  app.use((req, res, next) => {
    res.setHeader("Access-Control-Allow-Origin", "*");
    res.setHeader("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS");
    res.setHeader("Access-Control-Allow-Headers", "Content-Type");
    if (req.method === "OPTIONS") {
      return res.status(204).end();
    }
    return next();
  });
  app.use(express.json());

  app.get("/health", async (_req, res) => {
    try {
      await store.ping();
      res.status(200).json({
        status: "UP",
        service: "license-service",
        runtime: "node",
        database: "UP",
      });
    } catch (_err) {
      res.status(503).json({
        status: "DOWN",
        service: "license-service",
        runtime: "node",
        database: "DOWN",
      });
    }
  });

  app.all("/health", (_req, res) => {
    res.status(405).json({ error: "method not allowed" });
  });

  app.use("/licenses", createLicensesRouter(store));
  app.use(createWorkloadsRouter());

  app.use((err, _req, res, next) => {
    if (err instanceof SyntaxError && err.status === 400 && "body" in err) {
      return res.status(400).json({ error: "invalid JSON" });
    }
    return next(err);
  });

  app.use((_req, res) => {
    res.status(404).json({ error: "not found" });
  });

  app.listen(port, "0.0.0.0", () => {
    console.log(`license-service (node) listening on ${port}`);
  });
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
