const { badRequest } = require("../errors");

function positiveInteger(value, field) {
  if (!Number.isInteger(value) || value <= 0) throw badRequest("validation_error", `${field} must be a positive integer`);
  return value;
}

class AccountController {
  constructor(service) {
    this.service = service;
  }

  list = async (_req, res) => {
    res.status(200).json(await this.service.list());
  };

  transfer = async (req, res) => {
    const fromAccountId = positiveInteger(req.body?.from_account_id, "from_account_id");
    const toAccountId = positiveInteger(req.body?.to_account_id, "to_account_id");
    const amount = req.body?.amount;
    if (typeof amount !== "number" || !Number.isFinite(amount) || amount <= 0 || Math.round(amount * 100) !== amount * 100) {
      throw badRequest("validation_error", "amount must be a positive number with at most two decimal places");
    }
    res.status(200).json(await this.service.transfer(fromAccountId, toAccountId, amount));
  };
}

module.exports = { AccountController };
