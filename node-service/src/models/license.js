class License {
  constructor({ id, key, product, owner, status, duration_days, max_devices }) {
    if (id !== undefined) this.id = id;
    this.key = key;
    this.product = product;
    this.owner = owner;
    this.status = status;
    this.duration_days = duration_days;
    this.max_devices = max_devices;
  }
}

module.exports = { License };
