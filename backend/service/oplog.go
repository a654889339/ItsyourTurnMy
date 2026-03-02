package service

import (
	"encoding/json"
	"time"

	"finance-system/database"
	"finance-system/model"
)

// OperationLogService 操作日志服务
type OperationLogService struct{}

// NewOperationLogService 创建操作日志服务
func NewOperationLogService() *OperationLogService {
	return &OperationLogService{}
}

// 模块常量
const (
	ModuleDish        = "dish"        // 菜品
	ModuleOrder       = "order"       // 订单
	ModuleTable       = "table"       // 餐桌
	ModuleAccount     = "account"     // 账户
	ModuleTransaction = "transaction" // 交易
	ModuleCategory    = "category"    // 分类
	ModuleUser        = "user"        // 用户
)

// 操作常量
const (
	ActionCreate = "create" // 创建
	ActionUpdate = "update" // 更新
	ActionDelete = "delete" // 删除
)

// LogEntry 日志记录参数
type LogEntry struct {
	UserID      int64
	Username    string
	Module      string
	Action      string
	TargetID    int64
	TargetName  string
	Description string
	OldValue    interface{}
	NewValue    interface{}
	IP          string
}

// Log 记录操作日志
func (s *OperationLogService) Log(entry LogEntry) error {
	oldValueJSON := ""
	newValueJSON := ""

	if entry.OldValue != nil {
		if data, err := json.Marshal(entry.OldValue); err == nil {
			oldValueJSON = string(data)
		}
	}
	if entry.NewValue != nil {
		if data, err := json.Marshal(entry.NewValue); err == nil {
			newValueJSON = string(data)
		}
	}

	_, err := database.DB.Exec(`
		INSERT INTO operation_logs (user_id, username, module, action, target_id, target_name, description, old_value, new_value, ip, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, entry.UserID, entry.Username, entry.Module, entry.Action, entry.TargetID, entry.TargetName, entry.Description, oldValueJSON, newValueJSON, entry.IP, time.Now())

	return err
}

// ListLogsReq 日志列表请求
type ListLogsReq struct {
	Module    string // 模块筛选
	Action    string // 操作筛选
	StartDate string // 开始日期 (YYYY-MM-DD)
	EndDate   string // 结束日期 (YYYY-MM-DD)
	Keyword   string // 关键词搜索
	Page      int
	PageSize  int
}

// ListLogs 查询操作日志列表
func (s *OperationLogService) ListLogs(userID int64, req *ListLogsReq) ([]*model.OperationLog, int64, error) {
	query := `SELECT id, user_id, username, module, action, target_id, target_name, description, old_value, new_value, ip, created_at FROM operation_logs WHERE user_id = ?`
	countQuery := `SELECT COUNT(*) FROM operation_logs WHERE user_id = ?`
	args := []interface{}{userID}
	countArgs := []interface{}{userID}

	// 模块筛选
	if req.Module != "" {
		query += " AND module = ?"
		countQuery += " AND module = ?"
		args = append(args, req.Module)
		countArgs = append(countArgs, req.Module)
	}

	// 操作筛选
	if req.Action != "" {
		query += " AND action = ?"
		countQuery += " AND action = ?"
		args = append(args, req.Action)
		countArgs = append(countArgs, req.Action)
	}

	// 日期筛选
	if req.StartDate != "" {
		query += " AND created_at >= ?"
		countQuery += " AND created_at >= ?"
		args = append(args, req.StartDate+" 00:00:00")
		countArgs = append(countArgs, req.StartDate+" 00:00:00")
	}
	if req.EndDate != "" {
		query += " AND created_at <= ?"
		countQuery += " AND created_at <= ?"
		args = append(args, req.EndDate+" 23:59:59")
		countArgs = append(countArgs, req.EndDate+" 23:59:59")
	}

	// 关键词搜索
	if req.Keyword != "" {
		keyword := "%" + req.Keyword + "%"
		query += " AND (target_name LIKE ? OR description LIKE ?)"
		countQuery += " AND (target_name LIKE ? OR description LIKE ?)"
		args = append(args, keyword, keyword)
		countArgs = append(countArgs, keyword, keyword)
	}

	// 获取总数
	var total int64
	if err := database.DB.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// 分页
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}
	offset := (req.Page - 1) * req.PageSize

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, req.PageSize, offset)

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*model.OperationLog
	for rows.Next() {
		var log model.OperationLog
		err := rows.Scan(
			&log.ID, &log.UserID, &log.Username, &log.Module, &log.Action,
			&log.TargetID, &log.TargetName, &log.Description,
			&log.OldValue, &log.NewValue, &log.IP, &log.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		logs = append(logs, &log)
	}

	return logs, total, nil
}

// GetModuleText 获取模块中文名
func GetModuleText(module string) string {
	switch module {
	case ModuleDish:
		return "菜品管理"
	case ModuleOrder:
		return "订单管理"
	case ModuleTable:
		return "餐桌管理"
	case ModuleAccount:
		return "账户管理"
	case ModuleTransaction:
		return "收支记录"
	case ModuleCategory:
		return "分类管理"
	case ModuleUser:
		return "用户管理"
	default:
		return module
	}
}

// GetActionText 获取操作中文名
func GetActionText(action string) string {
	switch action {
	case ActionCreate:
		return "新增"
	case ActionUpdate:
		return "修改"
	case ActionDelete:
		return "删除"
	default:
		return action
	}
}
