const express = require("express");
const os = require("os");

const DEFAULT_IO_DELAY_MS = Number(process.env.IO_DELAY_MS) || 1000;
const DEFAULT_CPU_ITERATIONS = Number(process.env.CPU_ITERATIONS) || 75_000_000;
const DEFAULT_CPU_TASKS = Number(process.env.CPU_TASKS) || 4;

const LIMITS = {
  delayMs: { min: 1, max: 30_000 },
  iterations: { min: 1, max: 500_000_000 },
  tasks: { min: 1, max: 32 },
};

function readInteger(query, name, fallback, limits) {
  const raw = query[name];
  if (raw === undefined) return fallback;
  if (typeof raw !== "string" || !/^\d+$/.test(raw)) {
    throw new Error(`${name} must be an integer between ${limits.min} and ${limits.max}`);
  }
  const value = Number(raw);
  if (!Number.isSafeInteger(value) || value < limits.min || value > limits.max) {
    throw new Error(`${name} must be an integer between ${limits.min} and ${limits.max}`);
  }
  return value;
}

function increment(iterations) {
  let value = 0;
  for (let index = 0; index < iterations; index += 1) value += 1;
  return value;
}

function runSequentialCpuWork(tasks, iterations) {
  let result = 0;
  for (let task = 0; task < tasks; task += 1) result += increment(iterations);
  return result;
}

function createWorkloadsRouter() {
  const router = express.Router();

  router.get("/io", async (req, res) => {
    let delayMs;
    try {
      delayMs = readInteger(req.query, "delay_ms", DEFAULT_IO_DELAY_MS, LIMITS.delayMs);
    } catch (err) {
      return res.status(400).json({ error: err.message });
    }
    const startedAt = process.hrtime.bigint();
    await new Promise((resolve) => setTimeout(resolve, delayMs));
    const durationMs = Number(process.hrtime.bigint() - startedAt) / 1_000_000;
    return res.status(200).json({
      status: "completed",
      workload: "io",
      runtime: "node",
      operation: "non-blocking setTimeout",
      delay_ms: delayMs,
      duration_ms: Number(durationMs.toFixed(3)),
    });
  });

  router.get("/cpu", (req, res) => {
    let iterations;
    let tasks;
    try {
      iterations = readInteger(req.query, "iterations", DEFAULT_CPU_ITERATIONS, LIMITS.iterations);
      tasks = readInteger(req.query, "tasks", DEFAULT_CPU_TASKS, LIMITS.tasks);
    } catch (err) {
      return res.status(400).json({ error: err.message });
    }
    const startedAt = process.hrtime.bigint();
    const result = runSequentialCpuWork(tasks, iterations);
    const durationMs = Number(process.hrtime.bigint() - startedAt) / 1_000_000;
    return res.status(200).json({
      status: "completed",
      workload: "cpu",
      runtime: "node",
      mode: "event-loop-blocking",
      tasks,
      iterations_per_task: iterations,
      total_iterations: tasks * iterations,
      result,
      available_parallelism:
        typeof os.availableParallelism === "function" ? os.availableParallelism() : os.cpus().length,
      duration_ms: Number(durationMs.toFixed(3)),
    });
  });

  router.all(["/io", "/cpu"], (_req, res) => {
    res.status(405).json({ error: "method not allowed" });
  });

  return router;
}

module.exports = { LIMITS, createWorkloadsRouter, increment, readInteger, runSequentialCpuWork };
