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
		// 菜品表
		`CREATE TABLE IF NOT EXISTS dishes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			description TEXT DEFAULT '',
			price REAL NOT NULL,
			image TEXT DEFAULT '',
			category TEXT DEFAULT '',
			dietary_tags TEXT DEFAULT '',
			stock INTEGER DEFAULT -1,
			status TEXT DEFAULT 'available',
			sort_order INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		// 餐桌表
		`CREATE TABLE IF NOT EXISTS tables (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			table_no TEXT NOT NULL,
			qr_code_token TEXT UNIQUE NOT NULL,
			status TEXT DEFAULT 'active',
			capacity INTEGER DEFAULT 4,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			UNIQUE(user_id, table_no)
		)`,
		// 订单表
		`CREATE TABLE IF NOT EXISTS orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			table_id INTEGER,
			table_no TEXT DEFAULT '',
			order_no TEXT UNIQUE NOT NULL,
			total_price REAL NOT NULL,
			status TEXT DEFAULT 'pending',
			order_source TEXT DEFAULT 'admin',
			customer_name TEXT DEFAULT '',
			remark TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (table_id) REFERENCES tables(id)
		)`,
		// 订单项表
		`CREATE TABLE IF NOT EXISTS order_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_id INTEGER NOT NULL,
			dish_id INTEGER NOT NULL,
			dish_name TEXT NOT NULL,
			dish_image TEXT DEFAULT '',
			price REAL NOT NULL,
			quantity INTEGER NOT NULL,
			remark TEXT DEFAULT '',
			FOREIGN KEY (order_id) REFERENCES orders(id),
			FOREIGN KEY (dish_id) REFERENCES dishes(id)
		)`,
		// 菜品变化记录表
		`CREATE TABLE IF NOT EXISTS dish_change_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			dish_id INTEGER NOT NULL,
			type TEXT NOT NULL,
			old_value REAL NOT NULL,
			new_value REAL NOT NULL,
			remark TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (dish_id) REFERENCES dishes(id)
		)`,
		// 创建索引
		`CREATE INDEX IF NOT EXISTS idx_accounts_user_id ON accounts(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_account_id ON transactions(account_id)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_date ON transactions(transaction_date)`,
		`CREATE INDEX IF NOT EXISTS idx_categories_user_id ON categories(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_dishes_user_id ON dishes(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_dishes_category ON dishes(category)`,
		`CREATE INDEX IF NOT EXISTS idx_tables_user_id ON tables(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tables_qr_code_token ON tables(qr_code_token)`,
		`CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status)`,
		`CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id)`,
		`CREATE INDEX IF NOT EXISTS idx_dish_change_logs_dish_id ON dish_change_logs(dish_id)`,
	}

	for _, query := range queries {
		if _, err := DB.Exec(query); err != nil {
			return err
		}
	}

	log.Println("数据库表创建成功")

	// 执行数据库迁移（添加新字段到已有表）
	if err := migrateDatabase(); err != nil {
		log.Printf("数据库迁移警告: %v", err)
	}

	// 迁移后创建依赖新字段的索引
	postMigrationIndexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_orders_table_id ON orders(table_id)`,
		`CREATE INDEX IF NOT EXISTS idx_orders_order_source ON orders(order_source)`,
	}
	for _, query := range postMigrationIndexes {
		if _, err := DB.Exec(query); err != nil {
			log.Printf("创建索引警告: %v", err)
		}
	}

	return initDefaultCategories()
}

// migrateDatabase 数据库迁移，添加新字段到已有表
func migrateDatabase() error {
	// 检查并添加 orders 表的新字段
	migrations := []struct {
		table  string
		column string
		ddl    string
	}{
		{"orders", "table_id", "ALTER TABLE orders ADD COLUMN table_id INTEGER"},
		{"orders", "table_no", "ALTER TABLE orders ADD COLUMN table_no TEXT DEFAULT ''"},
		{"orders", "order_source", "ALTER TABLE orders ADD COLUMN order_source TEXT DEFAULT 'admin'"},
		{"orders", "customer_name", "ALTER TABLE orders ADD COLUMN customer_name TEXT DEFAULT ''"},
	}

	for _, m := range migrations {
		// 检查列是否存在
		var count int
		err := DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, m.table, m.column).Scan(&count)
		if err != nil {
			continue
		}
		if count == 0 {
			// 列不存在，添加它
			if _, err := DB.Exec(m.ddl); err != nil {
				log.Printf("迁移 %s.%s 失败: %v", m.table, m.column, err)
			} else {
				log.Printf("迁移成功: 添加 %s.%s", m.table, m.column)
			}
		}
	}

	return nil
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
