package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"finance-system/database"
	"finance-system/model"
)

// AccountService 账户服务
type AccountService struct{}

// NewAccountService 创建账户服务
func NewAccountService() *AccountService {
	return &AccountService{}
}

// CreateAccount 创建账户
func (s *AccountService) CreateAccount(ctx context.Context, userID int64, name, accountType string, balance float64, currency string) (*model.Account, error) {
	if currency == "" {
		currency = "CNY"
	}

	now := time.Now()
	result, err := database.DB.Exec(
		"INSERT INTO accounts (user_id, name, type, balance, currency, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		userID, name, accountType, balance, currency, now, now,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &model.Account{
		ID:        id,
		UserID:    userID,
		Name:      name,
		Type:      accountType,
		Balance:   balance,
		Currency:  currency,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// GetAccount 获取账户
func (s *AccountService) GetAccount(ctx context.Context, userID, accountID int64) (*model.Account, error) {
	var account model.Account
	err := database.DB.QueryRow(
		"SELECT id, user_id, name, type, balance, currency, created_at, updated_at FROM accounts WHERE id = ? AND user_id = ?",
		accountID, userID,
	).Scan(&account.ID, &account.UserID, &account.Name, &account.Type, &account.Balance, &account.Currency, &account.CreatedAt, &account.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, errors.New("账户不存在")
	}
	if err != nil {
		return nil, err
	}

	return &account, nil
}

// UpdateAccount 更新账户
func (s *AccountService) UpdateAccount(ctx context.Context, userID, accountID int64, name, accountType string) (*model.Account, error) {
	now := time.Now()
	result, err := database.DB.Exec(
		"UPDATE accounts SET name = ?, type = ?, updated_at = ? WHERE id = ? AND user_id = ?",
		name, accountType, now, accountID, userID,
	)
	if err != nil {
		return nil, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, errors.New("账户不存在")
	}

	return s.GetAccount(ctx, userID, accountID)
}

// DeleteAccount 删除账户
func (s *AccountService) DeleteAccount(ctx context.Context, userID, accountID int64) error {
	// 检查账户是否有交易记录
	var count int
	err := database.DB.QueryRow("SELECT COUNT(*) FROM transactions WHERE account_id = ?", accountID).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("账户下有交易记录，无法删除")
	}

	result, err := database.DB.Exec("DELETE FROM accounts WHERE id = ? AND user_id = ?", accountID, userID)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("账户不存在")
	}

	return nil
}

// ListAccounts 获取账户列表
func (s *AccountService) ListAccounts(ctx context.Context, userID int64, page, pageSize int) ([]*model.Account, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 获取总数
	var total int
	err := database.DB.QueryRow("SELECT COUNT(*) FROM accounts WHERE user_id = ?", userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 获取列表
	offset := (page - 1) * pageSize
	rows, err := database.DB.Query(
		"SELECT id, user_id, name, type, balance, currency, created_at, updated_at FROM accounts WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?",
		userID, pageSize, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var accounts []*model.Account
	for rows.Next() {
		var account model.Account
		if err := rows.Scan(&account.ID, &account.UserID, &account.Name, &account.Type, &account.Balance, &account.Currency, &account.CreatedAt, &account.UpdatedAt); err != nil {
			return nil, 0, err
		}
		accounts = append(accounts, &account)
	}

	return accounts, total, nil
}

// UpdateBalance 更新账户余额
func (s *AccountService) UpdateBalance(ctx context.Context, accountID int64, amount float64) error {
	_, err := database.DB.Exec(
		"UPDATE accounts SET balance = balance + ?, updated_at = ? WHERE id = ?",
		amount, time.Now(), accountID,
	)
	return err
}
