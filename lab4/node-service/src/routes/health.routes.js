const express = require("express");
const { health } = require("../controllers/health.controller");

function createHealthRouter() {
  const router = express.Router();
  router.get("/", health);
  return router;
}

module.exports = { createHealthRouter };
