const express = require("express");
const { asyncHandler } = require("../middleware/async-handler");

function createExportRouter(controller) {
  const router = express.Router();
  router.get("/", asyncHandler(controller.export));
  return router;
}

module.exports = { createExportRouter };
