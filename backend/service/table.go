package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"finance-system/database"
	"finance-system/model"
)

// TableService 餐桌服务
type TableService struct{}

// NewTableService 创建餐桌服务
func NewTableService() *TableService {
	return &TableService{}
}

// generateTableToken 生成餐桌二维码令牌
func generateTableToken(tableID, userID int64) string {
	timestamp := time.Now().Unix()
	data := fmt.Sprintf("%d_%d_%d", tableID, timestamp, userID)

	h := hmac.New(sha256.New, jwtSecret)
	h.Write([]byte(data))
	signature := hex.EncodeToString(h.Sum(nil))[:16] // 取前16位

	token := fmt.Sprintf("%d_%d_%s", tableID, timestamp, signature)
	return base64.URLEncoding.EncodeToString([]byte(token))
}

// ValidateTableToken 验证餐桌令牌并返回餐桌信息
func (s *TableService) ValidateTableToken(token string) (*model.Table, error) {
	// 1. Base64解码
	decoded, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return nil, errors.New("无效的二维码")
	}

	// 2. 解析token组成
	parts := strings.Split(string(decoded), "_")
	if len(parts) != 3 {
		return nil, errors.New("无效的二维码格式")
	}

	tableID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, errors.New("无效的餐桌ID")
	}

	// 3. 查询餐桌
	table, err := s.GetTableByID(tableID)
	if err != nil {
		return nil, errors.New("餐桌不存在")
	}

	if table.Status != "active" {
		return nil, errors.New("餐桌已禁用")
	}

	// 4. 验证token是否匹配
	if table.QRCodeToken != token {
		return nil, errors.New("二维码已失效，请扫描新的二维码")
	}

	return table, nil
}

// GetTableByID 根据ID获取餐桌
func (s *TableService) GetTableByID(tableID int64) (*model.Table, error) {
	var table model.Table
	err := database.DB.QueryRow(`
		SELECT id, user_id, table_no, qr_code_token, status, capacity, created_at, updated_at
		FROM tables WHERE id = ?
	`, tableID).Scan(
		&table.ID, &table.UserID, &table.TableNo, &table.QRCodeToken,
		&table.Status, &table.Capacity, &table.CreatedAt, &table.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("餐桌不存在")
	}
	if err != nil {
		return nil, err
	}

	return &table, nil
}

// CreateTable 创建餐桌
func (s *TableService) CreateTable(ctx context.Context, userID int64, tableNo string, capacity int) (*model.Table, error) {
	if tableNo == "" {
		return nil, errors.New("桌号不能为空")
	}
	if capacity <= 0 {
		capacity = 4
	}

	// 检查桌号是否已存在
	var existingID int64
	err := database.DB.QueryRow(
		"SELECT id FROM tables WHERE user_id = ? AND table_no = ?",
		userID, tableNo,
	).Scan(&existingID)
	if err == nil {
		return nil, errors.New("桌号已存在")
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	now := time.Now()

	// 先插入获取ID，再生成token
	result, err := database.DB.Exec(`
		INSERT INTO tables (user_id, table_no, qr_code_token, status, capacity, created_at, updated_at)
		VALUES (?, ?, ?, 'active', ?, ?, ?)
	`, userID, tableNo, "temp", capacity, now, now)
	if err != nil {
		return nil, err
	}

	tableID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	// 生成真实token并更新
	token := generateTableToken(tableID, userID)
	_, err = database.DB.Exec("UPDATE tables SET qr_code_token = ? WHERE id = ?", token, tableID)
	if err != nil {
		return nil, err
	}

	return &model.Table{
		ID:          tableID,
		UserID:      userID,
		TableNo:     tableNo,
		QRCodeToken: token,
		Status:      "active",
		Capacity:    capacity,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// UpdateTable 更新餐桌信息
func (s *TableService) UpdateTable(ctx context.Context, userID, tableID int64, tableNo string, capacity int, status string) (*model.Table, error) {
	// 检查餐桌是否存在且属于当前用户
	table, err := s.GetTableByID(tableID)
	if err != nil {
		return nil, err
	}
	if table.UserID != userID {
		return nil, errors.New("无权操作此餐桌")
	}

	// 如果桌号变更，检查是否冲突
	if tableNo != "" && tableNo != table.TableNo {
		var existingID int64
		err := database.DB.QueryRow(
			"SELECT id FROM tables WHERE user_id = ? AND table_no = ? AND id != ?",
			userID, tableNo, tableID,
		).Scan(&existingID)
		if err == nil {
			return nil, errors.New("桌号已存在")
		}
		if err != sql.ErrNoRows {
			return nil, err
		}
		table.TableNo = tableNo
	}

	if capacity > 0 {
		table.Capacity = capacity
	}
	if status == "active" || status == "disabled" {
		table.Status = status
	}

	now := time.Now()
	_, err = database.DB.Exec(`
		UPDATE tables SET table_no = ?, capacity = ?, status = ?, updated_at = ?
		WHERE id = ?
	`, table.TableNo, table.Capacity, table.Status, now, tableID)
	if err != nil {
		return nil, err
	}

	table.UpdatedAt = now
	return table, nil
}

// DeleteTable 删除餐桌
func (s *TableService) DeleteTable(ctx context.Context, userID, tableID int64) error {
	// 检查餐桌是否存在且属于当前用户
	table, err := s.GetTableByID(tableID)
	if err != nil {
		return err
	}
	if table.UserID != userID {
		return errors.New("无权操作此餐桌")
	}

	// 检查是否有未完成的订单
	var orderCount int
	err = database.DB.QueryRow(`
		SELECT COUNT(*) FROM orders
		WHERE table_id = ? AND status NOT IN ('completed', 'cancelled')
	`, tableID).Scan(&orderCount)
	if err != nil {
		return err
	}
	if orderCount > 0 {
		return errors.New("该餐桌有未完成的订单，无法删除")
	}

	_, err = database.DB.Exec("DELETE FROM tables WHERE id = ?", tableID)
	return err
}

// ListTables 获取餐桌列表
func (s *TableService) ListTables(ctx context.Context, userID int64, status string) ([]*model.Table, error) {
	query := "SELECT id, user_id, table_no, qr_code_token, status, capacity, created_at, updated_at FROM tables WHERE user_id = ?"
	args := []interface{}{userID}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	query += " ORDER BY table_no ASC"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []*model.Table
	for rows.Next() {
		var t model.Table
		err := rows.Scan(
			&t.ID, &t.UserID, &t.TableNo, &t.QRCodeToken,
			&t.Status, &t.Capacity, &t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		tables = append(tables, &t)
	}

	return tables, nil
}

// RegenerateToken 重新生成二维码令牌
func (s *TableService) RegenerateToken(ctx context.Context, userID, tableID int64) (string, error) {
	// 检查餐桌是否存在且属于当前用户
	table, err := s.GetTableByID(tableID)
	if err != nil {
		return "", err
	}
	if table.UserID != userID {
		return "", errors.New("无权操作此餐桌")
	}

	// 生成新token
	token := generateTableToken(tableID, userID)
	now := time.Now()

	_, err = database.DB.Exec(
		"UPDATE tables SET qr_code_token = ?, updated_at = ? WHERE id = ?",
		token, now, tableID,
	)
	if err != nil {
		return "", err
	}

	return token, nil
}

// GetTableToken 获取餐桌的token（用于生成二维码）
func (s *TableService) GetTableToken(ctx context.Context, userID, tableID int64) (string, error) {
	var token string
	var tableUserID int64
	err := database.DB.QueryRow(
		"SELECT user_id, qr_code_token FROM tables WHERE id = ?",
		tableID,
	).Scan(&tableUserID, &token)

	if err == sql.ErrNoRows {
		return "", errors.New("餐桌不存在")
	}
	if err != nil {
		return "", err
	}
	if tableUserID != userID {
		return "", errors.New("无权操作此餐桌")
	}

	return token, nil
}
