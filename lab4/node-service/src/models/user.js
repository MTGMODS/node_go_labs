class User {
  constructor({ id = 0, name, email }) {
    this.id = id;
    this.name = name;
    this.email = email;
  }
}

module.exports = { User };
