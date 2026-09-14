const ALLOWED_STATUSES = new Set([
  "NOT_ACTIVATED",
  "ACTIVE",
  "EXPIRED",
  "BANNED",
]);

class ValidationError extends Error {
  constructor(message) {
    super(message);
    this.name = "ValidationError";
  }
}

class ConflictError extends Error {
  constructor(message) {
    super(message);
    this.name = "ConflictError";
  }
}

class Store {
  constructor() {
    // Map ≈ dict у Python. Node однопотоковий, lock не потрібен.
    this.licenses = new Map();
    this.nextId = 1;
    this.seed();
  }

  seed() {
    this.create({
      key: "MTGM-VIP1-AAAA-0001",
      product: "MTG MODS VIP",
      owner: "bogdan",
      status: "ACTIVE",
      duration_days: 30,
      max_devices: 2,
    });
  }

  list() {
    return Array.from(this.licenses.values());
  }

  getById(id) {
    return this.licenses.get(id) || null;
  }

  create(body) {
    const data = this.normalize(body, null);
    this.assertUniqueKey(data.key, null);
    const license = { id: this.nextId++, ...data };
    this.licenses.set(license.id, license);
    return license;
  }

  update(id, body) {
    const existing = this.licenses.get(id);
    if (!existing) {
      return null;
    }
    const data = this.normalize(body, existing);
    this.assertUniqueKey(data.key, id);
    const license = { id, ...data };
    this.licenses.set(id, license);
    return license;
  }

  remove(id) {
    return this.licenses.delete(id);
  }

  assertUniqueKey(key, currentId) {
    for (const license of this.licenses.values()) {
      if (license.key === key && license.id !== currentId) {
        throw new ConflictError("license key already exists");
      }
    }
  }

  normalize(body, existing) {
    if (!body || typeof body !== "object" || Array.isArray(body)) {
      throw new ValidationError("body must be a JSON object");
    }

    const key = this.readString(body, "key", existing ? existing.key : undefined, true);
    const product = this.readString(body, "product", existing ? existing.product : undefined, true);
    const owner = this.readString(body, "owner", existing ? existing.owner : undefined, true);
    const status = this.readStatus(body, existing ? existing.status : "NOT_ACTIVATED");
    const durationDays = this.readInt(body, "duration_days", existing ? existing.duration_days : 30, 1);
    const maxDevices = this.readInt(body, "max_devices", existing ? existing.max_devices : 1, 1);

    return {
      key,
      product,
      owner,
      status,
      duration_days: durationDays,
      max_devices: maxDevices,
    };
  }

  readString(body, field, fallback, required) {
    if (!Object.hasOwn(body, field)) {
      if (required && fallback === undefined) {
        throw new ValidationError(`${field} is required`);
      }
      return fallback;
    }
    if (typeof body[field] !== "string") {
      throw new ValidationError(`${field} must be a string`);
    }
    const value = body[field].trim();
    if (!value) {
      throw new ValidationError(`${field} must not be empty`);
    }
    return value;
  }

  readStatus(body, fallback) {
    if (!Object.hasOwn(body, "status")) {
      return fallback;
    }
    if (typeof body.status !== "string" || !ALLOWED_STATUSES.has(body.status)) {
      throw new ValidationError(
        "status must be one of: NOT_ACTIVATED, ACTIVE, EXPIRED, BANNED",
      );
    }
    return body.status;
  }

  readInt(body, field, fallback, min) {
    if (!Object.hasOwn(body, field)) {
      return fallback;
    }
    const value = body[field];
    if (!Number.isInteger(value) || value < min) {
      throw new ValidationError(`${field} must be an integer >= ${min}`);
    }
    return value;
  }
}

module.exports = {
  Store,
  ValidationError,
  ConflictError,
};
