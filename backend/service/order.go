package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"finance-system/database"
	"finance-system/model"
)

// OrderService 订单服务
type OrderService struct {
	dishService *DishService
}

// NewOrderService 创建订单服务
func NewOrderService(dishService *DishService) *OrderService {
	return &OrderService{dishService: dishService}
}

// CreateOrderReq 创建订单请求
type CreateOrderReq struct {
	Items   []CreateOrderItemReq `json:"items"`
	Remark  string               `json:"remark"`
	TableID *int64               `json:"table_id"`
}

// CreateOrderItemReq 创建订单项请求
type CreateOrderItemReq struct {
	DishID   int64  `json:"dish_id"`
	Quantity int    `json:"quantity"`
	Remark   string `json:"remark"`
}

// CreateOrder 创建订单
func (s *OrderService) CreateOrder(ctx context.Context, userID int64, req *CreateOrderReq) (*model.Order, error) {
	if len(req.Items) == 0 {
		return nil, errors.New("订单项不能为空")
	}

	// 生成订单编号
	orderNo := fmt.Sprintf("ORD%d%d", time.Now().UnixNano(), userID)

	// 开启事务
	tx, err := database.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 计算总价并验证菜品
	var totalPrice float64
	var orderItems []model.OrderItem

	// 收集库存变化信息，用于事务提交后记录
	type stockChange struct {
		DishID   int64
		OldStock int
		NewStock int
	}
	var stockChanges []stockChange

	for _, item := range req.Items {
		if item.Quantity <= 0 {
			return nil, errors.New("菜品数量必须大于0")
		}

		// 获取菜品信息
		var dish model.Dish
		err := tx.QueryRow(`
			SELECT id, name, price, image, stock, status FROM dishes WHERE id = ? AND user_id = ?
		`, item.DishID, userID).Scan(&dish.ID, &dish.Name, &dish.Price, &dish.Image, &dish.Stock, &dish.Status)

		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("菜品ID %d 不存在", item.DishID)
		}
		if err != nil {
			return nil, err
		}

		if dish.Status != "available" {
			return nil, fmt.Errorf("菜品 %s 暂时不可用", dish.Name)
		}

		if dish.Stock != -1 && dish.Stock < item.Quantity {
			return nil, fmt.Errorf("菜品 %s 库存不足", dish.Name)
		}

		// 扣减库存
		if dish.Stock != -1 {
			newStock := dish.Stock - item.Quantity
			_, err = tx.Exec("UPDATE dishes SET stock = ?, updated_at = ? WHERE id = ?", newStock, time.Now(), dish.ID)
			if err != nil {
				return nil, err
			}
			// 记录库存变化
			stockChanges = append(stockChanges, stockChange{DishID: dish.ID, OldStock: dish.Stock, NewStock: newStock})
		}

		totalPrice += dish.Price * float64(item.Quantity)
		orderItems = append(orderItems, model.OrderItem{
			DishID:    dish.ID,
			DishName:  dish.Name,
			DishImage: dish.Image,
			Price:     dish.Price,
			Quantity:  item.Quantity,
			Remark:    item.Remark,
		})
	}

	// 创建订单（后台下单，可选关联餐桌）
	now := time.Now()
	var tableID interface{}
	var tableNo string

	// 如果指定了桌号，查询桌号信息
	if req.TableID != nil && *req.TableID > 0 {
		var tNo string
		err := tx.QueryRow("SELECT table_no FROM tables WHERE id = ? AND user_id = ?", *req.TableID, userID).Scan(&tNo)
		if err == nil {
			tableID = *req.TableID
			tableNo = tNo
		} else {
			tableID = nil
		}
	}

	result, err := tx.Exec(`
		INSERT INTO orders (user_id, table_id, table_no, order_no, total_price, status, order_source, customer_name, remark, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'pending', 'admin', '', ?, ?, ?)
	`, userID, tableID, tableNo, orderNo, totalPrice, req.Remark, now, now)

	if err != nil {
		return nil, err
	}

	orderID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	// 创建订单项
	for i := range orderItems {
		result, err := tx.Exec(`
			INSERT INTO order_items (order_id, dish_id, dish_name, dish_image, price, quantity, remark)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, orderID, orderItems[i].DishID, orderItems[i].DishName, orderItems[i].DishImage,
			orderItems[i].Price, orderItems[i].Quantity, orderItems[i].Remark)

		if err != nil {
			return nil, err
		}

		itemID, _ := result.LastInsertId()
		orderItems[i].ID = itemID
		orderItems[i].OrderID = orderID
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// 事务提交成功后，记录库存变化日志
	for _, sc := range stockChanges {
		s.dishService.LogDishStockChange(sc.DishID, sc.OldStock, sc.NewStock, "点餐扣减", orderNo)
	}

	order := &model.Order{
		ID:          orderID,
		UserID:      userID,
		OrderNo:     orderNo,
		TableNo:     tableNo,
		TotalPrice:  totalPrice,
		Status:      "pending",
		OrderSource: "admin",
		Remark:      req.Remark,
		Items:       orderItems,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if req.TableID != nil && *req.TableID > 0 {
		order.TableID = req.TableID
	}

	return order, nil
}

// GetOrder 获取订单详情
func (s *OrderService) GetOrder(ctx context.Context, userID, orderID int64) (*model.Order, error) {
	var order model.Order
	var tableID sql.NullInt64
	err := database.DB.QueryRow(`
		SELECT id, user_id, table_id, table_no, order_no, total_price, status, order_source, customer_name, remark, created_at, updated_at
		FROM orders WHERE id = ? AND user_id = ?
	`, orderID, userID).Scan(
		&order.ID, &order.UserID, &tableID, &order.TableNo, &order.OrderNo, &order.TotalPrice,
		&order.Status, &order.OrderSource, &order.CustomerName, &order.Remark, &order.CreatedAt, &order.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("订单不存在")
	}
	if err != nil {
		return nil, err
	}

	if tableID.Valid {
		order.TableID = &tableID.Int64
	}

	// 获取订单项
	rows, err := database.DB.Query(`
		SELECT id, order_id, dish_id, dish_name, dish_image, price, quantity, remark
		FROM order_items WHERE order_id = ?
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item model.OrderItem
		err := rows.Scan(&item.ID, &item.OrderID, &item.DishID, &item.DishName,
			&item.DishImage, &item.Price, &item.Quantity, &item.Remark)
		if err != nil {
			return nil, err
		}
		order.Items = append(order.Items, item)
	}

	return &order, nil
}

// ListOrdersReq 订单列表查询参数
type ListOrdersReq struct {
	Status      string // 订单状态筛选
	OrderSource string // 订单来源筛选 (admin/scan)
	TableID     int64  // 餐桌ID筛选
	StartDate   string // 开始日期 (YYYY-MM-DD)
	EndDate     string // 结束日期 (YYYY-MM-DD)
	Page        int
	PageSize    int
}

// ListOrders 获取订单列表
func (s *OrderService) ListOrders(ctx context.Context, userID int64, status string, page, pageSize int) ([]*model.Order, int64, error) {
	return s.ListOrdersWithFilter(ctx, userID, &ListOrdersReq{
		Status:   status,
		Page:     page,
		PageSize: pageSize,
	})
}

// ListOrdersWithFilter 带筛选条件获取订单列表
func (s *OrderService) ListOrdersWithFilter(ctx context.Context, userID int64, req *ListOrdersReq) ([]*model.Order, int64, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	// 构建查询
	query := "SELECT id, user_id, table_id, table_no, order_no, total_price, status, order_source, customer_name, remark, created_at, updated_at FROM orders WHERE user_id = ?"
	countQuery := "SELECT COUNT(*) FROM orders WHERE user_id = ?"
	args := []interface{}{userID}
	countArgs := []interface{}{userID}

	if req.Status != "" {
		query += " AND status = ?"
		countQuery += " AND status = ?"
		args = append(args, req.Status)
		countArgs = append(countArgs, req.Status)
	}

	if req.OrderSource != "" {
		query += " AND order_source = ?"
		countQuery += " AND order_source = ?"
		args = append(args, req.OrderSource)
		countArgs = append(countArgs, req.OrderSource)
	}

	if req.TableID > 0 {
		query += " AND table_id = ?"
		countQuery += " AND table_id = ?"
		args = append(args, req.TableID)
		countArgs = append(countArgs, req.TableID)
	}

	if req.StartDate != "" {
		query += " AND date(created_at) >= ?"
		countQuery += " AND date(created_at) >= ?"
		args = append(args, req.StartDate)
		countArgs = append(countArgs, req.StartDate)
	}

	if req.EndDate != "" {
		query += " AND date(created_at) <= ?"
		countQuery += " AND date(created_at) <= ?"
		args = append(args, req.EndDate)
		countArgs = append(countArgs, req.EndDate)
	}

	// 获取总数
	var total int64
	err := database.DB.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, req.PageSize, (req.Page-1)*req.PageSize)

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orders []*model.Order
	for rows.Next() {
		var order model.Order
		var tableID sql.NullInt64
		err := rows.Scan(&order.ID, &order.UserID, &tableID, &order.TableNo, &order.OrderNo, &order.TotalPrice,
			&order.Status, &order.OrderSource, &order.CustomerName, &order.Remark, &order.CreatedAt, &order.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}

		if tableID.Valid {
			order.TableID = &tableID.Int64
		}

		// 获取订单项
		itemRows, err := database.DB.Query(`
			SELECT id, order_id, dish_id, dish_name, dish_image, price, quantity, remark
			FROM order_items WHERE order_id = ?
		`, order.ID)
		if err != nil {
			return nil, 0, err
		}

		for itemRows.Next() {
			var item model.OrderItem
			err := itemRows.Scan(&item.ID, &item.OrderID, &item.DishID, &item.DishName,
				&item.DishImage, &item.Price, &item.Quantity, &item.Remark)
			if err != nil {
				itemRows.Close()
				return nil, 0, err
			}
			order.Items = append(order.Items, item)
		}
		itemRows.Close()

		orders = append(orders, &order)
	}

	return orders, total, nil
}

// UpdateOrderStatus 更新订单状态
func (s *OrderService) UpdateOrderStatus(ctx context.Context, userID, orderID int64, status string) error {
	validStatuses := map[string]bool{
		"pending":   true,
		"confirmed": true,
		"preparing": true,
		"completed": true,
		"cancelled": true,
	}

	if !validStatuses[status] {
		return errors.New("无效的订单状态")
	}

	// 收集库存恢复信息，用于事务提交后记录
	type stockRestore struct {
		DishID   int64
		OldStock int
		NewStock int
	}
	var stockRestores []stockRestore
	var orderNo string

	// 如果是取消订单，需要恢复库存
	if status == "cancelled" {
		// 先获取订单号
		var currentStatus string
		err := database.DB.QueryRow("SELECT order_no, status FROM orders WHERE id = ? AND user_id = ?", orderID, userID).Scan(&orderNo, &currentStatus)
		if err == sql.ErrNoRows {
			return errors.New("订单不存在")
		}
		if err != nil {
			return err
		}
		if currentStatus == "cancelled" {
			return errors.New("订单已取消")
		}

		// 获取订单项和当前库存
		rows, err := database.DB.Query(`
			SELECT oi.dish_id, oi.quantity, d.stock FROM order_items oi
			JOIN dishes d ON oi.dish_id = d.id
			WHERE oi.order_id = ?
		`, orderID)
		if err != nil {
			return err
		}

		// 先收集所有数据，然后关闭rows
		type itemInfo struct {
			DishID   int64
			Quantity int
			Stock    int
		}
		var items []itemInfo
		for rows.Next() {
			var info itemInfo
			if err := rows.Scan(&info.DishID, &info.Quantity, &info.Stock); err != nil {
				rows.Close()
				return err
			}
			items = append(items, info)
		}
		rows.Close()

		// 使用事务更新库存和订单状态
		tx, err := database.DB.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		for _, item := range items {
			if item.Stock != -1 {
				newStock := item.Stock + item.Quantity
				_, err = tx.Exec("UPDATE dishes SET stock = ?, updated_at = ? WHERE id = ?", newStock, time.Now(), item.DishID)
				if err != nil {
					return err
				}
				stockRestores = append(stockRestores, stockRestore{DishID: item.DishID, OldStock: item.Stock, NewStock: newStock})
			}
		}

		// 更新订单状态
		result, err := tx.Exec(`UPDATE orders SET status = ?, updated_at = ? WHERE id = ? AND user_id = ?`, status, time.Now(), orderID, userID)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return errors.New("订单不存在或无权修改")
		}

		// 提交事务
		if err := tx.Commit(); err != nil {
			return err
		}

		// 事务提交成功后，记录库存变化日志
		for _, sr := range stockRestores {
			s.dishService.LogDishStockChange(sr.DishID, sr.OldStock, sr.NewStock, "取消订单恢复", orderNo)
		}

		return nil
	}

	// 非取消状态，直接更新
	result, err := database.DB.Exec(`
		UPDATE orders SET status = ?, updated_at = ? WHERE id = ? AND user_id = ?
	`, status, time.Now(), orderID, userID)

	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("订单不存在或无权修改")
	}

	return nil
}

// UpdateOrderReq 更新订单请求
type UpdateOrderReq struct {
	Items  []CreateOrderItemReq `json:"items"`
	Remark string               `json:"remark"`
}

// UpdateOrder 更新订单（只能更新待确认的订单）
func (s *OrderService) UpdateOrder(ctx context.Context, userID, orderID int64, req *UpdateOrderReq) (*model.Order, error) {
	// 验证订单状态
	var status string
	err := database.DB.QueryRow("SELECT status FROM orders WHERE id = ? AND user_id = ?", orderID, userID).Scan(&status)
	if err == sql.ErrNoRows {
		return nil, errors.New("订单不存在")
	}
	if err != nil {
		return nil, err
	}

	if status != "pending" {
		return nil, errors.New("只能修改待确认的订单")
	}

	if len(req.Items) == 0 {
		return nil, errors.New("订单项不能为空")
	}

	// 开启事务
	tx, err := database.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 先恢复原订单的库存
	rows, err := tx.Query(`
		SELECT oi.dish_id, oi.quantity FROM order_items oi
		WHERE oi.order_id = ?
	`, orderID)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var dishID int64
		var quantity int
		if err := rows.Scan(&dishID, &quantity); err != nil {
			rows.Close()
			return nil, err
		}
		// 恢复库存
		_, err = tx.Exec(`
			UPDATE dishes SET stock = CASE WHEN stock = -1 THEN -1 ELSE stock + ? END, updated_at = ?
			WHERE id = ?
		`, quantity, time.Now(), dishID)
		if err != nil {
			rows.Close()
			return nil, err
		}
	}
	rows.Close()

	// 删除原订单项
	_, err = tx.Exec("DELETE FROM order_items WHERE order_id = ?", orderID)
	if err != nil {
		return nil, err
	}

	// 计算新总价并验证菜品
	var totalPrice float64
	var orderItems []model.OrderItem

	for _, item := range req.Items {
		if item.Quantity <= 0 {
			return nil, errors.New("菜品数量必须大于0")
		}

		// 获取菜品信息
		var dish model.Dish
		err := tx.QueryRow(`
			SELECT id, name, price, image, stock, status FROM dishes WHERE id = ? AND user_id = ?
		`, item.DishID, userID).Scan(&dish.ID, &dish.Name, &dish.Price, &dish.Image, &dish.Stock, &dish.Status)

		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("菜品ID %d 不存在", item.DishID)
		}
		if err != nil {
			return nil, err
		}

		if dish.Status != "available" {
			return nil, fmt.Errorf("菜品 %s 暂时不可用", dish.Name)
		}

		if dish.Stock != -1 && dish.Stock < item.Quantity {
			return nil, fmt.Errorf("菜品 %s 库存不足", dish.Name)
		}

		// 扣减库存
		if dish.Stock != -1 {
			newStock := dish.Stock - item.Quantity
			_, err = tx.Exec("UPDATE dishes SET stock = ?, updated_at = ? WHERE id = ?", newStock, time.Now(), dish.ID)
			if err != nil {
				return nil, err
			}
		}

		totalPrice += dish.Price * float64(item.Quantity)
		orderItems = append(orderItems, model.OrderItem{
			OrderID:   orderID,
			DishID:    dish.ID,
			DishName:  dish.Name,
			DishImage: dish.Image,
			Price:     dish.Price,
			Quantity:  item.Quantity,
			Remark:    item.Remark,
		})
	}

	// 更新订单
	now := time.Now()
	_, err = tx.Exec(`
		UPDATE orders SET total_price = ?, remark = ?, updated_at = ? WHERE id = ?
	`, totalPrice, req.Remark, now, orderID)
	if err != nil {
		return nil, err
	}

	// 创建新订单项
	for i := range orderItems {
		result, err := tx.Exec(`
			INSERT INTO order_items (order_id, dish_id, dish_name, dish_image, price, quantity, remark)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, orderID, orderItems[i].DishID, orderItems[i].DishName, orderItems[i].DishImage,
			orderItems[i].Price, orderItems[i].Quantity, orderItems[i].Remark)

		if err != nil {
			return nil, err
		}

		itemID, _ := result.LastInsertId()
		orderItems[i].ID = itemID
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// 返回更新后的订单
	return s.GetOrder(ctx, userID, orderID)
}

// DeleteOrder 删除订单
func (s *OrderService) DeleteOrder(ctx context.Context, userID, orderID int64) error {
	// 只能删除已取消或已完成的订单
	var status string
	err := database.DB.QueryRow("SELECT status FROM orders WHERE id = ? AND user_id = ?", orderID, userID).Scan(&status)
	if err == sql.ErrNoRows {
		return errors.New("订单不存在")
	}
	if err != nil {
		return err
	}

	if status != "cancelled" && status != "completed" {
		return errors.New("只能删除已取消或已完成的订单")
	}

	// 删除订单项
	_, err = database.DB.Exec("DELETE FROM order_items WHERE order_id = ?", orderID)
	if err != nil {
		return err
	}

	// 删除订单
	_, err = database.DB.Exec("DELETE FROM orders WHERE id = ?", orderID)
	return err
}
