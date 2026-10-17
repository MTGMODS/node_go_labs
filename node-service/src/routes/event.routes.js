const express = require("express");
const { asyncHandler } = require("../middleware/async-handler");

function createEventRouter(controller) {
  const router = express.Router();
  router.get("/", asyncHandler(controller.list));
  return router;
}

module.exports = { createEventRouter };
