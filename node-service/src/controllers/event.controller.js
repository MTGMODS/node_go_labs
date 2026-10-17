const { badRequest } = require("../errors");

class EventController {
  constructor(eventLogger) {
    this.eventLogger = eventLogger;
  }

  list = async (req, res) => {
    const rawLimit = req.query.limit ?? "50";
    if (!/^\d+$/.test(rawLimit)) throw badRequest("validation_error", "limit must be an integer between 1 and 200");
    const limit = Number(rawLimit);
    if (limit < 1 || limit > 200) throw badRequest("validation_error", "limit must be an integer between 1 and 200");
    res.status(200).json(await this.eventLogger.list(limit));
  };
}

module.exports = { EventController };
