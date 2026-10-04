const { createApp } = require("./app");

const port = Number(process.env.PORT) || 3004;
createApp().listen(port, "0.0.0.0", () => {
  console.log(`lab4 node service listening on ${port}`);
});
