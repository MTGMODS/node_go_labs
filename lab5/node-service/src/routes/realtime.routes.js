const express = require("express");

function createRealtimeRouter(controller) {
  const router = express.Router();
  router.get("/clients", controller.stats);
  return router;
}

module.exports = { createRealtimeRouter };
