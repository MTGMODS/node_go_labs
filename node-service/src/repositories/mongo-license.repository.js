const { repositoryError } = require("../errors");

function fromDocument(document) {
  if (!document) return null;
  const { _id, ...license } = document;
  return { id: _id, ...license };
}

class MongoLicenseRepository {
  constructor(db) {
    this.collection = db.collection("licenses");
    this.counters = db.collection("counters");
  }

  async initialize() {
    await this.collection.createIndex({ key: 1 }, { unique: true });
  }

  async list() {
    return (await this.collection.find({}).sort({ _id: 1 }).toArray()).map(fromDocument);
  }

  async findById(id) {
    return fromDocument(await this.collection.findOne({ _id: id }));
  }

  async findByKey(key) {
    return fromDocument(await this.collection.findOne({ key }));
  }

  async nextId() {
    const counter = await this.counters.findOneAndUpdate(
      { _id: "licenses" },
      { $inc: { value: 1 } },
      { upsert: true, returnDocument: "after" },
    );
    return counter.value;
  }

  async create(license) {
    const document = { _id: await this.nextId(), ...license };
    try {
      await this.collection.insertOne(document);
      return fromDocument(document);
    } catch (error) {
      if (error.code === 11000) throw repositoryError("duplicate_key", "License key already exists");
      throw error;
    }
  }

  async update(license) {
    const { id, ...fields } = license;
    try {
      const document = await this.collection.findOneAndUpdate(
        { _id: id },
        { $set: fields },
        { returnDocument: "after" },
      );
      return fromDocument(document);
    } catch (error) {
      if (error.code === 11000) throw repositoryError("duplicate_key", "License key already exists");
      throw error;
    }
  }

  async delete(id) {
    return (await this.collection.deleteOne({ _id: id })).deletedCount === 1;
  }

  async count() {
    return this.collection.countDocuments();
  }

  async *iterate(batchSize = 100) {
    const cursor = this.collection.find({}).sort({ _id: 1 }).batchSize(batchSize);
    for await (const document of cursor) yield fromDocument(document);
  }
}

module.exports = { MongoLicenseRepository };
