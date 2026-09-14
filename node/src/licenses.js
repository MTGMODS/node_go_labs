const express = require("express");
const { ValidationError, ConflictError } = require("./store");

function parseId(req, res) {
  const id = Number(req.params.id);
  if (!Number.isInteger(id) || id <= 0) {
    res.status(400).json({ error: "id must be a positive integer" });
    return null;
  }
  return id;
}

function sendError(res, err) {
  if (err instanceof ValidationError) {
    return res.status(400).json({ error: err.message });
  }
  if (err instanceof ConflictError) {
    return res.status(409).json({ error: err.message });
  }
  return res.status(500).json({ error: "internal server error" });
}

function createLicensesRouter(store) {
  const router = express.Router();

  router.get("/", (_req, res) => {
    res.status(200).json(store.list());
  });

  router.post("/", (req, res) => {
    try {
      const created = store.create(req.body);
      res.status(201).json(created);
    } catch (err) {
      sendError(res, err);
    }
  });

  router.get("/:id", (req, res) => {
    const id = parseId(req, res);
    if (id === null) {
      return;
    }
    const license = store.getById(id);
    if (!license) {
      return res.status(404).json({ error: "license not found" });
    }
    res.status(200).json(license);
  });

  router.put("/:id", (req, res) => {
    const id = parseId(req, res);
    if (id === null) {
      return;
    }
    try {
      const updated = store.update(id, req.body);
      if (!updated) {
        return res.status(404).json({ error: "license not found" });
      }
      res.status(200).json(updated);
    } catch (err) {
      sendError(res, err);
    }
  });

  router.delete("/:id", (req, res) => {
    const id = parseId(req, res);
    if (id === null) {
      return;
    }
    if (!store.remove(id)) {
      return res.status(404).json({ error: "license not found" });
    }
    res.status(204).send();
  });

  router.all(["/", "/:id"], (_req, res) => {
    res.status(405).json({ error: "method not allowed" });
  });

  return router;
}

module.exports = { createLicensesRouter };
