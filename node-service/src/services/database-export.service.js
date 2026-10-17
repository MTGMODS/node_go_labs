const fs = require("node:fs");
const fsp = require("node:fs/promises");
const path = require("node:path");
const { Readable, Transform } = require("node:stream");
const { pipeline } = require("node:stream/promises");

function csvValue(value) {
  const text = String(value ?? "");
  return `"${text.replaceAll('"', '""')}"`;
}

class DatabaseExportService {
  constructor(repository, { dataDir, publish = () => {} } = {}) {
    this.repository = repository;
    this.dataDir = dataDir || process.env.DATA_DIR || path.join(process.cwd(), "data");
    this.publish = publish;
  }

  async exportCSV(batchSize = 100) {
    await fsp.mkdir(this.dataDir, { recursive: true });
    const filename = `licenses-node-${Date.now()}.csv`;
    const outputPath = path.join(this.dataDir, filename);
    const total = await this.repository.count();
    let processed = 0;
    const publish = this.publish;

    async function* source(repository) {
      yield { header: true };
      for await (const license of repository.iterate(batchSize)) yield license;
    }

    const toCSV = new Transform({
      writableObjectMode: true,
      transform(record, _encoding, callback) {
        if (record.header) return callback(null, "id,key,product,owner,status,duration_days,max_devices\n");
        processed += 1;
        const row = [record.id, record.key, record.product, record.owner, record.status, record.duration_days, record.max_devices]
          .map(csvValue).join(",") + "\n";
        publish({ type: "export_progress", runtime: "node", processed, total, timestamp: new Date().toISOString() });
        return callback(null, row);
      },
    });

    await pipeline(Readable.from(source(this.repository)), toCSV, fs.createWriteStream(outputPath));
    const event = { type: "export_completed", runtime: "node", processed, total, filename, timestamp: new Date().toISOString() };
    this.publish(event);
    return { filename, processed, total, backpressure: "Readable.pipe(Transform).pipe(Writable) via stream.pipeline" };
  }
}

module.exports = { DatabaseExportService };
