package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"finance-system/database"
	"finance-system/model"
)

// CategoryService 分类服务
type CategoryService struct{}

// NewCategoryService 创建分类服务
func NewCategoryService() *CategoryService {
	return &CategoryService{}
}

// CreateCategory 创建分类
func (s *CategoryService) CreateCategory(ctx context.Context, userID int64, name, categoryType, icon, color string) (*model.Category, error) {
	if icon == "" {
		icon = "tag"
	}
	if color == "" {
		color = "#1890ff"
	}

	now := time.Now()
	result, err := database.DB.Exec(
		"INSERT INTO categories (user_id, name, type, icon, color, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		userID, name, categoryType, icon, color, now,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &model.Category{
		ID:        id,
		UserID:    userID,
		Name:      name,
		Type:      categoryType,
		Icon:      icon,
		Color:     color,
		CreatedAt: now,
	}, nil
}

// GetCategory 获取分类
func (s *CategoryService) GetCategory(ctx context.Context, userID, categoryID int64) (*model.Category, error) {
	var category model.Category
	err := database.DB.QueryRow(
		"SELECT id, user_id, name, type, icon, color, created_at FROM categories WHERE id = ? AND (user_id = ? OR user_id = 0)",
		categoryID, userID,
	).Scan(&category.ID, &category.UserID, &category.Name, &category.Type, &category.Icon, &category.Color, &category.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, errors.New("分类不存在")
	}
	if err != nil {
		return nil, err
	}

	return &category, nil
}

// UpdateCategory 更新分类
func (s *CategoryService) UpdateCategory(ctx context.Context, userID, categoryID int64, name, icon, color string) (*model.Category, error) {
	// 只能更新用户自己创建的分类
	result, err := database.DB.Exec(
		"UPDATE categories SET name = ?, icon = ?, color = ? WHERE id = ? AND user_id = ?",
		name, icon, color, categoryID, userID,
	)
	if err != nil {
		return nil, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, errors.New("分类不存在或无权修改")
	}

	return s.GetCategory(ctx, userID, categoryID)
}

// DeleteCategory 删除分类
func (s *CategoryService) DeleteCategory(ctx context.Context, userID, categoryID int64) error {
	// 检查是否有交易记录使用此分类
	var count int
	err := database.DB.QueryRow("SELECT COUNT(*) FROM transactions WHERE category_id = ?", categoryID).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该分类下有交易记录，无法删除")
	}

	// 只能删除用户自己创建的分类
	result, err := database.DB.Exec("DELETE FROM categories WHERE id = ? AND user_id = ?", categoryID, userID)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("分类不存在或无权删除")
	}

	return nil
}

// ListCategories 获取分类列表
func (s *CategoryService) ListCategories(ctx context.Context, userID int64, categoryType string) ([]*model.Category, error) {
	query := "SELECT id, user_id, name, type, icon, color, created_at FROM categories WHERE (user_id = ? OR user_id = 0)"
	args := []interface{}{userID}

	if categoryType != "" {
		query += " AND type = ?"
		args = append(args, categoryType)
	}

	query += " ORDER BY user_id DESC, id ASC"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []*model.Category
	for rows.Next() {
		var category model.Category
		if err := rows.Scan(&category.ID, &category.UserID, &category.Name, &category.Type, &category.Icon, &category.Color, &category.CreatedAt); err != nil {
			return nil, err
		}
		categories = append(categories, &category)
	}

	return categories, nil
}
