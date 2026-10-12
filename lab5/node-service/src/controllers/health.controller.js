function health(_req, res) {
  res.status(200).json({
    status: "UP",
    service: "license-stream-service",
    runtime: "node",
    capabilities: ["licenses", "streams", "websocket"],
  });
}

module.exports = { health };
