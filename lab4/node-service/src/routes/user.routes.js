const express = require("express");
const { asyncHandler } = require("../middleware/async-handler");
const { validateUser } = require("../middleware/validate-user");

function createUserRouter(controller) {
  const router = express.Router();
  router.get("/", asyncHandler(controller.list));
  router.get("/:id", asyncHandler(controller.get));
  router.post("/", validateUser, asyncHandler(controller.create));
  router.put("/:id", validateUser, asyncHandler(controller.update));
  router.delete("/:id", asyncHandler(controller.delete));
  return router;
}

module.exports = { createUserRouter };
