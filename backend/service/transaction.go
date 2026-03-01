package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"finance-system/database"
	"finance-system/model"
)

// TransactionService 交易服务
type TransactionService struct {
	accountService *AccountService
}

// NewTransactionService 创建交易服务
func NewTransactionService(accountService *AccountService) *TransactionService {
	return &TransactionService{accountService: accountService}
}

// CreateTransaction 创建交易记录
func (s *TransactionService) CreateTransaction(ctx context.Context, userID int64, req *CreateTransactionReq) (*model.Transaction, error) {
	// 验证账户归属
	_, err := s.accountService.GetAccount(ctx, userID, req.AccountID)
	if err != nil {
		return nil, errors.New("账户不存在")
	}

	// 解析日期
	transDate, err := time.Parse("2006-01-02", req.TransactionDate)
	if err != nil {
		transDate = time.Now()
	}

	now := time.Now()
	result, err := database.DB.Exec(
		`INSERT INTO transactions (user_id, account_id, type, amount, category_id, description, transaction_date, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, req.AccountID, req.Type, req.Amount, req.CategoryID, req.Description, transDate, now, now,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	// 更新账户余额
	balanceChange := req.Amount
	if req.Type == "expense" {
		balanceChange = -req.Amount
	}
	if err := s.accountService.UpdateBalance(ctx, req.AccountID, balanceChange); err != nil {
		return nil, err
	}

	return s.GetTransaction(ctx, userID, id)
}

// CreateTransactionReq 创建交易请求
type CreateTransactionReq struct {
	AccountID       int64
	Type            string
	Amount          float64
	CategoryID      int64
	Description     string
	TransactionDate string
}

// GetTransaction 获取交易记录
func (s *TransactionService) GetTransaction(ctx context.Context, userID, transactionID int64) (*model.Transaction, error) {
	var trans model.Transaction
	var categoryName sql.NullString

	err := database.DB.QueryRow(
		`SELECT t.id, t.user_id, t.account_id, t.type, t.amount, t.category_id,
		COALESCE(c.name, '') as category_name, t.description, t.transaction_date, t.created_at
		FROM transactions t
		LEFT JOIN categories c ON t.category_id = c.id
		WHERE t.id = ? AND t.user_id = ?`,
		transactionID, userID,
	).Scan(&trans.ID, &trans.UserID, &trans.AccountID, &trans.Type, &trans.Amount,
		&trans.CategoryID, &categoryName, &trans.Description, &trans.TransactionDate, &trans.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, errors.New("交易记录不存在")
	}
	if err != nil {
		return nil, err
	}

	if categoryName.Valid {
		trans.CategoryName = categoryName.String
	}

	return &trans, nil
}

// UpdateTransaction 更新交易记录
func (s *TransactionService) UpdateTransaction(ctx context.Context, userID int64, req *UpdateTransactionReq) (*model.Transaction, error) {
	// 获取原交易记录
	oldTrans, err := s.GetTransaction(ctx, userID, req.ID)
	if err != nil {
		return nil, err
	}

	// 解析日期
	transDate, err := time.Parse("2006-01-02", req.TransactionDate)
	if err != nil {
		transDate = oldTrans.TransactionDate
	}

	now := time.Now()
	_, err = database.DB.Exec(
		`UPDATE transactions SET account_id = ?, type = ?, amount = ?, category_id = ?,
		description = ?, transaction_date = ?, updated_at = ? WHERE id = ? AND user_id = ?`,
		req.AccountID, req.Type, req.Amount, req.CategoryID, req.Description, transDate, now, req.ID, userID,
	)
	if err != nil {
		return nil, err
	}

	// 更新账户余额（恢复旧金额，应用新金额）
	oldAmount := oldTrans.Amount
	if oldTrans.Type == "expense" {
		oldAmount = -oldAmount
	}
	s.accountService.UpdateBalance(ctx, oldTrans.AccountID, -oldAmount)

	newAmount := req.Amount
	if req.Type == "expense" {
		newAmount = -newAmount
	}
	s.accountService.UpdateBalance(ctx, req.AccountID, newAmount)

	return s.GetTransaction(ctx, userID, req.ID)
}

// UpdateTransactionReq 更新交易请求
type UpdateTransactionReq struct {
	ID              int64
	AccountID       int64
	Type            string
	Amount          float64
	CategoryID      int64
	Description     string
	TransactionDate string
}

// DeleteTransaction 删除交易记录
func (s *TransactionService) DeleteTransaction(ctx context.Context, userID, transactionID int64) error {
	// 获取交易记录
	trans, err := s.GetTransaction(ctx, userID, transactionID)
	if err != nil {
		return err
	}

	// 删除记录
	result, err := database.DB.Exec("DELETE FROM transactions WHERE id = ? AND user_id = ?", transactionID, userID)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("交易记录不存在")
	}

	// 恢复账户余额
	amount := trans.Amount
	if trans.Type == "expense" {
		amount = -amount
	}
	return s.accountService.UpdateBalance(ctx, trans.AccountID, -amount)
}

// ListTransactions 获取交易记录列表
func (s *TransactionService) ListTransactions(ctx context.Context, userID int64, req *ListTransactionsReq) ([]*model.Transaction, int, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	// 构建查询条件
	where := "WHERE t.user_id = ?"
	args := []interface{}{userID}

	if req.AccountID > 0 {
		where += " AND t.account_id = ?"
		args = append(args, req.AccountID)
	}
	if req.Type != "" {
		where += " AND t.type = ?"
		args = append(args, req.Type)
	}
	if req.StartDate != "" {
		where += " AND t.transaction_date >= ?"
		args = append(args, req.StartDate)
	}
	if req.EndDate != "" {
		where += " AND t.transaction_date <= ?"
		args = append(args, req.EndDate)
	}

	// 获取总数
	var total int
	countQuery := "SELECT COUNT(*) FROM transactions t " + where
	err := database.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 获取列表
	offset := (req.Page - 1) * req.PageSize
	query := `SELECT t.id, t.user_id, t.account_id, t.type, t.amount, t.category_id,
		COALESCE(c.name, '') as category_name, t.description, t.transaction_date, t.created_at
		FROM transactions t
		LEFT JOIN categories c ON t.category_id = c.id ` + where +
		" ORDER BY t.transaction_date DESC, t.id DESC LIMIT ? OFFSET ?"
	args = append(args, req.PageSize, offset)

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var transactions []*model.Transaction
	for rows.Next() {
		var trans model.Transaction
		var categoryName sql.NullString
		if err := rows.Scan(&trans.ID, &trans.UserID, &trans.AccountID, &trans.Type, &trans.Amount,
			&trans.CategoryID, &categoryName, &trans.Description, &trans.TransactionDate, &trans.CreatedAt); err != nil {
			return nil, 0, err
		}
		if categoryName.Valid {
			trans.CategoryName = categoryName.String
		}
		transactions = append(transactions, &trans)
	}

	return transactions, total, nil
}

// ListTransactionsReq 交易列表请求
type ListTransactionsReq struct {
	AccountID int64
	Type      string
	StartDate string
	EndDate   string
	Page      int
	PageSize  int
}
