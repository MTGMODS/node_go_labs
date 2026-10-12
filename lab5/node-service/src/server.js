const http = require("node:http");
const { createApp } = require("./app");
const { StreamProcessingService } = require("./services/stream.service");
const { WebSocketHub } = require("./websocket/websocket-hub");

function createHttpServer({ repository, dataDir, websocketOptions } = {}) {
  const hub = new WebSocketHub(websocketOptions);
  const streamService = new StreamProcessingService({
    dataDir,
    onEvent: (event) => hub.broadcast(event),
  });
  const app = createApp(repository, { hub, streamService });
  const server = http.createServer(app);
  hub.attach(server);
  return { app, server, hub, streamService };
}

if (require.main === module) {
  const port = Number(process.env.PORT || 3005);
  const { server } = createHttpServer();
  server.listen(port, "0.0.0.0", () => {
    console.log(`lab5 license-stream-service (node) listening on ${port}`);
  });
}

module.exports = { createHttpServer };
