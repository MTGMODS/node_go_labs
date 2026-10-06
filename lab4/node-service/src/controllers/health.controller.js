function health(_req, res) {
  res.status(200).json({ status: "UP", service: "license-service", runtime: "node" });
}

module.exports = { health };
