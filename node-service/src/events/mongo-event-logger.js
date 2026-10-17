class MongoEventLogger {
  constructor(db) {
    this.collection = db.collection("events");
  }

  async log(event) {
    await this.collection.insertOne({ ...event, timestamp: new Date(event.timestamp) });
  }

  async list(limit = 50) {
    const documents = await this.collection.find({}).sort({ timestamp: -1 }).limit(limit).toArray();
    return documents.map(({ _id, timestamp, ...event }) => ({
      id: _id.toString(),
      ...event,
      timestamp: timestamp.toISOString(),
    }));
  }
}

module.exports = { MongoEventLogger };
