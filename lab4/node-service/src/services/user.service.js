const { domainError } = require("../errors");
const { User } = require("../models/user");

class UserService {
  constructor(repository) {
    this.repository = repository;
  }

  async listUsers() {
    return this.repository.list();
  }

  async getUser(id) {
    const user = await this.repository.findById(id);
    if (!user) {
      throw domainError("user_not_found", `User with id ${id} was not found`);
    }
    return user;
  }

  async createUser(input) {
    const normalized = this.normalize(input);
    if (await this.repository.findByEmail(normalized.email)) {
      throw domainError("email_conflict", `User with email ${normalized.email} already exists`);
    }
    return this.repository.create(new User(normalized));
  }

  async updateUser(id, input) {
    await this.getUser(id);
    const normalized = this.normalize(input);
    const emailOwner = await this.repository.findByEmail(normalized.email);
    if (emailOwner && emailOwner.id !== id) {
      throw domainError("email_conflict", `User with email ${normalized.email} already exists`);
    }
    return this.repository.update(new User({ id, ...normalized }));
  }

  async deleteUser(id) {
    await this.getUser(id);
    await this.repository.delete(id);
  }

  normalize(input) {
    return {
      name: input.name.trim(),
      email: input.email.trim().toLowerCase(),
    };
  }
}

module.exports = { UserService };
