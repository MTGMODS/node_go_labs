const express = require("express");
const { asyncHandler } = require("../middleware/async-handler");

function createAccountRouter(controller) {
  const router = express.Router();
  router.get("/", asyncHandler(controller.list));
  router.post("/transfer", asyncHandler(controller.transfer));
  return router;
}

module.exports = { createAccountRouter };
