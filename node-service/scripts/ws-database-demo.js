const WebSocket = require("ws");

const wsURL = process.argv[2] || "ws://127.0.0.1:3006/ws";
const httpURL = process.argv[3] || "http://127.0.0.1:3006";

function connect(name) {
  return new Promise((resolve, reject) => {
    const socket = new WebSocket(wsURL);
    const messages = [];
    socket.on("message", (data) => messages.push(JSON.parse(data.toString())));
    socket.once("open", () => resolve({ name, socket, messages }));
    socket.once("error", reject);
  });
}

async function waitForEvent(client, eventType, timeoutMs = 5000) {
  const started = Date.now();
  while (Date.now() - started < timeoutMs) {
    const event = client.messages.find((message) => message.type === eventType);
    if (event) return event;
    await new Promise((resolve) => setTimeout(resolve, 25));
  }
  throw new Error(`${client.name} did not receive ${eventType}`);
}

async function main() {
  const clients = await Promise.all([connect("client-a"), connect("client-b")]);
  const key = `WS-DEMO-${Date.now()}`;
  const response = await fetch(`${httpURL}/api/licenses`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ key, product: "MTG MODS VIP", owner: "websocket-demo" }),
  });
  if (response.status !== 201) throw new Error(`POST failed with ${response.status}: ${await response.text()}`);
  const received = await Promise.all(clients.map((client) => waitForEvent(client, "license_created")));
  console.log(JSON.stringify({ clients: clients.length, event: received[0].type, key, receivedByAll: true }, null, 2));
  for (const client of clients) client.socket.close();
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
