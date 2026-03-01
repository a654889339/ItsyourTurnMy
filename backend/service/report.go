package service

import (
	"context"
	"fmt"
	"time"

	"finance-system/database"
	"finance-system/model"
)

// ReportService 报表服务
type ReportService struct{}

// NewReportService 创建报表服务
func NewReportService() *ReportService {
	return &ReportService{}
}

// GetStats 获取统计数据
func (s *ReportService) GetStats(ctx context.Context, userID int64, startDate, endDate string) (*model.StatsSummary, []*model.CategoryStats, []*model.DailyStats, error) {
	// 获取收支汇总
	summary := &model.StatsSummary{}

	query := `SELECT type, SUM(amount) as total FROM transactions WHERE user_id = ?`
	args := []interface{}{userID}

	if startDate != "" {
		query += " AND transaction_date >= ?"
		args = append(args, startDate)
	}
	if endDate != "" {
		query += " AND transaction_date <= ?"
		args = append(args, endDate)
	}
	query += " GROUP BY type"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var transType string
		var total float64
		if err := rows.Scan(&transType, &total); err != nil {
			return nil, nil, nil, err
		}
		if transType == "income" {
			summary.TotalIncome = total
		} else if transType == "expense" {
			summary.TotalExpense = total
		}
	}
	summary.Balance = summary.TotalIncome - summary.TotalExpense

	// 获取分类统计
	categoryStats, err := s.getCategoryStats(ctx, userID, startDate, endDate)
	if err != nil {
		return nil, nil, nil, err
	}

	// 获取每日统计
	dailyStats, err := s.getDailyStats(ctx, userID, startDate, endDate)
	if err != nil {
		return nil, nil, nil, err
	}

	return summary, categoryStats, dailyStats, nil
}

// getCategoryStats 获取分类统计
func (s *ReportService) getCategoryStats(ctx context.Context, userID int64, startDate, endDate string) ([]*model.CategoryStats, error) {
	query := `SELECT t.category_id, COALESCE(c.name, '未分类') as category_name, t.type, SUM(t.amount) as total
		FROM transactions t
		LEFT JOIN categories c ON t.category_id = c.id
		WHERE t.user_id = ?`
	args := []interface{}{userID}

	if startDate != "" {
		query += " AND t.transaction_date >= ?"
		args = append(args, startDate)
	}
	if endDate != "" {
		query += " AND t.transaction_date <= ?"
		args = append(args, endDate)
	}
	query += " GROUP BY t.category_id, c.name, t.type ORDER BY total DESC"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*model.CategoryStats
	var totalIncome, totalExpense float64

	// 先收集数据
	type tempStat struct {
		CategoryID   int64
		CategoryName string
		Type         string
		Amount       float64
	}
	var tempStats []tempStat

	for rows.Next() {
		var ts tempStat
		if err := rows.Scan(&ts.CategoryID, &ts.CategoryName, &ts.Type, &ts.Amount); err != nil {
			return nil, err
		}
		tempStats = append(tempStats, ts)
		if ts.Type == "income" {
			totalIncome += ts.Amount
		} else {
			totalExpense += ts.Amount
		}
	}

	// 计算百分比
	for _, ts := range tempStats {
		stat := &model.CategoryStats{
			CategoryID:   ts.CategoryID,
			CategoryName: ts.CategoryName,
			Type:         ts.Type,
			Amount:       ts.Amount,
		}
		if ts.Type == "income" && totalIncome > 0 {
			stat.Percentage = ts.Amount / totalIncome * 100
		} else if ts.Type == "expense" && totalExpense > 0 {
			stat.Percentage = ts.Amount / totalExpense * 100
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

// getDailyStats 获取每日统计
func (s *ReportService) getDailyStats(ctx context.Context, userID int64, startDate, endDate string) ([]*model.DailyStats, error) {
	query := `SELECT DATE(transaction_date) as date, type, SUM(amount) as total
		FROM transactions WHERE user_id = ?`
	args := []interface{}{userID}

	if startDate != "" {
		query += " AND transaction_date >= ?"
		args = append(args, startDate)
	}
	if endDate != "" {
		query += " AND transaction_date <= ?"
		args = append(args, endDate)
	}
	query += " GROUP BY DATE(transaction_date), type ORDER BY date"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 用map聚合每日数据
	dailyMap := make(map[string]*model.DailyStats)

	for rows.Next() {
		var date, transType string
		var total float64
		if err := rows.Scan(&date, &transType, &total); err != nil {
			return nil, err
		}

		if _, ok := dailyMap[date]; !ok {
			dailyMap[date] = &model.DailyStats{Date: date}
		}

		if transType == "income" {
			dailyMap[date].Income = total
		} else {
			dailyMap[date].Expense = total
		}
	}

	// 转换为切片并排序
	var stats []*model.DailyStats
	for _, s := range dailyMap {
		stats = append(stats, s)
	}

	return stats, nil
}

// GetMonthlyReport 获取月度报表
func (s *ReportService) GetMonthlyReport(ctx context.Context, userID int64, year, month int) (*MonthlyReport, error) {
	startDate := fmt.Sprintf("%04d-%02d-01", year, month)

	// 计算月末日期
	nextMonth := time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.UTC)
	endDate := nextMonth.AddDate(0, 0, -1).Format("2006-01-02")

	summary, categoryStats, dailyStats, err := s.GetStats(ctx, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// 分离收入和支出分类统计
	var incomeStats, expenseStats []*model.CategoryStats
	for _, cs := range categoryStats {
		if cs.Type == "income" {
			incomeStats = append(incomeStats, cs)
		} else {
			expenseStats = append(expenseStats, cs)
		}
	}

	return &MonthlyReport{
		Year:              year,
		Month:             month,
		Summary:           summary,
		IncomeByCategory:  incomeStats,
		ExpenseByCategory: expenseStats,
		DailyStats:        dailyStats,
	}, nil
}

// MonthlyReport 月度报表
type MonthlyReport struct {
	Year              int
	Month             int
	Summary           *model.StatsSummary
	IncomeByCategory  []*model.CategoryStats
	ExpenseByCategory []*model.CategoryStats
	DailyStats        []*model.DailyStats
}
