const { DomainError, HttpError } = require("../errors");

function notFoundHandler(req, _res, next) {
  next(new HttpError(404, "route_not_found", `Route ${req.method} ${req.originalUrl} was not found`));
}

function errorHandler(error, _req, res, _next) {
  if (error instanceof SyntaxError && error.status === 400 && "body" in error) {
    return res.status(400).json({ error: "invalid_json", message: "Request body contains invalid JSON" });
  }
  if (error instanceof HttpError) {
    return res.status(error.status).json({ error: error.code, message: error.message });
  }
  if (error instanceof DomainError) {
    const statusByCode = { license_not_found: 404, license_key_conflict: 409 };
    return res.status(statusByCode[error.code] ?? 400).json({ error: error.code, message: error.message });
  }
  console.error(error);
  return res.status(500).json({ error: "internal_error", message: "Internal server error" });
}

module.exports = { errorHandler, notFoundHandler };
