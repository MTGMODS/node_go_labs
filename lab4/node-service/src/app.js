const express = require("express");
const { UserController } = require("./controllers/user.controller");
const { errorHandler, notFoundHandler } = require("./middleware/error-handler");
const { requestLogger } = require("./middleware/request-logger");
const { InMemoryUserRepository } = require("./repositories/user.repository");
const { createHealthRouter } = require("./routes/health.routes");
const { createUserRouter } = require("./routes/user.routes");
const { UserService } = require("./services/user.service");

function createApp(repository = new InMemoryUserRepository()) {
  const service = new UserService(repository);
  const controller = new UserController(service);
  const app = express();

  app.use(requestLogger);
  app.use(express.json({ limit: "32kb" }));
  app.use("/health", createHealthRouter());
  app.use("/api/users", createUserRouter(controller));
  app.use(notFoundHandler);
  app.use(errorHandler);
  return app;
}

module.exports = { createApp };
