const express = require("express");
const { LicenseController } = require("./controllers/license.controller");
const { errorHandler, notFoundHandler } = require("./middleware/error-handler");
const { requestLogger } = require("./middleware/request-logger");
const { InMemoryLicenseRepository } = require("./repositories/license.repository");
const { createHealthRouter } = require("./routes/health.routes");
const { createLicenseRouter } = require("./routes/license.routes");
const { LicenseService } = require("./services/license.service");

function createApp(repository = new InMemoryLicenseRepository()) {
  const service = new LicenseService(repository);
  const controller = new LicenseController(service);
  const app = express();

  app.use(requestLogger);
  app.use(express.json({ limit: "32kb" }));
  app.use("/health", createHealthRouter());
  app.use("/licenses", createLicenseRouter(controller));
  app.use(notFoundHandler);
  app.use(errorHandler);
  return app;
}

module.exports = { createApp };
