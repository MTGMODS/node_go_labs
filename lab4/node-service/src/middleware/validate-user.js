const { badRequest } = require("../errors");

const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

function validateUser(req, _res, next) {
  const { name, email } = req.body ?? {};
  if (typeof name !== "string" || typeof email !== "string") {
    return next(badRequest("validation_error", "name and email must be strings"));
  }
  if (!name.trim() || !email.trim()) {
    return next(badRequest("validation_error", "name and email are required"));
  }
  if (!emailPattern.test(email.trim())) {
    return next(badRequest("validation_error", "email must be valid"));
  }
  return next();
}

module.exports = { validateUser };
