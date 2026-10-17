const applicationDatabase = db.getSiblingDB(process.env.MONGO_INITDB_DATABASE);
applicationDatabase.licenses.createIndex({ key: 1 }, { unique: true });
applicationDatabase.events.createIndex({ timestamp: -1 });
