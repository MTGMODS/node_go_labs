class InMemoryUserRepository {
  constructor() {
    this.users = new Map();
    this.nextId = 1;
  }

  async list() {
    return [...this.users.values()].map((user) => ({ ...user }));
  }

  async findById(id) {
    const user = this.users.get(id);
    return user ? { ...user } : null;
  }

  async findByEmail(email) {
    const normalized = email.toLowerCase();
    const user = [...this.users.values()].find((item) => item.email === normalized);
    return user ? { ...user } : null;
  }

  async create(user) {
    const stored = { ...user, id: this.nextId };
    this.nextId += 1;
    this.users.set(stored.id, stored);
    return { ...stored };
  }

  async update(user) {
    if (!this.users.has(user.id)) return null;
    const stored = { ...user };
    this.users.set(stored.id, stored);
    return { ...stored };
  }

  async delete(id) {
    return this.users.delete(id);
  }
}

module.exports = { InMemoryUserRepository };
