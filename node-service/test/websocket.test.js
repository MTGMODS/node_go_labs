const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const os = require("node:os");
const path = require("node:path");
const { once } = require("node:events");
const { WebSocket } = require("ws");
const { createHttpServer } = require("../src/server");
const { WebSocketHub } = require("../src/websocket/websocket-hub");

async function waitFor(messages, predicate) {
  const deadline = Date.now() + 2000;
  while (Date.now() < deadline) {
    const match = messages.find(predicate);
    if (match) return match;
    await new Promise((resolve) => setTimeout(resolve, 10));
  }
  throw new Error("Timed out waiting for WebSocket message");
}

test("WebSocket server registers clients, validates messages and broadcasts", async () => {
  const dataDir = await fs.mkdtemp(path.join(os.tmpdir(), "lab5-node-ws-"));
  const runtime = createHttpServer({ dataDir });
  runtime.server.listen(0, "127.0.0.1");
  await once(runtime.server, "listening");
  const { port } = runtime.server.address();
  const first = new WebSocket(`ws://127.0.0.1:${port}/ws`);
  const second = new WebSocket(`ws://127.0.0.1:${port}/ws`);
  const firstMessages = [];
  const secondMessages = [];
  first.on("message", (data) => firstMessages.push(JSON.parse(data.toString())));
  second.on("message", (data) => secondMessages.push(JSON.parse(data.toString())));

  try {
    await Promise.all([once(first, "open"), once(second, "open")]);
    await Promise.all([
      waitFor(firstMessages, (message) => message.type === "connected"),
      waitFor(secondMessages, (message) => message.type === "connected"),
    ]);

    first.send(JSON.stringify({ type: "message", text: "Hello" }));
    const broadcast = await waitFor(secondMessages, (message) => message.type === "message");
    assert.equal(broadcast.text, "Hello");

    first.send("not-json");
    const error = await waitFor(firstMessages, (message) => message.type === "error");
    assert.equal(error.error, "invalid_json");
    assert.equal(runtime.hub.stats().activeClients, 2);
  } finally {
    first.terminate();
    second.terminate();
    runtime.hub.close();
    await new Promise((resolve) => runtime.server.close(resolve));
    await fs.rm(dataDir, { recursive: true, force: true });
  }
});

test("WebSocket hub terminates a client that repeatedly exceeds the buffer limit", () => {
  const hub = new WebSocketHub({ maxBufferedBytes: 1, maxDroppedMessages: 2 });
  let terminated = false;
  hub.clients.set("slow", {
    id: "slow",
    socket: {
      readyState: WebSocket.OPEN,
      bufferedAmount: 2,
      terminate: () => { terminated = true; },
    },
    droppedMessages: 0,
    state: "OPEN",
  });

  hub.broadcast({ type: "progress" });
  hub.broadcast({ type: "progress" });
  assert.equal(terminated, true);
  assert.equal(hub.clients.size, 0);
});
