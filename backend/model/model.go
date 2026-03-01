package model

import "time"

// User 用户模型
type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Account 账户模型
type Account struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"` // cash, bank, credit, investment
	Balance   float64   `json:"balance"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Transaction 交易记录模型
type Transaction struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	AccountID       int64     `json:"account_id"`
	Type            string    `json:"type"` // income, expense, transfer
	Amount          float64   `json:"amount"`
	CategoryID      int64     `json:"category_id"`
	CategoryName    string    `json:"category_name"`
	Description     string    `json:"description"`
	TransactionDate time.Time `json:"transaction_date"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Category 分类模型
type Category struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"` // income, expense
	Icon      string    `json:"icon"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
}

// StatsSummary 统计摘要
type StatsSummary struct {
	TotalIncome  float64 `json:"total_income"`
	TotalExpense float64 `json:"total_expense"`
	Balance      float64 `json:"balance"`
}

// CategoryStats 分类统计
type CategoryStats struct {
	CategoryID   int64   `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Type         string  `json:"type"`
	Amount       float64 `json:"amount"`
	Percentage   float64 `json:"percentage"`
}

// DailyStats 每日统计
type DailyStats struct {
	Date    string  `json:"date"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
}
