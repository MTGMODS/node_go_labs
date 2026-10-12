const crypto = require("node:crypto");
const fs = require("node:fs");
const fsp = require("node:fs/promises");
const path = require("node:path");
const { once } = require("node:events");
const { Readable, Transform } = require("node:stream");
const { pipeline } = require("node:stream/promises");
const { domainError } = require("../errors");

const MEBIBYTE = 1024 * 1024;

class StreamProcessingService {
  constructor({ dataDir, onEvent = () => {} } = {}) {
    this.dataDir = dataDir || process.env.DATA_DIR || path.join(process.cwd(), "data");
    this.sourcePath = path.join(this.dataDir, "licenses-source.ndjson");
    this.jobs = new Map();
    this.jobPromises = new Map();
    this.onEvent = onEvent;
  }

  async generateSource(sizeMB = 100) {
    await fsp.mkdir(this.dataDir, { recursive: true });
    const targetBytes = sizeMB * MEBIBYTE;
    const line = Buffer.from(JSON.stringify({
      key: "MTGM-STREAM-DEMO",
      product: "MTG MODS VIP",
      owner: "stream-user",
      status: "NOT_ACTIVATED",
    }) + "\n");
    const repetitions = Math.ceil((64 * 1024) / line.length);
    const block = Buffer.concat(Array.from({ length: repetitions }, () => line));

    async function* source() {
      let written = 0;
      while (written < targetBytes) {
        const chunk = block.subarray(0, Math.min(block.length, targetBytes - written));
        written += chunk.length;
        yield chunk;
      }
    }

    await pipeline(Readable.from(source()), fs.createWriteStream(this.sourcePath));
    const stat = await fsp.stat(this.sourcePath);
    return { path: path.basename(this.sourcePath), sizeBytes: stat.size, sizeMB: stat.size / MEBIBYTE };
  }

  async sourceInfo() {
    try {
      const stat = await fsp.stat(this.sourcePath);
      return { exists: true, path: path.basename(this.sourcePath), sizeBytes: stat.size, sizeMB: stat.size / MEBIBYTE };
    } catch (error) {
      if (error.code === "ENOENT") return { exists: false, path: path.basename(this.sourcePath) };
      throw error;
    }
  }

  async startProcessing({ mode = "safe", slowDelayMs = 2, highWaterMarkKB = 64 } = {}) {
    const source = await this.sourceInfo();
    if (!source.exists) {
      throw domainError("stream_source_not_found", "Generate the source file before starting processing");
    }

    const id = crypto.randomUUID();
    const job = {
      id,
      runtime: "node",
      mode,
      status: "running",
      sourceBytes: source.sizeBytes,
      processedBytes: 0,
      progressPercent: 0,
      lineCount: 0,
      slowDelayMs,
      highWaterMarkKB,
      outputFile: `processed-${mode}-${id}.ndjson`,
      startedAt: new Date().toISOString(),
    };
    this.jobs.set(id, job);
    const promise = this.runJob(job).catch(() => {}).finally(() => this.jobPromises.delete(id));
    this.jobPromises.set(id, promise);
    return this.snapshot(job);
  }

  getJob(id) {
    const job = this.jobs.get(id);
    if (!job) throw domainError("stream_job_not_found", `Stream job ${id} was not found`);
    return this.snapshot(job);
  }

  listJobs() {
    return [...this.jobs.values()].map((job) => this.snapshot(job));
  }

  async waitForJob(id) {
    const promise = this.jobPromises.get(id);
    if (promise) await promise;
    return this.getJob(id);
  }

  async runJob(job) {
    const outputPath = path.join(this.dataDir, job.outputFile);
    const highWaterMark = job.highWaterMarkKB * 1024;
    const baselineRSS = process.memoryUsage().rss;
    let peakRSS = baselineRSS;
    const started = process.hrtime.bigint();
    const sampler = setInterval(() => {
      peakRSS = Math.max(peakRSS, process.memoryUsage().rss);
    }, 5);

    const counter = this.createCounterTransform(job, highWaterMark);
    const slowConsumer = new Transform({
      highWaterMark,
      transform(chunk, _encoding, callback) {
        setTimeout(() => callback(null, chunk), job.slowDelayMs);
      },
    });

    try {
      if (job.mode === "safe") {
        await pipeline(
          fs.createReadStream(this.sourcePath, { highWaterMark }),
          counter,
          slowConsumer,
          fs.createWriteStream(outputPath, { highWaterMark }),
        );
      } else {
        await this.runWithoutBackpressure(counter, slowConsumer, outputPath, highWaterMark);
      }

      const durationMs = Number(process.hrtime.bigint() - started) / 1e6;
      job.status = "completed";
      job.progressPercent = 100;
      job.completedAt = new Date().toISOString();
      job.durationMs = Number(durationMs.toFixed(2));
      job.peakRSSBytes = peakRSS;
      job.rssGrowthBytes = Math.max(0, peakRSS - baselineRSS);
      this.onEvent({ type: "completed", job: this.snapshot(job) });
    } catch (error) {
      job.status = "failed";
      job.error = error.message;
      job.completedAt = new Date().toISOString();
      this.onEvent({ type: "failed", job: this.snapshot(job) });
      throw error;
    } finally {
      clearInterval(sampler);
    }
  }

  createCounterTransform(job, highWaterMark) {
    let hasData = false;
    let lastByte = null;
    let nextNotification = MEBIBYTE;
    return new Transform({
      highWaterMark,
      transform: (chunk, _encoding, callback) => {
        for (const byte of chunk) {
          if (byte === 0x0a) job.lineCount += 1;
        }
        hasData = true;
        lastByte = chunk[chunk.length - 1];
        job.processedBytes += chunk.length;
        job.progressPercent = Number(((job.processedBytes / job.sourceBytes) * 100).toFixed(2));

        if (job.processedBytes >= nextNotification || job.processedBytes === job.sourceBytes) {
          nextNotification = job.processedBytes + MEBIBYTE;
          this.onEvent({
            type: "progress",
            jobId: job.id,
            runtime: "node",
            mode: job.mode,
            processed: job.processedBytes,
            total: job.sourceBytes,
            percent: job.progressPercent,
          });
        }

        callback(null, Buffer.from(chunk.toString("utf8").toUpperCase()));
      },
      flush(callback) {
        if (hasData && lastByte !== 0x0a) {
          job.lineCount += 1;
        }
        callback();
      },
    });
  }

  async runWithoutBackpressure(counter, slowConsumer, outputPath, highWaterMark) {
    const readable = fs.createReadStream(this.sourcePath, { highWaterMark });
    const writable = fs.createWriteStream(outputPath, { highWaterMark });
    counter.pipe(slowConsumer).pipe(writable);

    const failed = new Promise((_, reject) => {
      for (const stream of [readable, counter, slowConsumer, writable]) stream.once("error", reject);
    });
    readable.on("data", (chunk) => {
      counter.write(chunk); // intentionally ignores false: demonstrates missing Backpressure
    });
    readable.on("end", () => counter.end());

    await Promise.race([once(writable, "finish"), failed]);
  }

  snapshot(job) {
    return { ...job };
  }
}

module.exports = { StreamProcessingService };
