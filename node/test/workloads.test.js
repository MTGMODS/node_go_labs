const test = require("node:test");
const assert = require("node:assert/strict");
const { increment, readInteger, runSequentialCpuWork } = require("../src/workloads");

test("increment performs the requested amount of work", () => {
  assert.equal(increment(10), 10);
});

test("sequential workload preserves total work", () => {
  assert.equal(runSequentialCpuWork(4, 25), 100);
});

test("integer query parser accepts valid values", () => {
  assert.equal(readInteger({ iterations: "42" }, "iterations", 10, { min: 1, max: 100 }), 42);
});

test("integer query parser rejects malformed and out-of-range values", () => {
  assert.throws(() => readInteger({ tasks: "2.5" }, "tasks", 1, { min: 1, max: 32 }), /tasks must be an integer/);
  assert.throws(() => readInteger({ tasks: "33" }, "tasks", 1, { min: 1, max: 32 }), /tasks must be an integer/);
});
