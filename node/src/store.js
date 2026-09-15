const { Pool } = require("pg");

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

const SELECT_COLUMNS = "id, key, product, owner, status, duration_days, max_devices";

function mapRow(row) {
  return {
    id: row.id,
    key: row.key,
    product: row.product,
    owner: row.owner,
    status: row.status,
    duration_days: row.duration_days,
    max_devices: row.max_devices,
  };
}

class Store {
  constructor(databaseUrl) {
    this.pool = new Pool({ connectionString: databaseUrl });
  }

  async ping() {
    await this.pool.query("SELECT 1");
  }

  async list() {
    const { rows } = await this.pool.query(
      `SELECT ${SELECT_COLUMNS} FROM licenses ORDER BY id`,
    );
    return rows.map(mapRow);
  }

  async getById(id) {
    const { rows } = await this.pool.query(
      `SELECT ${SELECT_COLUMNS} FROM licenses WHERE id = $1`,
      [id],
    );
    return rows[0] ? mapRow(rows[0]) : null;
  }

  async create(body) {
    const data = this.normalize(body, null);
    try {
      const { rows } = await this.pool.query(
        `INSERT INTO licenses (key, product, owner, status, duration_days, max_devices)
         VALUES ($1, $2, $3, $4, $5, $6)
         RETURNING ${SELECT_COLUMNS}`,
        [data.key, data.product, data.owner, data.status, data.duration_days, data.max_devices],
      );
      return mapRow(rows[0]);
    } catch (err) {
      this.rethrow(err);
    }
  }

  async update(id, body) {
    const existing = await this.getById(id);
    if (!existing) {
      return null;
    }
    const data = this.normalize(body, existing);
    try {
      const { rows } = await this.pool.query(
        `UPDATE licenses
         SET key = $1, product = $2, owner = $3, status = $4, duration_days = $5, max_devices = $6
         WHERE id = $7
         RETURNING ${SELECT_COLUMNS}`,
        [data.key, data.product, data.owner, data.status, data.duration_days, data.max_devices, id],
      );
      return rows[0] ? mapRow(rows[0]) : null;
    } catch (err) {
      this.rethrow(err);
    }
  }

  async remove(id) {
    const result = await this.pool.query("DELETE FROM licenses WHERE id = $1", [id]);
    return result.rowCount > 0;
  }

  rethrow(err) {
    if (err && err.code === "23505") {
      throw new ConflictError("license key already exists");
    }
    throw err;
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
