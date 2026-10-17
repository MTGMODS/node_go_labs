const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const os = require("node:os");
const path = require("node:path");
const { InMemoryLicenseRepository } = require("../src/repositories/license.repository");
const { DatabaseExportService } = require("../src/services/database-export.service");

test("database export streams repository records to CSV", async () => {
  const directory = await fs.mkdtemp(path.join(os.tmpdir(), "lab6-node-export-"));
  try {
    const repository = new InMemoryLicenseRepository();
    await repository.create({ key: "EXPORT-1", product: "VIP", owner: "one", status: "ACTIVE", duration_days: 30, max_devices: 1 });
    await repository.create({ key: "EXPORT-2", product: "VIP", owner: "two", status: "ACTIVE", duration_days: 30, max_devices: 1 });
    const progress = [];
    const service = new DatabaseExportService(repository, { dataDir: directory, publish: (event) => progress.push(event) });
    const result = await service.exportCSV(1);
    const csv = await fs.readFile(path.join(directory, result.filename), "utf8");
    assert.equal(result.processed, 2);
    assert.match(csv, /EXPORT-1/);
    assert.match(csv, /EXPORT-2/);
    assert.equal(progress.filter((event) => event.type === "export_progress").length, 2);
  } finally {
    await fs.rm(directory, { recursive: true, force: true });
  }
});
