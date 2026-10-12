const express = require("express");
const { asyncHandler } = require("../middleware/async-handler");

function createStreamRouter(controller) {
  const router = express.Router();
  router.get("/source", asyncHandler(controller.sourceInfo));
  router.post("/source", asyncHandler(controller.generateSource));
  router.post("/process", asyncHandler(controller.start));
  router.get("/jobs", asyncHandler(controller.listJobs));
  router.get("/jobs/:id", asyncHandler(controller.getJob));
  return router;
}

module.exports = { createStreamRouter };
