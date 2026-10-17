const { badRequest } = require("../errors");

function readInteger(value, field, fallback, min, max) {
  if (value === undefined) return fallback;
  if (!Number.isInteger(value) || value < min || value > max) {
    throw badRequest("validation_error", `${field} must be an integer between ${min} and ${max}`);
  }
  return value;
}

class StreamController {
  constructor(service) {
    this.service = service;
  }

  sourceInfo = async (_req, res) => {
    res.status(200).json(await this.service.sourceInfo());
  };

  generateSource = async (req, res) => {
    const sizeMB = readInteger(req.body?.size_mb, "size_mb", 100, 1, 1024);
    res.status(201).json(await this.service.generateSource(sizeMB));
  };

  start = async (req, res) => {
    const mode = req.body?.mode ?? "safe";
    if (!new Set(["safe", "unsafe"]).has(mode)) {
      throw badRequest("validation_error", "mode must be safe or unsafe");
    }
    const slowDelayMs = readInteger(req.body?.slow_delay_ms, "slow_delay_ms", 2, 0, 100);
    const highWaterMarkKB = readInteger(req.body?.high_water_mark_kb, "high_water_mark_kb", 64, 4, 1024);
    res.status(202).json(await this.service.startProcessing({ mode, slowDelayMs, highWaterMarkKB }));
  };

  listJobs = async (_req, res) => {
    res.status(200).json(this.service.listJobs());
  };

  getJob = async (req, res) => {
    res.status(200).json(this.service.getJob(req.params.id));
  };
}

module.exports = { StreamController };
