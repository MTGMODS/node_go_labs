const test = require("node:test");
const assert = require("node:assert/strict");
const { InMemoryUserRepository } = require("../src/repositories/user.repository");
const { UserService } = require("../src/services/user.service");

function createService() {
  return new UserService(new InMemoryUserRepository());
}

test("service creates and finds a user", async () => {
  const service = createService();
  const created = await service.createUser({ name: " John Doe ", email: " JOHN@EXAMPLE.COM " });
  assert.deepEqual(created, { id: 1, name: "John Doe", email: "john@example.com" });
  assert.deepEqual(await service.getUser(1), created);
});

test("service updates a user", async () => {
  const service = createService();
  await service.createUser({ name: "John", email: "john@example.com" });
  const updated = await service.updateUser(1, { name: "Jane", email: "jane@example.com" });
  assert.deepEqual(updated, { id: 1, name: "Jane", email: "jane@example.com" });
});

test("service deletes a user", async () => {
  const service = createService();
  await service.createUser({ name: "John", email: "john@example.com" });
  await service.deleteUser(1);
  await assert.rejects(() => service.getUser(1), (error) => error.code === "user_not_found");
});

test("service rejects duplicate emails", async () => {
  const service = createService();
  await service.createUser({ name: "John", email: "john@example.com" });
  await assert.rejects(
    () => service.createUser({ name: "Other", email: "JOHN@example.com" }),
    (error) => error.code === "email_conflict",
  );
});

test("service reports a missing user", async () => {
  await assert.rejects(() => createService().getUser(42), (error) => error.code === "user_not_found");
});
