const express = require("express");
const { asyncHandler } = require("../middleware/async-handler");
const { validateLicense } = require("../middleware/validate-license");

function createLicenseRouter(controller) {
  const router = express.Router();
  router.get("/", asyncHandler(controller.list));
  router.get("/:id", asyncHandler(controller.get));
  router.post("/", validateLicense, asyncHandler(controller.create));
  router.put("/:id", validateLicense, asyncHandler(controller.update));
  router.delete("/:id", asyncHandler(controller.delete));
  return router;
}

module.exports = { createLicenseRouter };
