const test = require("node:test");
const assert = require("node:assert/strict");
const { createApp } = require("../src/app");

async function withServer(run) {
  const server = createApp().listen(0, "127.0.0.1");
  await new Promise((resolve) => server.once("listening", resolve));
  try {
    const { port } = server.address();
    await run(`http://127.0.0.1:${port}`);
  } finally {
    await new Promise((resolve) => server.close(resolve));
  }
}

async function postUser(baseURL, body) {
  return fetch(`${baseURL}/api/users`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(body),
  });
}

test("API supports successful POST and GET", async () => {
  await withServer(async (baseURL) => {
    const created = await postUser(baseURL, { name: "John Doe", email: "john@example.com" });
    assert.equal(created.status, 201);
    const response = await fetch(`${baseURL}/api/users/1`);
    assert.equal(response.status, 200);
    assert.equal((await response.json()).email, "john@example.com");
  });
});

test("API rejects an invalid POST", async () => {
  await withServer(async (baseURL) => {
    const response = await postUser(baseURL, { name: "", email: "invalid" });
    assert.equal(response.status, 400);
    assert.equal((await response.json()).error, "validation_error");
  });
});

test("API returns 404 for a missing user", async () => {
  await withServer(async (baseURL) => {
    const response = await fetch(`${baseURL}/api/users/42`);
    assert.equal(response.status, 404);
    assert.equal((await response.json()).error, "user_not_found");
  });
});

test("API deletes a user", async () => {
  await withServer(async (baseURL) => {
    await postUser(baseURL, { name: "John Doe", email: "john@example.com" });
    const deleted = await fetch(`${baseURL}/api/users/1`, { method: "DELETE" });
    assert.equal(deleted.status, 204);
    assert.equal((await fetch(`${baseURL}/api/users/1`)).status, 404);
  });
});

test("API returns 409 for a duplicate email and 400 for invalid JSON", async () => {
  await withServer(async (baseURL) => {
    await postUser(baseURL, { name: "John", email: "john@example.com" });
    assert.equal((await postUser(baseURL, { name: "Jane", email: "john@example.com" })).status, 409);
    const malformed = await fetch(`${baseURL}/api/users`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: "{",
    });
    assert.equal(malformed.status, 400);
    assert.equal((await malformed.json()).error, "invalid_json");
  });
});

test("API converts an unexpected repository error to 500", async () => {
  const failingRepository = {
    async list() {
      throw new Error("storage failed");
    },
  };
  const server = createApp(failingRepository).listen(0, "127.0.0.1");
  await new Promise((resolve) => server.once("listening", resolve));
  try {
    const { port } = server.address();
    const response = await fetch(`http://127.0.0.1:${port}/api/users`);
    assert.equal(response.status, 500);
    assert.equal((await response.json()).error, "internal_error");
  } finally {
    await new Promise((resolve) => server.close(resolve));
  }
});
