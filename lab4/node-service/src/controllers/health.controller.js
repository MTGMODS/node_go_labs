function health(_req, res) {
  res.status(200).json({ status: "UP", runtime: "node" });
}

module.exports = { health };
