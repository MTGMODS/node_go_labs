const { WebSocket } = require("ws");

const url = process.argv[2] || "ws://127.0.0.1:3006/ws";
const first = new WebSocket(url);
const second = new WebSocket(url);
let ready = 0;

function onReady() {
  ready += 1;
  if (ready !== 2) return;
  first.send(JSON.stringify({ type: "message", text: "Hello from client A" }));
  setTimeout(() => first.send("invalid-json"), 100);
  setTimeout(() => {
    first.close();
    second.close();
  }, 500);
}

for (const [name, socket] of [["A", first], ["B", second]]) {
  socket.on("open", onReady);
  socket.on("message", (data) => console.log(`client ${name}: ${data}`));
  socket.on("error", (error) => {
    console.error(`client ${name}: ${error.message}`);
    process.exitCode = 1;
  });
}
