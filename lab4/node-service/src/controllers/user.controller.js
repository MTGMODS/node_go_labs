const { badRequest } = require("../errors");

function parseId(raw) {
  if (!/^\d+$/.test(raw)) {
    throw badRequest("invalid_identifier", "User id must be a positive integer");
  }
  const id = Number(raw);
  if (!Number.isSafeInteger(id) || id <= 0) {
    throw badRequest("invalid_identifier", "User id must be a positive integer");
  }
  return id;
}

class UserController {
  constructor(service) {
    this.service = service;
  }

  list = async (_req, res) => {
    res.status(200).json(await this.service.listUsers());
  };

  get = async (req, res) => {
    res.status(200).json(await this.service.getUser(parseId(req.params.id)));
  };

  create = async (req, res) => {
    res.status(201).json(await this.service.createUser(req.body));
  };

  update = async (req, res) => {
    res.status(200).json(await this.service.updateUser(parseId(req.params.id), req.body));
  };

  delete = async (req, res) => {
    await this.service.deleteUser(parseId(req.params.id));
    res.status(204).end();
  };
}

module.exports = { UserController, parseId };
