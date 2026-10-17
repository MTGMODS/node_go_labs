function health(_req, res) {
  res.status(200).json({
    status: "UP",
      service: "license-persistence-service",
    runtime: "node",
      repository: process.env.REPO_TYPE || "in-memory",
      capabilities: ["licenses", "postgres", "mongo", "transactions", "streams", "websocket"],
  });
}

module.exports = { health };
