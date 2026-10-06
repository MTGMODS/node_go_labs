const { domainError } = require("../errors");
const { License } = require("../models/license");

class LicenseService {
  constructor(repository) {
    this.repository = repository;
  }

  async listLicenses() {
    return this.repository.list();
  }

  async getLicense(id) {
    const license = await this.repository.findById(id);
    if (!license) {
      throw domainError("license_not_found", `License with id ${id} was not found`);
    }
    return license;
  }

  async createLicense(input) {
    const data = this.applyInput(input, {
      status: "NOT_ACTIVATED",
      duration_days: 30,
      max_devices: 1,
    });
    if (await this.repository.findByKey(data.key)) {
      throw domainError("license_key_conflict", `License with key ${data.key} already exists`);
    }
    return this.repository.create(new License(data));
  }

  async updateLicense(id, input) {
    const existing = await this.getLicense(id);
    const data = this.applyInput(input, existing);
    const keyOwner = await this.repository.findByKey(data.key);
    if (keyOwner && keyOwner.id !== id) {
      throw domainError("license_key_conflict", `License with key ${data.key} already exists`);
    }
    return this.repository.update(new License({ id, ...data }));
  }

  async deleteLicense(id) {
    await this.getLicense(id);
    await this.repository.delete(id);
  }

  applyInput(input, base) {
    return {
      key: input.key === undefined ? base.key : input.key.trim(),
      product: input.product === undefined ? base.product : input.product.trim(),
      owner: input.owner === undefined ? base.owner : input.owner.trim(),
      status: input.status === undefined ? base.status : input.status,
      duration_days: input.duration_days === undefined ? base.duration_days : input.duration_days,
      max_devices: input.max_devices === undefined ? base.max_devices : input.max_devices,
    };
  }
}

module.exports = { LicenseService };
