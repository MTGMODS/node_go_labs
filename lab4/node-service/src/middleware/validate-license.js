const { badRequest } = require("../errors");

const allowedFields = new Set(["key", "product", "owner", "status", "duration_days", "max_devices"]);
const allowedStatuses = new Set(["NOT_ACTIVATED", "ACTIVE", "EXPIRED", "BANNED"]);

function validateLicense(req, _res, next) {
  const body = req.body;
  if (!body || typeof body !== "object" || Array.isArray(body)) {
    return next(badRequest("validation_error", "Request body must be a JSON object"));
  }

  const fields = Object.keys(body);
  if (fields.some((field) => !allowedFields.has(field))) {
    return next(badRequest("validation_error", "Request body contains unknown fields"));
  }
  if (req.method === "PUT" && fields.length === 0) {
    return next(badRequest("validation_error", "At least one field is required"));
  }
  if (req.method === "POST") {
    for (const field of ["key", "product", "owner"]) {
      if (!Object.hasOwn(body, field)) {
        return next(badRequest("validation_error", `${field} is required`));
      }
    }
  }

  for (const field of ["key", "product", "owner"]) {
    if (Object.hasOwn(body, field) && (typeof body[field] !== "string" || !body[field].trim())) {
      return next(badRequest("validation_error", `${field} must be a non-empty string`));
    }
  }
  if (Object.hasOwn(body, "status") &&
      (typeof body.status !== "string" || !allowedStatuses.has(body.status))) {
    return next(badRequest(
      "validation_error",
      "status must be one of: NOT_ACTIVATED, ACTIVE, EXPIRED, BANNED",
    ));
  }
  for (const field of ["duration_days", "max_devices"]) {
    if (Object.hasOwn(body, field) && (!Number.isInteger(body[field]) || body[field] < 1)) {
      return next(badRequest("validation_error", `${field} must be an integer >= 1`));
    }
  }
  return next();
}

module.exports = { validateLicense };
