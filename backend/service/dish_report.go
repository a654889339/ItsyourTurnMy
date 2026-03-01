package service

import (
	"database/sql"
	"time"

	"finance-system/database"
)

// DishReportService 菜品报表服务
type DishReportService struct{}

// NewDishReportService 创建菜品报表服务
func NewDishReportService() *DishReportService {
	return &DishReportService{}
}

// DishSalesItem 菜品销售项
type DishSalesItem struct {
	DishID    int64   `json:"dish_id"`
	DishName  string  `json:"dish_name"`
	Category  string  `json:"category"`
	Quantity  int     `json:"quantity"`
	Revenue   float64 `json:"revenue"`
}

// DailyStat 每日统计
type DailyStat struct {
	Date         string  `json:"date"`
	OrderCount   int     `json:"order_count"`
	TotalQuantity int    `json:"total_quantity"`
	TotalRevenue float64 `json:"total_revenue"`
}

// DishReportSummary 报表汇总
type DishReportSummary struct {
	Period        string           `json:"period"`        // daily, weekly, monthly, quarterly
	StartDate     string           `json:"start_date"`
	EndDate       string           `json:"end_date"`
	TotalOrders   int              `json:"total_orders"`
	TotalQuantity int              `json:"total_quantity"`
	TotalRevenue  float64          `json:"total_revenue"`
	DishCount     int              `json:"dish_count"`     // 上架菜品数量
	TopDishes     []DishSalesItem  `json:"top_dishes"`     // 销量TOP菜品
	DailyStats    []DailyStat      `json:"daily_stats"`    // 每日统计
	CategoryStats []CategoryStat   `json:"category_stats"` // 分类统计
}

// CategoryStat 分类统计
type CategoryStat struct {
	Category  string  `json:"category"`
	Quantity  int     `json:"quantity"`
	Revenue   float64 `json:"revenue"`
	Percentage float64 `json:"percentage"`
}

// GetDishReport 获取菜品报表
func (s *DishReportService) GetDishReport(userID int64, period string, startDate, endDate time.Time) (*DishReportSummary, error) {
	summary := &DishReportSummary{
		Period:    period,
		StartDate: startDate.Format("2006-01-02"),
		EndDate:   endDate.Format("2006-01-02"),
	}

	// 1. 获取订单总数、总数量、总收入
	err := database.DB.QueryRow(`
		SELECT COUNT(DISTINCT o.id), COALESCE(SUM(oi.quantity), 0), COALESCE(SUM(oi.price * oi.quantity), 0)
		FROM orders o
		LEFT JOIN order_items oi ON o.id = oi.order_id
		WHERE o.user_id = ? AND o.status != 'cancelled' AND o.created_at >= ? AND o.created_at < ?
	`, userID, startDate, endDate.AddDate(0, 0, 1)).Scan(&summary.TotalOrders, &summary.TotalQuantity, &summary.TotalRevenue)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// 2. 获取上架菜品数量
	err = database.DB.QueryRow(`
		SELECT COUNT(*) FROM dishes WHERE user_id = ? AND status = 'available'
	`, userID).Scan(&summary.DishCount)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// 3. 获取销量TOP10菜品
	topDishes, err := s.getTopDishes(userID, startDate, endDate, 10)
	if err != nil {
		return nil, err
	}
	summary.TopDishes = topDishes

	// 4. 获取每日统计
	dailyStats, err := s.getDailyStats(userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	summary.DailyStats = dailyStats

	// 5. 获取分类统计
	categoryStats, err := s.getCategoryStats(userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	summary.CategoryStats = categoryStats

	return summary, nil
}

// getTopDishes 获取销量TOP菜品
func (s *DishReportService) getTopDishes(userID int64, startDate, endDate time.Time, limit int) ([]DishSalesItem, error) {
	rows, err := database.DB.Query(`
		SELECT oi.dish_id, oi.dish_name, COALESCE(d.category, '未分类') as category,
			   SUM(oi.quantity) as total_quantity, SUM(oi.price * oi.quantity) as total_revenue
		FROM order_items oi
		JOIN orders o ON oi.order_id = o.id
		LEFT JOIN dishes d ON oi.dish_id = d.id
		WHERE o.user_id = ? AND o.status != 'cancelled' AND o.created_at >= ? AND o.created_at < ?
		GROUP BY oi.dish_id, oi.dish_name
		ORDER BY total_quantity DESC
		LIMIT ?
	`, userID, startDate, endDate.AddDate(0, 0, 1), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []DishSalesItem
	for rows.Next() {
		var item DishSalesItem
		if err := rows.Scan(&item.DishID, &item.DishName, &item.Category, &item.Quantity, &item.Revenue); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// getDailyStats 获取每日统计
func (s *DishReportService) getDailyStats(userID int64, startDate, endDate time.Time) ([]DailyStat, error) {
	rows, err := database.DB.Query(`
		SELECT DATE(o.created_at) as date,
			   COUNT(DISTINCT o.id) as order_count,
			   COALESCE(SUM(oi.quantity), 0) as total_quantity,
			   COALESCE(SUM(oi.price * oi.quantity), 0) as total_revenue
		FROM orders o
		LEFT JOIN order_items oi ON o.id = oi.order_id
		WHERE o.user_id = ? AND o.status != 'cancelled' AND o.created_at >= ? AND o.created_at < ?
		GROUP BY DATE(o.created_at)
		ORDER BY date ASC
	`, userID, startDate, endDate.AddDate(0, 0, 1))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []DailyStat
	for rows.Next() {
		var stat DailyStat
		if err := rows.Scan(&stat.Date, &stat.OrderCount, &stat.TotalQuantity, &stat.TotalRevenue); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}
	return stats, nil
}

// getCategoryStats 获取分类统计
func (s *DishReportService) getCategoryStats(userID int64, startDate, endDate time.Time) ([]CategoryStat, error) {
	rows, err := database.DB.Query(`
		SELECT COALESCE(d.category, '未分类') as category,
			   SUM(oi.quantity) as total_quantity,
			   SUM(oi.price * oi.quantity) as total_revenue
		FROM order_items oi
		JOIN orders o ON oi.order_id = o.id
		LEFT JOIN dishes d ON oi.dish_id = d.id
		WHERE o.user_id = ? AND o.status != 'cancelled' AND o.created_at >= ? AND o.created_at < ?
		GROUP BY category
		ORDER BY total_revenue DESC
	`, userID, startDate, endDate.AddDate(0, 0, 1))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []CategoryStat
	var totalRevenue float64 = 0
	for rows.Next() {
		var stat CategoryStat
		if err := rows.Scan(&stat.Category, &stat.Quantity, &stat.Revenue); err != nil {
			return nil, err
		}
		totalRevenue += stat.Revenue
		stats = append(stats, stat)
	}

	// 计算百分比
	for i := range stats {
		if totalRevenue > 0 {
			stats[i].Percentage = stats[i].Revenue / totalRevenue * 100
		}
	}

	return stats, nil
}

// GetPeriodDates 根据周期类型获取开始和结束日期
func GetPeriodDates(period string) (time.Time, time.Time) {
	now := time.Now()
	var startDate, endDate time.Time

	switch period {
	case "daily":
		// 今天
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endDate = startDate
	case "weekly":
		// 本周（从周一开始）
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		startDate = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 0, 6)
	case "monthly":
		// 本月
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 1, -1)
	case "quarterly":
		// 本季度
		quarter := (int(now.Month()) - 1) / 3
		startDate = time.Date(now.Year(), time.Month(quarter*3+1), 1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 3, -1)
	default:
		// 默认本月
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 1, -1)
	}

	return startDate, endDate
}

// GetTrendData 获取趋势数据（用于图表）
func (s *DishReportService) GetTrendData(userID int64, period string, count int) ([]DailyStat, error) {
	now := time.Now()
	var stats []DailyStat

	switch period {
	case "daily":
		// 最近N天
		startDate := time.Date(now.Year(), now.Month(), now.Day()-count+1, 0, 0, 0, 0, now.Location())
		endDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return s.getDailyStats(userID, startDate, endDate)
	case "weekly":
		// 最近N周
		for i := count - 1; i >= 0; i-- {
			weekStart := now.AddDate(0, 0, -int(now.Weekday())-7*i)
			weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, now.Location())
			weekEnd := weekStart.AddDate(0, 0, 6)

			var stat DailyStat
			stat.Date = weekStart.Format("2006-01-02")
			err := database.DB.QueryRow(`
				SELECT COUNT(DISTINCT o.id), COALESCE(SUM(oi.quantity), 0), COALESCE(SUM(oi.price * oi.quantity), 0)
				FROM orders o
				LEFT JOIN order_items oi ON o.id = oi.order_id
				WHERE o.user_id = ? AND o.status != 'cancelled' AND o.created_at >= ? AND o.created_at < ?
			`, userID, weekStart, weekEnd.AddDate(0, 0, 1)).Scan(&stat.OrderCount, &stat.TotalQuantity, &stat.TotalRevenue)
			if err != nil && err != sql.ErrNoRows {
				return nil, err
			}
			stats = append(stats, stat)
		}
	case "monthly":
		// 最近N月
		for i := count - 1; i >= 0; i-- {
			monthStart := time.Date(now.Year(), now.Month()-time.Month(i), 1, 0, 0, 0, 0, now.Location())
			monthEnd := monthStart.AddDate(0, 1, -1)

			var stat DailyStat
			stat.Date = monthStart.Format("2006-01")
			err := database.DB.QueryRow(`
				SELECT COUNT(DISTINCT o.id), COALESCE(SUM(oi.quantity), 0), COALESCE(SUM(oi.price * oi.quantity), 0)
				FROM orders o
				LEFT JOIN order_items oi ON o.id = oi.order_id
				WHERE o.user_id = ? AND o.status != 'cancelled' AND o.created_at >= ? AND o.created_at < ?
			`, userID, monthStart, monthEnd.AddDate(0, 0, 1)).Scan(&stat.OrderCount, &stat.TotalQuantity, &stat.TotalRevenue)
			if err != nil && err != sql.ErrNoRows {
				return nil, err
			}
			stats = append(stats, stat)
		}
	}

	return stats, nil
}
