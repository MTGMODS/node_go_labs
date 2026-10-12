const test = require("node:test");
const assert = require("node:assert/strict");
const { InMemoryLicenseRepository } = require("../src/repositories/license.repository");
const { LicenseService } = require("../src/services/license.service");

function createService() {
  return new LicenseService(new InMemoryLicenseRepository());
}

const input = { key: "MTGM-VIP1-AAAA-0001", product: "MTG MODS VIP", owner: "student" };

test("service creates and finds a license with defaults", async () => {
  const service = createService();
  const created = await service.createLicense({ key: ` ${input.key} `, product: ` ${input.product} `, owner: " student " });
  assert.deepEqual(created, {
    id: 1,
    ...input,
    status: "NOT_ACTIVATED",
    duration_days: 30,
    max_devices: 1,
  });
  assert.deepEqual(await service.getLicense(1), created);
});

test("service partially updates a license", async () => {
  const service = createService();
  await service.createLicense(input);
  const updated = await service.updateLicense(1, { status: "ACTIVE", max_devices: 2 });
  assert.equal(updated.key, input.key);
  assert.equal(updated.status, "ACTIVE");
  assert.equal(updated.max_devices, 2);
});

test("service deletes a license", async () => {
  const service = createService();
  await service.createLicense(input);
  await service.deleteLicense(1);
  await assert.rejects(() => service.getLicense(1), (error) => error.code === "license_not_found");
});

test("service rejects duplicate license keys", async () => {
  const service = createService();
  await service.createLicense(input);
  await assert.rejects(
    () => service.createLicense({ ...input, owner: "other" }),
    (error) => error.code === "license_key_conflict",
  );
});

test("service reports a missing license", async () => {
  await assert.rejects(() => createService().getLicense(42), (error) => error.code === "license_not_found");
});
