const { repositoryError } = require("../errors");

function mapRow(row) {
  if (!row) return null;
  return {
    id: row.id,
    key: row.license_key,
    product: row.product,
    owner: row.owner,
    status: row.status,
    duration_days: row.duration_days,
    max_devices: row.max_devices,
  };
}

class PostgresLicenseRepository {
  constructor(pool) {
    this.pool = pool;
  }

  async list() {
    const result = await this.pool.query("SELECT * FROM licenses ORDER BY id");
    return result.rows.map(mapRow);
  }

  async findById(id) {
    const result = await this.pool.query("SELECT * FROM licenses WHERE id = $1", [id]);
    return mapRow(result.rows[0]);
  }

  async findByKey(key) {
    const result = await this.pool.query("SELECT * FROM licenses WHERE license_key = $1", [key]);
    return mapRow(result.rows[0]);
  }

  async create(license) {
    try {
      const result = await this.pool.query(
        `INSERT INTO licenses (license_key, product, owner, status, duration_days, max_devices)
         VALUES ($1, $2, $3, $4, $5, $6) RETURNING *`,
        [license.key, license.product, license.owner, license.status, license.duration_days, license.max_devices],
      );
      return mapRow(result.rows[0]);
    } catch (error) {
      if (error.code === "23505") throw repositoryError("duplicate_key", "License key already exists");
      throw error;
    }
  }

  async update(license) {
    try {
      const result = await this.pool.query(
        `UPDATE licenses SET license_key=$2, product=$3, owner=$4, status=$5,
         duration_days=$6, max_devices=$7, updated_at=NOW() WHERE id=$1 RETURNING *`,
        [license.id, license.key, license.product, license.owner, license.status, license.duration_days, license.max_devices],
      );
      return mapRow(result.rows[0]);
    } catch (error) {
      if (error.code === "23505") throw repositoryError("duplicate_key", "License key already exists");
      throw error;
    }
  }

  async delete(id) {
    const result = await this.pool.query("DELETE FROM licenses WHERE id = $1", [id]);
    return result.rowCount > 0;
  }

  async count() {
    const result = await this.pool.query("SELECT COUNT(*)::int AS count FROM licenses");
    return result.rows[0].count;
  }

  async *iterate(batchSize = 100) {
    let afterId = 0;
    while (true) {
      const result = await this.pool.query(
        "SELECT * FROM licenses WHERE id > $1 ORDER BY id LIMIT $2",
        [afterId, batchSize],
      );
      if (result.rows.length === 0) return;
      for (const row of result.rows) {
        const license = mapRow(row);
        afterId = license.id;
        yield license;
      }
    }
  }
}

module.exports = { PostgresLicenseRepository };
