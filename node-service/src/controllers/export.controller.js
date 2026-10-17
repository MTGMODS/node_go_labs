const { badRequest } = require("../errors");

class ExportController {
  constructor(service) {
    this.service = service;
  }

  export = async (req, res) => {
    const raw = req.query.batch_size ?? "100";
    if (!/^\d+$/.test(raw)) throw badRequest("validation_error", "batch_size must be an integer between 1 and 1000");
    const batchSize = Number(raw);
    if (batchSize < 1 || batchSize > 1000) throw badRequest("validation_error", "batch_size must be an integer between 1 and 1000");
    res.status(200).json(await this.service.exportCSV(batchSize));
  };
}

module.exports = { ExportController };
