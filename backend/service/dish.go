package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"finance-system/database"
	"finance-system/model"
)

// DishService 菜品服务
type DishService struct{}

// NewDishService 创建菜品服务
func NewDishService() *DishService {
	return &DishService{}
}

// CreateDish 创建菜品
func (s *DishService) CreateDish(ctx context.Context, userID int64, req *CreateDishReq) (*model.Dish, error) {
	now := time.Now()

	result, err := database.DB.Exec(`
		INSERT INTO dishes (user_id, name, description, price, image, category, dietary_tags, stock, status, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, req.Name, req.Description, req.Price, req.Image, req.Category, req.DietaryTags, req.Stock, "available", req.SortOrder, now, now)

	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &model.Dish{
		ID:          id,
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Image:       req.Image,
		Category:    req.Category,
		DietaryTags: req.DietaryTags,
		Stock:       req.Stock,
		Status:      "available",
		SortOrder:   req.SortOrder,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// CreateDishReq 创建菜品请求
type CreateDishReq struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Image       string  `json:"image"`
	Category    string  `json:"category"`
	DietaryTags string  `json:"dietary_tags"`
	Stock       int     `json:"stock"`
	SortOrder   int     `json:"sort_order"`
}

// UpdateDish 更新菜品
func (s *DishService) UpdateDish(ctx context.Context, userID, dishID int64, req *UpdateDishReq) (*model.Dish, error) {
	// 获取旧的菜品信息
	var oldPrice float64
	var oldStock int
	var existingUserID int64
	err := database.DB.QueryRow("SELECT user_id, price, stock FROM dishes WHERE id = ?", dishID).Scan(&existingUserID, &oldPrice, &oldStock)
	if err == sql.ErrNoRows {
		return nil, errors.New("菜品不存在")
	}
	if err != nil {
		return nil, err
	}
	if existingUserID != userID {
		return nil, errors.New("无权修改此菜品")
	}

	now := time.Now()
	nowStr := now.Format("2006-01-02 15:04:05")

	// 记录价格变化
	if req.Price != oldPrice {
		database.DB.Exec(`
			INSERT INTO dish_change_logs (dish_id, type, old_value, new_value, remark, created_at)
			VALUES (?, 'price', ?, ?, '手动调整', ?)
		`, dishID, oldPrice, req.Price, nowStr)
	}

	// 记录库存变化
	if req.Stock != oldStock {
		database.DB.Exec(`
			INSERT INTO dish_change_logs (dish_id, type, old_value, new_value, remark, created_at)
			VALUES (?, 'stock', ?, ?, '手动调整', ?)
		`, dishID, float64(oldStock), float64(req.Stock), nowStr)
	}

	_, err = database.DB.Exec(`
		UPDATE dishes SET name = ?, description = ?, price = ?, image = ?, category = ?, dietary_tags = ?, stock = ?, status = ?, sort_order = ?, updated_at = ?
		WHERE id = ?
	`, req.Name, req.Description, req.Price, req.Image, req.Category, req.DietaryTags, req.Stock, req.Status, req.SortOrder, now, dishID)

	if err != nil {
		return nil, err
	}

	return s.GetDish(ctx, userID, dishID)
}

// UpdateDishReq 更新菜品请求
type UpdateDishReq struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Image       string  `json:"image"`
	Category    string  `json:"category"`
	DietaryTags string  `json:"dietary_tags"`
	Stock       int     `json:"stock"`
	Status      string  `json:"status"`
	SortOrder   int     `json:"sort_order"`
}

// GetDish 获取单个菜品
func (s *DishService) GetDish(ctx context.Context, userID, dishID int64) (*model.Dish, error) {
	var dish model.Dish
	err := database.DB.QueryRow(`
		SELECT id, user_id, name, description, price, image, category, dietary_tags, stock, status, sort_order, created_at, updated_at
		FROM dishes WHERE id = ? AND user_id = ?
	`, dishID, userID).Scan(
		&dish.ID, &dish.UserID, &dish.Name, &dish.Description, &dish.Price, &dish.Image,
		&dish.Category, &dish.DietaryTags, &dish.Stock, &dish.Status, &dish.SortOrder,
		&dish.CreatedAt, &dish.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("菜品不存在")
	}
	if err != nil {
		return nil, err
	}

	return &dish, nil
}

// ListDishes 获取菜品列表
func (s *DishService) ListDishes(ctx context.Context, userID int64, category, status string, page, pageSize int) ([]*model.Dish, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 构建查询条件
	query := "SELECT id, user_id, name, description, price, image, category, dietary_tags, stock, status, sort_order, created_at, updated_at FROM dishes WHERE user_id = ?"
	countQuery := "SELECT COUNT(*) FROM dishes WHERE user_id = ?"
	args := []interface{}{userID}
	countArgs := []interface{}{userID}

	if category != "" {
		query += " AND category = ?"
		countQuery += " AND category = ?"
		args = append(args, category)
		countArgs = append(countArgs, category)
	}

	if status != "" {
		query += " AND status = ?"
		countQuery += " AND status = ?"
		args = append(args, status)
		countArgs = append(countArgs, status)
	}

	// 获取总数
	var total int64
	err := database.DB.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	query += " ORDER BY sort_order ASC, created_at DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var dishes []*model.Dish
	for rows.Next() {
		var dish model.Dish
		err := rows.Scan(
			&dish.ID, &dish.UserID, &dish.Name, &dish.Description, &dish.Price, &dish.Image,
			&dish.Category, &dish.DietaryTags, &dish.Stock, &dish.Status, &dish.SortOrder,
			&dish.CreatedAt, &dish.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		dishes = append(dishes, &dish)
	}

	return dishes, total, nil
}

// DeleteDish 删除菜品
func (s *DishService) DeleteDish(ctx context.Context, userID, dishID int64) error {
	result, err := database.DB.Exec("DELETE FROM dishes WHERE id = ? AND user_id = ?", dishID, userID)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("菜品不存在或无权删除")
	}

	return nil
}

// GetDishCategories 获取菜品分类列表
func (s *DishService) GetDishCategories(ctx context.Context, userID int64) ([]string, error) {
	rows, err := database.DB.Query("SELECT DISTINCT category FROM dishes WHERE user_id = ? AND category != '' ORDER BY category", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}

	return categories, nil
}

// UpdateDishStock 更新菜品库存
func (s *DishService) UpdateDishStock(ctx context.Context, userID, dishID int64, stock int) error {
	result, err := database.DB.Exec("UPDATE dishes SET stock = ?, updated_at = ? WHERE id = ? AND user_id = ?",
		stock, time.Now(), dishID, userID)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("菜品不存在或无权修改")
	}

	return nil
}

// GetDishChangeLogs 获取菜品变化记录
func (s *DishService) GetDishChangeLogs(ctx context.Context, userID, dishID int64) ([]*model.DishChangeLog, error) {
	// 先检查菜品是否属于该用户
	var existingUserID int64
	err := database.DB.QueryRow("SELECT user_id FROM dishes WHERE id = ?", dishID).Scan(&existingUserID)
	if err == sql.ErrNoRows {
		return nil, errors.New("菜品不存在")
	}
	if err != nil {
		return nil, err
	}
	if existingUserID != userID {
		return nil, errors.New("无权查看此菜品")
	}

	rows, err := database.DB.Query(`
		SELECT id, dish_id, type, old_value, new_value, remark, created_at
		FROM dish_change_logs WHERE dish_id = ? ORDER BY created_at DESC LIMIT 50
	`, dishID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*model.DishChangeLog
	for rows.Next() {
		var log model.DishChangeLog
		if err := rows.Scan(&log.ID, &log.DishID, &log.Type, &log.OldValue, &log.NewValue, &log.Remark, &log.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, &log)
	}

	return logs, nil
}

// LogDishStockChange 记录库存变化（供订单服务调用）
func (s *DishService) LogDishStockChange(dishID int64, oldStock, newStock int, remark string) {
	nowStr := time.Now().Format("2006-01-02 15:04:05")
	database.DB.Exec(`
		INSERT INTO dish_change_logs (dish_id, type, old_value, new_value, remark, created_at)
		VALUES (?, 'stock', ?, ?, ?, ?)
	`, dishID, float64(oldStock), float64(newStock), remark, nowStr)
}
