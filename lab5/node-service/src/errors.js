class DomainError extends Error {
  constructor(code, message) {
    super(message);
    this.name = "DomainError";
    this.code = code;
  }
}

class HttpError extends Error {
  constructor(status, code, message) {
    super(message);
    this.name = "HttpError";
    this.status = status;
    this.code = code;
  }
}

const badRequest = (code, message) => new HttpError(400, code, message);
const domainError = (code, message) => new DomainError(code, message);

module.exports = { DomainError, HttpError, badRequest, domainError };
