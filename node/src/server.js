const express = require("express");

const { Store } = require("./store");
const { createLicensesRouter } = require("./licenses");

const app = express();
const store = new Store();
const port = Number(process.env.PORT) || 3000;

app.use(express.json());

app.get("/health", (_req, res) => {
  res.status(200).json({
    status: "UP",
    service: "license-service",
    runtime: "node",
  });
});

app.all("/health", (_req, res) => {
  res.status(405).json({ error: "method not allowed" });
});

app.use("/licenses", createLicensesRouter(store));

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
