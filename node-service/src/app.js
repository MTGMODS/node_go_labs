const express = require("express");
const { LicenseController } = require("./controllers/license.controller");
const { RealtimeController } = require("./controllers/realtime.controller");
const { StreamController } = require("./controllers/stream.controller");
const { AccountController } = require("./controllers/account.controller");
const { EventController } = require("./controllers/event.controller");
const { ExportController } = require("./controllers/export.controller");
const { errorHandler, notFoundHandler } = require("./middleware/error-handler");
const { requestLogger } = require("./middleware/request-logger");
const { InMemoryLicenseRepository } = require("./repositories/license.repository");
const { createHealthRouter } = require("./routes/health.routes");
const { createLicenseRouter } = require("./routes/license.routes");
const { createRealtimeRouter } = require("./routes/realtime.routes");
const { createStreamRouter } = require("./routes/stream.routes");
const { createAccountRouter } = require("./routes/account.routes");
const { createEventRouter } = require("./routes/event.routes");
const { createExportRouter } = require("./routes/export.routes");
const { LicenseService } = require("./services/license.service");

function createApp(repository = new InMemoryLicenseRepository(), {
  hub, streamService, publishEvent, eventLogger, exportService, accountService,
} = {}) {
  const service = new LicenseService(repository, publishEvent);
  const controller = new LicenseController(service);
  const streamController = streamService ? new StreamController(streamService) : null;
  const realtimeController = hub ? new RealtimeController(hub) : null;
  const eventController = eventLogger ? new EventController(eventLogger) : null;
  const exportController = exportService ? new ExportController(exportService) : null;
  const accountController = accountService ? new AccountController(accountService) : null;
  const app = express();

  app.use(requestLogger);
  app.use(express.json({ limit: "32kb" }));
  app.use("/health", createHealthRouter());
  if (exportController) app.use("/api/licenses/export", createExportRouter(exportController));
  app.use("/licenses", createLicenseRouter(controller));
  app.use("/api/licenses", createLicenseRouter(controller));
  if (eventController) app.use("/api/events", createEventRouter(eventController));
  if (accountController) app.use("/api/accounts", createAccountRouter(accountController));
  if (streamController) app.use("/streams", createStreamRouter(streamController));
  if (realtimeController) app.use("/realtime", createRealtimeRouter(realtimeController));
  app.use(express.static("public"));
  app.use(notFoundHandler);
  app.use(errorHandler);
  return app;
}

module.exports = { createApp };
