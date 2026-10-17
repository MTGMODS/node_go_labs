const { domainError } = require("../errors");

class AccountService {
  constructor(pool) {
    this.pool = pool;
  }

  async list() {
    const result = await this.pool.query("SELECT id, owner, balance::text AS balance FROM accounts ORDER BY id");
    return result.rows;
  }

  async transfer(fromAccountId, toAccountId, amount) {
    if (fromAccountId === toAccountId) {
      throw domainError("invalid_transfer", "Source and destination accounts must be different");
    }

    const client = await this.pool.connect();
    try {
      await client.query("BEGIN");
      const ids = [fromAccountId, toAccountId].sort((a, b) => a - b);
      const locked = await client.query(
        "SELECT id, owner, balance::text AS balance FROM accounts WHERE id = ANY($1::int[]) ORDER BY id FOR UPDATE",
        [ids],
      );
      if (locked.rows.length !== 2) throw domainError("account_not_found", "One or both accounts were not found");
      const source = locked.rows.find((account) => account.id === fromAccountId);
      if (Number(source.balance) < amount) {
        throw domainError("insufficient_balance", `Account ${fromAccountId} has insufficient balance`);
      }

      await client.query("UPDATE accounts SET balance = balance - $1 WHERE id = $2", [amount, fromAccountId]);
      await client.query("UPDATE accounts SET balance = balance + $1 WHERE id = $2", [amount, toAccountId]);
      await client.query("COMMIT");

      const result = await this.pool.query(
        "SELECT id, owner, balance::text AS balance FROM accounts WHERE id = ANY($1::int[]) ORDER BY id",
        [ids],
      );
      return { status: "committed", accounts: result.rows };
    } catch (error) {
      await client.query("ROLLBACK");
      throw error;
    } finally {
      client.release();
    }
  }
}

module.exports = { AccountService };
