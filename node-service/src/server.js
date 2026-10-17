const http = require("node:http");
const { createApp } = require("./app");
const { StreamProcessingService } = require("./services/stream.service");
const { DatabaseExportService } = require("./services/database-export.service");
const { AccountService } = require("./services/account.service");
const { WebSocketHub } = require("./websocket/websocket-hub");
const { createPostgresPool, verifyPostgres } = require("./db/postgres");
const { connectMongo } = require("./db/mongo");
const { MongoEventLogger } = require("./events/mongo-event-logger");
const { createLicenseRepository } = require("./repositories/license-repository.factory");

function createHttpServer({ repository, dataDir, websocketOptions, eventLogger, accountService } = {}) {
  const hub = new WebSocketHub(websocketOptions);
  const streamService = new StreamProcessingService({
    dataDir,
    onEvent: (event) => hub.broadcast(event),
  });
  const publishEvent = async (event) => {
    if (eventLogger) await eventLogger.log(event);
    hub.broadcast(event);
  };
  const exportService = repository && typeof repository.iterate === "function"
    ? new DatabaseExportService(repository, { dataDir, publish: (event) => hub.broadcast(event) })
    : null;
  const app = createApp(repository, {
    hub, streamService, publishEvent, eventLogger, exportService, accountService,
  });
  const server = http.createServer(app);
  hub.attach(server);
  return { app, server, hub, streamService };
}

async function createProductionServer() {
  const pool = createPostgresPool();
  await verifyPostgres(pool);
  const { client: mongoClient, db: mongoDb } = await connectMongo();
  const repository = await createLicenseRepository(process.env.REPO_TYPE, { pool, mongoDb });
  const eventLogger = new MongoEventLogger(mongoDb);
  const accountService = new AccountService(pool);
  const runtime = createHttpServer({ repository, eventLogger, accountService });
  return { ...runtime, pool, mongoClient };
}

if (require.main === module) {
  const port = Number(process.env.PORT || 3006);
  createProductionServer().then(({ server, pool, mongoClient, hub }) => {
    server.listen(port, "0.0.0.0", () => {
      console.log(`lab6 license persistence service (node, ${process.env.REPO_TYPE}) listening on ${port}`);
    });
    const shutdown = () => server.close(async () => {
      hub.close();
      await Promise.allSettled([pool.end(), mongoClient.close()]);
      process.exit(0);
    });
    process.on("SIGTERM", shutdown);
    process.on("SIGINT", shutdown);
  }).catch((error) => {
    console.error("Failed to start Node service", error);
    process.exit(1);
  });
}

module.exports = { createHttpServer, createProductionServer };
