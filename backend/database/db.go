package database

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

// InitDB 初始化数据库连接
func InitDB(dataSourceName string) error {
	var err error
	DB, err = sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return err
	}

	if err = DB.Ping(); err != nil {
		return err
	}

	log.Println("数据库连接成功")
	return createTables()
}

// createTables 创建数据库表
func createTables() error {
	queries := []string{
		// 用户表
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		// 账户表
		`CREATE TABLE IF NOT EXISTS accounts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			balance REAL DEFAULT 0,
			currency TEXT DEFAULT 'CNY',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		// 分类表
		`CREATE TABLE IF NOT EXISTS categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			icon TEXT DEFAULT '',
			color TEXT DEFAULT '#1890ff',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		// 交易记录表
		`CREATE TABLE IF NOT EXISTS transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			account_id INTEGER NOT NULL,
			type TEXT NOT NULL,
			amount REAL NOT NULL,
			category_id INTEGER,
			description TEXT DEFAULT '',
			transaction_date DATE NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (account_id) REFERENCES accounts(id),
			FOREIGN KEY (category_id) REFERENCES categories(id)
		)`,
		// 创建索引
		`CREATE INDEX IF NOT EXISTS idx_accounts_user_id ON accounts(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_account_id ON transactions(account_id)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_date ON transactions(transaction_date)`,
		`CREATE INDEX IF NOT EXISTS idx_categories_user_id ON categories(user_id)`,
	}

	for _, query := range queries {
		if _, err := DB.Exec(query); err != nil {
			return err
		}
	}

	log.Println("数据库表创建成功")
	return initDefaultCategories()
}

// initDefaultCategories 初始化默认分类
func initDefaultCategories() error {
	// 检查是否已有默认分类
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM categories WHERE user_id = 0").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	// 默认支出分类
	expenseCategories := []struct {
		name  string
		icon  string
		color string
	}{
		{"餐饮", "restaurant", "#f5222d"},
		{"交通", "car", "#fa541c"},
		{"购物", "shopping-cart", "#fa8c16"},
		{"娱乐", "play-circle", "#faad14"},
		{"居住", "home", "#a0d911"},
		{"医疗", "medicine-box", "#52c41a"},
		{"教育", "book", "#13c2c2"},
		{"其他支出", "ellipsis", "#1890ff"},
	}

	// 默认收入分类
	incomeCategories := []struct {
		name  string
		icon  string
		color string
	}{
		{"工资", "dollar", "#52c41a"},
		{"奖金", "gift", "#faad14"},
		{"投资收益", "stock", "#1890ff"},
		{"兼职", "tool", "#722ed1"},
		{"其他收入", "ellipsis", "#13c2c2"},
	}

	stmt, err := DB.Prepare("INSERT INTO categories (user_id, name, type, icon, color, created_at) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, c := range expenseCategories {
		if _, err := stmt.Exec(0, c.name, "expense", c.icon, c.color, now); err != nil {
			return err
		}
	}

	for _, c := range incomeCategories {
		if _, err := stmt.Exec(0, c.name, "income", c.icon, c.color, now); err != nil {
			return err
		}
	}

	log.Println("默认分类初始化成功")
	return nil
}

// Close 关闭数据库连接
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
