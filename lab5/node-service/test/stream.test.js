const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const os = require("node:os");
const path = require("node:path");
const { StreamProcessingService } = require("../src/services/stream.service");

test("stream service generates and processes a file with and without backpressure", async () => {
  const dataDir = await fs.mkdtemp(path.join(os.tmpdir(), "lab5-node-stream-"));
  const events = [];
  const service = new StreamProcessingService({ dataDir, onEvent: (event) => events.push(event) });

  try {
    const source = await service.generateSource(1);
    assert.equal(source.sizeBytes, 1024 * 1024);

    const safe = await service.startProcessing({ mode: "safe", slowDelayMs: 0, highWaterMarkKB: 16 });
    const safeResult = await service.waitForJob(safe.id);
    assert.equal(safeResult.status, "completed");
    assert.equal(safeResult.processedBytes, source.sizeBytes);
    assert.ok(safeResult.lineCount > 0);

    const sourceData = await fs.readFile(path.join(dataDir, "licenses-source.ndjson"));
    const expectedOutput = Buffer.from(sourceData.toString("utf8").toUpperCase());
    const safeOutput = await fs.readFile(path.join(dataDir, safeResult.outputFile));
    assert.equal(safeOutput.length, sourceData.length);
    assert.deepEqual(safeOutput, expectedOutput);

    const unsafe = await service.startProcessing({ mode: "unsafe", slowDelayMs: 0, highWaterMarkKB: 16 });
    const unsafeResult = await service.waitForJob(unsafe.id);
    assert.equal(unsafeResult.status, "completed");
    assert.equal(unsafeResult.processedBytes, source.sizeBytes);
    const unsafeOutput = await fs.readFile(path.join(dataDir, unsafeResult.outputFile));
    assert.equal(unsafeOutput.length, sourceData.length);
    assert.deepEqual(unsafeOutput, expectedOutput);
    assert.ok(events.some((event) => event.type === "progress"));
    assert.ok(events.some((event) => event.type === "completed"));
  } finally {
    await fs.rm(dataDir, { recursive: true, force: true });
  }
});

test("stream service requires a generated source file", async () => {
  const dataDir = await fs.mkdtemp(path.join(os.tmpdir(), "lab5-node-empty-"));
  const service = new StreamProcessingService({ dataDir });
  try {
    await assert.rejects(
      () => service.startProcessing(),
      (error) => error.code === "stream_source_not_found",
    );
  } finally {
    await fs.rm(dataDir, { recursive: true, force: true });
  }
});
