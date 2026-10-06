const test = require("node:test");
const assert = require("node:assert/strict");
const { createApp } = require("../src/app");

async function withServer(run, repository) {
  const server = createApp(repository).listen(0, "127.0.0.1");
  await new Promise((resolve) => server.once("listening", resolve));
  try {
    const { port } = server.address();
    await run(`http://127.0.0.1:${port}`);
  } finally {
    await new Promise((resolve) => server.close(resolve));
  }
}

function send(baseURL, method, path, body) {
  return fetch(`${baseURL}${path}`, {
    method,
    headers: body === undefined ? undefined : { "content-type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
}

const license = { key: "MTGM-VIP1-AAAA-0001", product: "MTG MODS VIP", owner: "student" };

test("API supports successful POST and GET", async () => {
  await withServer(async (baseURL) => {
    const created = await send(baseURL, "POST", "/licenses", license);
    assert.equal(created.status, 201);
    assert.equal((await created.json()).status, "NOT_ACTIVATED");
    const found = await send(baseURL, "GET", "/licenses/1");
    assert.equal(found.status, 200);
    assert.equal((await found.json()).key, license.key);
  });
});

test("API rejects an invalid POST before the service", async () => {
  await withServer(async (baseURL) => {
    const response = await send(baseURL, "POST", "/licenses", { key: "", product: "VIP" });
    assert.equal(response.status, 400);
  });
});

test("API returns 404 for a missing license", async () => {
  await withServer(async (baseURL) => {
    const response = await send(baseURL, "GET", "/licenses/42");
    assert.equal(response.status, 404);
  });
});

test("API supports partial PUT and DELETE", async () => {
  await withServer(async (baseURL) => {
    await send(baseURL, "POST", "/licenses", license);
    const updated = await send(baseURL, "PUT", "/licenses/1", { status: "ACTIVE" });
    assert.equal(updated.status, 200);
    assert.equal((await updated.json()).status, "ACTIVE");
    const deleted = await send(baseURL, "DELETE", "/licenses/1");
    assert.equal(deleted.status, 204);
  });
});

test("API returns 409 for a duplicate key and 400 for invalid JSON", async () => {
  await withServer(async (baseURL) => {
    await send(baseURL, "POST", "/licenses", license);
    const conflict = await send(baseURL, "POST", "/licenses", { ...license, owner: "other" });
    assert.equal(conflict.status, 409);
    const invalid = await fetch(`${baseURL}/licenses`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: "{",
    });
    assert.equal(invalid.status, 400);
  });
});

test("API converts an unexpected repository error to 500", async () => {
  const failingRepository = {
    list: async () => { throw new Error("storage failed"); },
    findById: async () => null,
    findByKey: async () => null,
    create: async () => null,
    update: async () => null,
    delete: async () => false,
  };
  await withServer(async (baseURL) => {
    const response = await send(baseURL, "GET", "/licenses");
    assert.equal(response.status, 500);
  }, failingRepository);
});
