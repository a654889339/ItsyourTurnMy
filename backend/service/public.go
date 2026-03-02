package service

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"finance-system/database"
	"finance-system/model"
)

// PublicService 公开API服务（无需登录）
type PublicService struct {
	tableService *TableService
	dishService  *DishService
}

// NewPublicService 创建公开API服务
func NewPublicService(tableService *TableService, dishService *DishService) *PublicService {
	return &PublicService{
		tableService: tableService,
		dishService:  dishService,
	}
}

// PublicMenuResponse 公开菜单响应
type PublicMenuResponse struct {
	Table      PublicTableInfo   `json:"table"`
	Categories []string          `json:"categories"`
	Dishes     []*model.Dish     `json:"dishes"`
}

// PublicTableInfo 公开餐桌信息
type PublicTableInfo struct {
	TableNo  string `json:"table_no"`
	Capacity int    `json:"capacity"`
}

// GetPublicMenu 通过token获取菜单
func (s *PublicService) GetPublicMenu(token string) (*PublicMenuResponse, int64, error) {
	// 验证token并获取餐桌信息
	table, err := s.tableService.ValidateTableToken(token)
	if err != nil {
		return nil, 0, err
	}

	// 获取该商家的所有可用菜品
	dishes, _, err := s.dishService.ListDishes(nil, table.UserID, "", "available", 1, 1000)
	if err != nil {
		return nil, 0, err
	}

	// 获取分类列表
	categories, err := s.dishService.GetDishCategories(nil, table.UserID)
	if err != nil {
		return nil, 0, err
	}

	return &PublicMenuResponse{
		Table: PublicTableInfo{
			TableNo:  table.TableNo,
			Capacity: table.Capacity,
		},
		Categories: categories,
		Dishes:     dishes,
	}, table.UserID, nil
}

// PublicOrderRequest 公开下单请求
type PublicOrderRequest struct {
	CustomerName string                 `json:"customer_name"` // 顾客称呼（可选）
	Items        []CreateOrderItemReq   `json:"items"`
	Remark       string                 `json:"remark"`
}

// PublicOrderResponse 公开下单响应
type PublicOrderResponse struct {
	OrderNo    string             `json:"order_no"`
	TableNo    string             `json:"table_no"`
	TotalPrice float64            `json:"total_price"`
	Status     string             `json:"status"`
	Items      []model.OrderItem  `json:"items"`
	CreatedAt  time.Time          `json:"created_at"`
}

// CreatePublicOrder 通过token提交订单
func (s *PublicService) CreatePublicOrder(token string, req *PublicOrderRequest) (*PublicOrderResponse, error) {
	// 验证token并获取餐桌信息
	table, err := s.tableService.ValidateTableToken(token)
	if err != nil {
		return nil, err
	}

	// 验证订单项
	if len(req.Items) == 0 {
		return nil, errors.New("订单不能为空")
	}

	// 开始事务
	tx, err := database.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 生成订单编号
	orderNo := fmt.Sprintf("ORD%d%d", time.Now().UnixNano(), table.UserID)

	// 验证菜品并计算总价
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
			return nil, errors.New("商品数量必须大于0")
		}

		// 查询菜品（验证属于该商家）
		var dish model.Dish
		err := tx.QueryRow(`
			SELECT id, user_id, name, price, image, stock, status
			FROM dishes WHERE id = ? AND user_id = ?
		`, item.DishID, table.UserID).Scan(
			&dish.ID, &dish.UserID, &dish.Name, &dish.Price, &dish.Image, &dish.Stock, &dish.Status,
		)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("菜品不存在: %d", item.DishID)
		}
		if err != nil {
			return nil, err
		}

		// 检查菜品状态
		if dish.Status != "available" {
			return nil, fmt.Errorf("菜品 %s 已下架或售罄", dish.Name)
		}

		// 检查库存
		if dish.Stock != -1 && dish.Stock < item.Quantity {
			return nil, fmt.Errorf("菜品 %s 库存不足", dish.Name)
		}

		// 扣减库存
		if dish.Stock != -1 {
			newStock := dish.Stock - item.Quantity
			_, err = tx.Exec(
				"UPDATE dishes SET stock = ? WHERE id = ?",
				newStock, dish.ID,
			)
			if err != nil {
				return nil, err
			}
			// 记录库存变化
			stockChanges = append(stockChanges, stockChange{DishID: dish.ID, OldStock: dish.Stock, NewStock: newStock})
		}

		// 计算小计
		subtotal := dish.Price * float64(item.Quantity)
		totalPrice += subtotal

		orderItems = append(orderItems, model.OrderItem{
			DishID:    dish.ID,
			DishName:  dish.Name,
			DishImage: dish.Image,
			Price:     dish.Price,
			Quantity:  item.Quantity,
			Remark:    item.Remark,
		})
	}

	now := time.Now()

	// 创建订单
	result, err := tx.Exec(`
		INSERT INTO orders (user_id, table_id, table_no, order_no, total_price, status, order_source, customer_name, remark, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'pending', 'scan', ?, ?, ?, ?)
	`, table.UserID, table.ID, table.TableNo, orderNo, totalPrice, req.CustomerName, req.Remark, now, now)
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
		s.dishService.LogDishStockChange(sc.DishID, sc.OldStock, sc.NewStock, "扫码点餐扣减", orderNo)
	}

	return &PublicOrderResponse{
		OrderNo:    orderNo,
		TableNo:    table.TableNo,
		TotalPrice: totalPrice,
		Status:     "pending",
		Items:      orderItems,
		CreatedAt:  now,
	}, nil
}

// PublicOrderStatusResponse 订单状态响应
type PublicOrderStatusResponse struct {
	OrderNo      string            `json:"order_no"`
	TableNo      string            `json:"table_no"`
	TotalPrice   float64           `json:"total_price"`
	Status       string            `json:"status"`
	StatusText   string            `json:"status_text"`
	CustomerName string            `json:"customer_name"`
	Remark       string            `json:"remark"`
	Items        []model.OrderItem `json:"items"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// GetPublicOrderStatus 查询订单状态
func (s *PublicService) GetPublicOrderStatus(token, orderNo string) (*PublicOrderStatusResponse, error) {
	// 验证token并获取餐桌信息
	table, err := s.tableService.ValidateTableToken(token)
	if err != nil {
		return nil, err
	}

	// 查询订单
	var order model.Order
	var tableID sql.NullInt64
	err = database.DB.QueryRow(`
		SELECT id, user_id, table_id, table_no, order_no, total_price, status, customer_name, remark, created_at, updated_at
		FROM orders WHERE order_no = ? AND user_id = ?
	`, orderNo, table.UserID).Scan(
		&order.ID, &order.UserID, &tableID, &order.TableNo, &order.OrderNo,
		&order.TotalPrice, &order.Status, &order.CustomerName, &order.Remark,
		&order.CreatedAt, &order.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, errors.New("订单不存在")
	}
	if err != nil {
		return nil, err
	}

	// 验证订单是否属于该餐桌
	if !tableID.Valid || tableID.Int64 != table.ID {
		return nil, errors.New("无权查看此订单")
	}

	// 查询订单项
	rows, err := database.DB.Query(`
		SELECT id, order_id, dish_id, dish_name, dish_image, price, quantity, remark
		FROM order_items WHERE order_id = ?
	`, order.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.OrderItem
	for rows.Next() {
		var item model.OrderItem
		err := rows.Scan(
			&item.ID, &item.OrderID, &item.DishID, &item.DishName,
			&item.DishImage, &item.Price, &item.Quantity, &item.Remark,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	// 转换状态文本
	statusText := getStatusText(order.Status)

	return &PublicOrderStatusResponse{
		OrderNo:      order.OrderNo,
		TableNo:      order.TableNo,
		TotalPrice:   order.TotalPrice,
		Status:       order.Status,
		StatusText:   statusText,
		CustomerName: order.CustomerName,
		Remark:       order.Remark,
		Items:        items,
		CreatedAt:    order.CreatedAt,
		UpdatedAt:    order.UpdatedAt,
	}, nil
}

// getStatusText 获取状态中文文本
func getStatusText(status string) string {
	switch status {
	case "pending":
		return "待确认"
	case "confirmed":
		return "已确认"
	case "preparing":
		return "制作中"
	case "completed":
		return "已完成"
	case "cancelled":
		return "已取消"
	default:
		return status
	}
}

// GetTableOrders 获取本桌所有订单
func (s *PublicService) GetTableOrders(token string) ([]*PublicOrderStatusResponse, error) {
	// 验证token并获取餐桌信息
	table, err := s.tableService.ValidateTableToken(token)
	if err != nil {
		return nil, err
	}

	// 查询该餐桌的所有订单（最近24小时内的）
	rows, err := database.DB.Query(`
		SELECT id, order_no, table_no, total_price, status, customer_name, remark, created_at, updated_at
		FROM orders
		WHERE table_id = ? AND created_at > datetime('now', '-24 hours')
		ORDER BY created_at DESC
	`, table.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*PublicOrderStatusResponse
	for rows.Next() {
		var order PublicOrderStatusResponse
		var orderID int64
		err := rows.Scan(
			&orderID, &order.OrderNo, &order.TableNo, &order.TotalPrice,
			&order.Status, &order.CustomerName, &order.Remark,
			&order.CreatedAt, &order.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		order.StatusText = getStatusText(order.Status)

		// 查询订单项
		itemRows, err := database.DB.Query(`
			SELECT id, order_id, dish_id, dish_name, dish_image, price, quantity, remark
			FROM order_items WHERE order_id = ?
		`, orderID)
		if err != nil {
			return nil, err
		}

		for itemRows.Next() {
			var item model.OrderItem
			err := itemRows.Scan(
				&item.ID, &item.OrderID, &item.DishID, &item.DishName,
				&item.DishImage, &item.Price, &item.Quantity, &item.Remark,
			)
			if err != nil {
				itemRows.Close()
				return nil, err
			}
			order.Items = append(order.Items, item)
		}
		itemRows.Close()

		orders = append(orders, &order)
	}

	return orders, nil
}
