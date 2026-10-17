package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Account struct {
	ID      int    `json:"id"`
	Owner   string `json:"owner"`
	Balance string `json:"balance"`
}

type AccountError struct {
	Code    string
	Message string
}

func (e *AccountError) Error() string { return e.Message }

type AccountService struct {
	db *sql.DB
}

func NewAccountService(database *sql.DB) *AccountService {
	return &AccountService{db: database}
}

func (s *AccountService) List() ([]Account, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.db.QueryContext(ctx, "SELECT id, owner, balance::text FROM accounts ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	accounts := make([]Account, 0)
	for rows.Next() {
		var account Account
		if err := rows.Scan(&account.ID, &account.Owner, &account.Balance); err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, rows.Err()
}

func (s *AccountService) Transfer(fromID, toID int, amount float64) (_ []Account, err error) {
	if fromID == toID {
		return nil, &AccountError{Code: "invalid_transfer", Message: "Source and destination accounts must be different"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(ctx,
		"SELECT id, owner, balance::text FROM accounts WHERE id IN ($1,$2) ORDER BY id FOR UPDATE", fromID, toID)
	if err != nil {
		return nil, err
	}
	accounts := make(map[int]Account)
	for rows.Next() {
		var account Account
		if scanErr := rows.Scan(&account.ID, &account.Owner, &account.Balance); scanErr != nil {
			rows.Close()
			return nil, scanErr
		}
		accounts[account.ID] = account
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if len(accounts) != 2 {
		return nil, &AccountError{Code: "account_not_found", Message: "One or both accounts were not found"}
	}
	var sourceBalance float64
	if _, scanErr := fmt.Sscan(accounts[fromID].Balance, &sourceBalance); scanErr != nil {
		return nil, scanErr
	}
	if sourceBalance < amount {
		return nil, &AccountError{Code: "insufficient_balance", Message: fmt.Sprintf("Account %d has insufficient balance", fromID)}
	}
	if _, err = tx.ExecContext(ctx, "UPDATE accounts SET balance=balance-$1 WHERE id=$2", amount, fromID); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE accounts SET balance=balance+$1 WHERE id=$2", amount, toID); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return s.accountsByID(fromID, toID)
}

func (s *AccountService) accountsByID(first, second int) ([]Account, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, owner, balance::text FROM accounts WHERE id IN ($1,$2) ORDER BY id", first, second)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	accounts := make([]Account, 0, 2)
	for rows.Next() {
		var account Account
		if err := rows.Scan(&account.ID, &account.Owner, &account.Balance); err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, rows.Err()
}
