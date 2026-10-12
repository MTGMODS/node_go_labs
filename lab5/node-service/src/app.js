const express = require("express");
const { LicenseController } = require("./controllers/license.controller");
const { RealtimeController } = require("./controllers/realtime.controller");
const { StreamController } = require("./controllers/stream.controller");
const { errorHandler, notFoundHandler } = require("./middleware/error-handler");
const { requestLogger } = require("./middleware/request-logger");
const { InMemoryLicenseRepository } = require("./repositories/license.repository");
const { createHealthRouter } = require("./routes/health.routes");
const { createLicenseRouter } = require("./routes/license.routes");
const { createRealtimeRouter } = require("./routes/realtime.routes");
const { createStreamRouter } = require("./routes/stream.routes");
const { LicenseService } = require("./services/license.service");

function createApp(repository = new InMemoryLicenseRepository(), { hub, streamService } = {}) {
  const service = new LicenseService(repository);
  const controller = new LicenseController(service);
  const streamController = streamService ? new StreamController(streamService) : null;
  const realtimeController = hub ? new RealtimeController(hub) : null;
  const app = express();

  app.use(requestLogger);
  app.use(express.json({ limit: "32kb" }));
  app.use("/health", createHealthRouter());
  app.use("/licenses", createLicenseRouter(controller));
  if (streamController) app.use("/streams", createStreamRouter(streamController));
  if (realtimeController) app.use("/realtime", createRealtimeRouter(realtimeController));
  app.use(express.static("public"));
  app.use(notFoundHandler);
  app.use(errorHandler);
  return app;
}

module.exports = { createApp };
