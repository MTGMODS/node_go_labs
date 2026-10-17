class RealtimeController {
  constructor(hub) {
    this.hub = hub;
  }

  stats = (_req, res) => {
    res.status(200).json(this.hub.stats());
  };
}

module.exports = { RealtimeController };
