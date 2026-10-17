const crypto = require("node:crypto");
const { WebSocket, WebSocketServer } = require("ws");

class WebSocketHub {
  constructor({ maxBufferedBytes = 256 * 1024, maxDroppedMessages = 5, heartbeatMs = 30_000 } = {}) {
    this.maxBufferedBytes = maxBufferedBytes;
    this.maxDroppedMessages = maxDroppedMessages;
    this.heartbeatMs = heartbeatMs;
    this.clients = new Map();
    this.wss = null;
    this.heartbeat = null;
  }

  attach(server) {
    if (this.wss) return this.wss;
    this.wss = new WebSocketServer({ server, path: "/ws", maxPayload: 16 * 1024 });
    this.wss.on("connection", (socket, request) => this.register(socket, request));
    this.heartbeat = setInterval(() => this.checkConnections(), this.heartbeatMs);
    this.heartbeat.unref();
    return this.wss;
  }

  register(socket, request) {
    const clientId = crypto.randomUUID();
    const client = {
      id: clientId,
      socket,
      connectedAt: new Date().toISOString(),
      state: "OPEN",
      alive: true,
      droppedMessages: 0,
      remoteAddress: request.socket.remoteAddress,
    };
    this.clients.set(clientId, client);

    socket.on("pong", () => { client.alive = true; });
    socket.on("message", (data, isBinary) => this.handleMessage(client, data, isBinary));
    socket.on("close", () => this.unregister(clientId));
    socket.on("error", () => this.unregister(clientId));

    this.send(client, {
      type: "connected",
      clientId,
      connectedAt: client.connectedAt,
    });
  }

  handleMessage(client, data, isBinary) {
    if (isBinary) {
      this.sendError(client, "invalid_message", "Binary messages are not supported");
      return;
    }

    let message;
    try {
      message = JSON.parse(data.toString());
    } catch {
      this.sendError(client, "invalid_json", "Message must be valid JSON");
      return;
    }

    if (message?.type !== "message" || typeof message.text !== "string" ||
        !message.text.trim() || message.text.length > 4096) {
      this.sendError(client, "invalid_message", "Expected { type: message, text: non-empty string }");
      return;
    }

    this.broadcast({
      type: "message",
      clientId: client.id,
      text: message.text.trim(),
      sentAt: new Date().toISOString(),
    }, { excludeId: client.id });
  }

  broadcast(payload, { excludeId } = {}) {
    const serialized = JSON.stringify(payload);
    for (const client of this.clients.values()) {
      if (client.id === excludeId || client.socket.readyState !== WebSocket.OPEN) continue;

      if (client.socket.bufferedAmount > this.maxBufferedBytes) {
        client.droppedMessages += 1;
        if (client.droppedMessages >= this.maxDroppedMessages) {
          client.state = "TERMINATED_SLOW_CONSUMER";
          client.socket.terminate();
          this.clients.delete(client.id);
        }
        continue;
      }

      client.socket.send(serialized);
    }
  }

  send(client, payload) {
    if (client.socket.readyState === WebSocket.OPEN) {
      client.socket.send(JSON.stringify(payload));
    }
  }

  sendError(client, code, message) {
    this.send(client, { type: "error", error: code, message });
  }

  unregister(clientId) {
    const client = this.clients.get(clientId);
    if (client) client.state = "CLOSED";
    this.clients.delete(clientId);
  }

  checkConnections() {
    for (const client of this.clients.values()) {
      if (!client.alive) {
        client.state = "TERMINATED_HEARTBEAT";
        client.socket.terminate();
        this.clients.delete(client.id);
        continue;
      }
      client.alive = false;
      client.socket.ping();
    }
  }

  stats() {
    return {
      activeClients: this.clients.size,
      maxBufferedBytes: this.maxBufferedBytes,
      slowClientPolicy: `drop ${this.maxDroppedMessages} messages, then terminate`,
      clients: [...this.clients.values()].map(({ socket: _socket, alive: _alive, ...client }) => client),
    };
  }

  close() {
    if (this.heartbeat) clearInterval(this.heartbeat);
    for (const client of this.clients.values()) client.socket.terminate();
    this.clients.clear();
    if (this.wss) this.wss.close();
  }
}

module.exports = { WebSocketHub };
