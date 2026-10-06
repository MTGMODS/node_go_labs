class InMemoryLicenseRepository {
  constructor() {
    this.licenses = new Map();
    this.nextId = 1;
  }

  async list() {
    return [...this.licenses.values()].map((license) => ({ ...license }));
  }

  async findById(id) {
    const license = this.licenses.get(id);
    return license ? { ...license } : null;
  }

  async findByKey(key) {
    const license = [...this.licenses.values()].find((item) => item.key === key);
    return license ? { ...license } : null;
  }

  async create(license) {
    const stored = { ...license, id: this.nextId };
    this.nextId += 1;
    this.licenses.set(stored.id, stored);
    return { ...stored };
  }

  async update(license) {
    if (!this.licenses.has(license.id)) return null;
    const stored = { ...license };
    this.licenses.set(stored.id, stored);
    return { ...stored };
  }

  async delete(id) {
    return this.licenses.delete(id);
  }
}

module.exports = { InMemoryLicenseRepository };
