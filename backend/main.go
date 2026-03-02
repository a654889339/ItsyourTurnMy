package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"finance-system/config"
	"finance-system/database"
	"finance-system/service"
)

var (
	authService        *service.AuthService
	accountService     *service.AccountService
	transactionService *service.TransactionService
	categoryService    *service.CategoryService
	reportService      *service.ReportService
	emailService       *service.EmailService
	dishService        *service.DishService
	orderService       *service.OrderService
	uploadService      *service.UploadService
	dishReportService  *service.DishReportService
	tableService       *service.TableService
	publicService      *service.PublicService
)

func main() {
	// 命令行参数
	configPath := flag.String("config", "", "配置文件路径")
	flag.Parse()

	// 加载配置
	var cfg *config.Config
	var err error

	if *configPath != "" {
		cfg, err = config.Load(*configPath)
		if err != nil {
			log.Fatalf("加载配置文件失败: %v", err)
		}
		log.Printf("从配置文件加载配置: %s", *configPath)
	} else if os.Getenv("CONFIG_FILE") != "" {
		cfg, err = config.Load(os.Getenv("CONFIG_FILE"))
		if err != nil {
			log.Fatalf("加载配置文件失败: %v", err)
		}
		log.Printf("从环境变量CONFIG_FILE加载配置: %s", os.Getenv("CONFIG_FILE"))
	} else {
		// 尝试默认配置文件路径
		defaultPaths := []string{
			"./config/config.yaml",
			"./config.yaml",
			"/etc/finance/config.yaml",
		}
		for _, p := range defaultPaths {
			if _, err := os.Stat(p); err == nil {
				cfg, err = config.Load(p)
				if err != nil {
					log.Fatalf("加载配置文件失败: %v", err)
				}
				log.Printf("从默认路径加载配置: %s", p)
				break
			}
		}
		// 如果没有配置文件，从环境变量加载
		if cfg == nil {
			cfg = config.LoadFromEnv()
			log.Println("从环境变量加载配置")
		}
	}

	// 确保数据目录存在
	if cfg.Database.Driver == "sqlite" {
		dataDir := filepath.Dir(cfg.Database.SQLitePath)
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			log.Fatalf("创建数据目录失败: %v", err)
		}
	}

	// 初始化数据库
	var dbPath string
	if cfg.Database.Driver == "mysql" {
		dbPath = cfg.Database.GetMySQLDSN()
	} else {
		dbPath = cfg.Database.SQLitePath
	}

	if err := database.InitDB(dbPath); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer database.Close()

	// 初始化速率限制器
	service.InitRateLimiters()

	// 初始化服务
	authService = service.NewAuthService()
	// 设置JWT密钥
	if cfg.JWT.Secret != "" {
		service.SetJWTSecret(cfg.JWT.Secret)
	}
	accountService = service.NewAccountService()
	transactionService = service.NewTransactionService(accountService)
	categoryService = service.NewCategoryService()
	reportService = service.NewReportService()
	emailService = service.NewEmailService(&cfg.Email)
	dishService = service.NewDishService()
	orderService = service.NewOrderService(dishService)
	uploadService = service.NewUploadService("./uploads", "")
	dishReportService = service.NewDishReportService()
	tableService = service.NewTableService()
	publicService = service.NewPublicService(tableService, dishService)

	// 创建路由
	mux := http.NewServeMux()

	// 健康检查 (用于负载均衡探测)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/api/v1/health", handleHealth)

	// 认证相关
	mux.HandleFunc("/api/v1/auth/send-code", handleSendVerificationCode)
	mux.HandleFunc("/api/v1/auth/register", handleRegister)
	mux.HandleFunc("/api/v1/auth/login", handleLogin)
	mux.HandleFunc("/api/v1/auth/me", authMiddleware(handleGetCurrentUser))

	// 账户相关
	mux.HandleFunc("/api/v1/accounts", authMiddleware(handleAccounts))
	mux.HandleFunc("/api/v1/accounts/", authMiddleware(handleAccountByID))

	// 交易相关
	mux.HandleFunc("/api/v1/transactions", authMiddleware(handleTransactions))
	mux.HandleFunc("/api/v1/transactions/", authMiddleware(handleTransactionByID))

	// 分类相关
	mux.HandleFunc("/api/v1/categories", authMiddleware(handleCategories))
	mux.HandleFunc("/api/v1/categories/", authMiddleware(handleCategoryByID))

	// 报表相关
	mux.HandleFunc("/api/v1/reports/stats", authMiddleware(handleStats))
	mux.HandleFunc("/api/v1/reports/monthly", authMiddleware(handleMonthlyReport))

	// 菜品管理
	mux.HandleFunc("/api/v1/dishes", authMiddleware(handleDishes))
	mux.HandleFunc("/api/v1/dishes/", authMiddleware(handleDishByID))
	mux.HandleFunc("/api/v1/dishes/categories", authMiddleware(handleDishCategories))

	// 订单管理
	mux.HandleFunc("/api/v1/orders", authMiddleware(handleOrders))
	mux.HandleFunc("/api/v1/orders/", authMiddleware(handleOrderByID))

	// 餐桌管理
	mux.HandleFunc("/api/v1/tables", authMiddleware(handleTables))
	mux.HandleFunc("/api/v1/tables/", authMiddleware(handleTableByID))

	// 公开API（扫码点单，无需登录）
	mux.HandleFunc("/api/v1/public/menu/", handlePublicMenu)
	mux.HandleFunc("/api/v1/public/order/", handlePublicOrder)
	mux.HandleFunc("/api/v1/public/orders/", handlePublicTableOrders)

	// 菜品报表
	mux.HandleFunc("/api/v1/dish-reports", authMiddleware(handleDishReports))
	mux.HandleFunc("/api/v1/dish-reports/trend", authMiddleware(handleDishReportTrend))

	// 图片上传
	mux.HandleFunc("/api/v1/upload/image", authMiddleware(handleUploadImage))

	// 静态文件服务（图片访问）
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))

	// 启用CORS
	handler := corsMiddleware(mux, cfg)

	// 创建HTTP服务器
	server := &http.Server{
		Addr:         cfg.Server.GetAddr(),
		Handler:      handler,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// 优雅关闭
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Println("正在关闭服务器...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("服务器关闭错误: %v", err)
		}
	}()

	log.Printf("服务器启动在 %s (模式: %s)", cfg.Server.GetAddr(), cfg.Server.Mode)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("服务器启动失败: %v", err)
	}

	log.Println("服务器已关闭")
}

// CORS中间件
func corsMiddleware(next http.Handler, cfg *config.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// 检查是否允许该来源
		allowed := false
		for _, o := range cfg.CORS.AllowedOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if allowed && origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else if len(cfg.CORS.AllowedOrigins) > 0 && cfg.CORS.AllowedOrigins[0] == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.CORS.AllowedMethods, ", "))
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.CORS.AllowedHeaders, ", "))

		if cfg.CORS.AllowCredentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// 认证中间件
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			jsonError(w, "未授权", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		userID, err := authService.ValidateToken(token)
		if err != nil {
			jsonError(w, "无效的Token", http.StatusUnauthorized)
			return
		}

		// 获取用户名
		user, _ := authService.GetUserByID(r.Context(), userID)
		username := ""
		if user != nil {
			username = user.Username
		}

		ctx := context.WithValue(r.Context(), "user_id", userID)
		ctx = context.WithValue(ctx, "username", username)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// JSON响应帮助函数
func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    code,
		"message": message,
	})
}

func getUserID(r *http.Request) int64 {
	return r.Context().Value("user_id").(int64)
}

// 健康检查
func handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]interface{}{
		"status":  "ok",
		"time":    time.Now().Format(time.RFC3339),
		"version": "1.0.0",
	})
}

// ==================== 认证处理 ====================

// getClientIP 获取客户端真实IP
func getClientIP(r *http.Request) string {
	// 优先从 X-Forwarded-For 获取（反向代理场景）
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// X-Forwarded-For 可能包含多个IP，取第一个
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// 其次从 X-Real-IP 获取
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// 最后从 RemoteAddr 获取
	ip := r.RemoteAddr
	// 去掉端口号
	if colonIdx := strings.LastIndex(ip, ":"); colonIdx != -1 {
		ip = ip[:colonIdx]
	}
	return ip
}

// 发送邮箱验证码
func handleSendVerificationCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	// 速率限制检查
	clientIP := getClientIP(r)
	if allowed, remaining := service.SendCodeRateLimiter.Allow(clientIP); !allowed {
		jsonError(w, fmt.Sprintf("请求过于频繁，请 %d 秒后再试", remaining), http.StatusTooManyRequests)
		return
	}

	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "无效的请求", http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		jsonError(w, "邮箱不能为空", http.StatusBadRequest)
		return
	}

	// 检查邮箱是否已注册
	if authService.EmailExists(req.Email) {
		jsonError(w, "该邮箱已被注册", http.StatusBadRequest)
		return
	}

	_, err := emailService.SendVerificationCode(req.Email)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"message": "验证码已发送",
	})
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	// 速率限制检查
	clientIP := getClientIP(r)
	if allowed, remaining := service.RegisterRateLimiter.Allow(clientIP); !allowed {
		jsonError(w, fmt.Sprintf("请求过于频繁，请 %d 秒后再试", remaining), http.StatusTooManyRequests)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
		Code     string `json:"code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "无效的请求", http.StatusBadRequest)
		return
	}

	// 验证验证码
	if req.Code == "" {
		jsonError(w, "验证码不能为空", http.StatusBadRequest)
		return
	}

	if !emailService.VerifyCode(req.Email, req.Code) {
		jsonError(w, "验证码错误或已过期", http.StatusBadRequest)
		return
	}

	user, err := authService.Register(r.Context(), req.Username, req.Password, req.Email)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonResponse(w, user)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	// 速率限制检查
	clientIP := getClientIP(r)
	if allowed, remaining := service.LoginRateLimiter.Allow(clientIP); !allowed {
		jsonError(w, fmt.Sprintf("登录尝试过于频繁，请 %d 秒后再试", remaining), http.StatusTooManyRequests)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "无效的请求", http.StatusBadRequest)
		return
	}

	token, user, err := authService.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		jsonError(w, err.Error(), http.StatusUnauthorized)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"token": token,
		"user":  user,
	})
}

func handleGetCurrentUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	userID := getUserID(r)
	user, err := authService.GetUserByID(r.Context(), userID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	jsonResponse(w, user)
}

// ==================== 账户处理 ====================

func handleAccounts(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	switch r.Method {
	case "GET":
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

		accounts, total, err := accountService.ListAccounts(r.Context(), userID, page, pageSize)
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, map[string]interface{}{
			"accounts": accounts,
			"total":    total,
		})

	case "POST":
		var req struct {
			Name           string  `json:"name"`
			Type           string  `json:"type"`
			InitialBalance float64 `json:"initial_balance"`
			Currency       string  `json:"currency"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "无效的请求", http.StatusBadRequest)
			return
		}

		account, err := accountService.CreateAccount(r.Context(), userID, req.Name, req.Type, req.InitialBalance, req.Currency)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		jsonResponse(w, account)

	default:
		jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
	}
}

func handleAccountByID(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/accounts/")
	accountID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, "无效的账户ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		account, err := accountService.GetAccount(r.Context(), userID, accountID)
		if err != nil {
			jsonError(w, err.Error(), http.StatusNotFound)
			return
		}
		jsonResponse(w, account)

	case "PUT":
		var req struct {
			Name string `json:"name"`
			Type string `json:"type"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "无效的请求", http.StatusBadRequest)
			return
		}

		account, err := accountService.UpdateAccount(r.Context(), userID, accountID, req.Name, req.Type)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		jsonResponse(w, account)

	case "DELETE":
		if err := accountService.DeleteAccount(r.Context(), userID, accountID); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		jsonResponse(w, map[string]string{"message": "删除成功"})

	default:
		jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
	}
}

// ==================== 交易处理 ====================

func handleTransactions(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	switch r.Method {
	case "GET":
		accountID, _ := strconv.ParseInt(r.URL.Query().Get("account_id"), 10, 64)
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

		req := &service.ListTransactionsReq{
			AccountID: accountID,
			Type:      r.URL.Query().Get("type"),
			StartDate: r.URL.Query().Get("start_date"),
			EndDate:   r.URL.Query().Get("end_date"),
			Page:      page,
			PageSize:  pageSize,
		}

		transactions, total, err := transactionService.ListTransactions(r.Context(), userID, req)
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, map[string]interface{}{
			"transactions": transactions,
			"total":        total,
		})

	case "POST":
		var req struct {
			AccountID       int64   `json:"account_id"`
			Type            string  `json:"type"`
			Amount          float64 `json:"amount"`
			CategoryID      int64   `json:"category_id"`
			Description     string  `json:"description"`
			TransactionDate string  `json:"transaction_date"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "无效的请求", http.StatusBadRequest)
			return
		}

		transaction, err := transactionService.CreateTransaction(r.Context(), userID, &service.CreateTransactionReq{
			AccountID:       req.AccountID,
			Type:            req.Type,
			Amount:          req.Amount,
			CategoryID:      req.CategoryID,
			Description:     req.Description,
			TransactionDate: req.TransactionDate,
		})
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		jsonResponse(w, transaction)

	default:
		jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
	}
}

func handleTransactionByID(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/transactions/")
	transactionID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, "无效的交易ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		transaction, err := transactionService.GetTransaction(r.Context(), userID, transactionID)
		if err != nil {
			jsonError(w, err.Error(), http.StatusNotFound)
			return
		}
		jsonResponse(w, transaction)

	case "PUT":
		var req struct {
			AccountID       int64   `json:"account_id"`
			Type            string  `json:"type"`
			Amount          float64 `json:"amount"`
			CategoryID      int64   `json:"category_id"`
			Description     string  `json:"description"`
			TransactionDate string  `json:"transaction_date"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "无效的请求", http.StatusBadRequest)
			return
		}

		transaction, err := transactionService.UpdateTransaction(r.Context(), userID, &service.UpdateTransactionReq{
			ID:              transactionID,
			AccountID:       req.AccountID,
			Type:            req.Type,
			Amount:          req.Amount,
			CategoryID:      req.CategoryID,
			Description:     req.Description,
			TransactionDate: req.TransactionDate,
		})
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		jsonResponse(w, transaction)

	case "DELETE":
		if err := transactionService.DeleteTransaction(r.Context(), userID, transactionID); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		jsonResponse(w, map[string]string{"message": "删除成功"})

	default:
		jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
	}
}

// ==================== 分类处理 ====================

func handleCategories(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	switch r.Method {
	case "GET":
		categoryType := r.URL.Query().Get("type")
		categories, err := categoryService.ListCategories(r.Context(), userID, categoryType)
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, map[string]interface{}{
			"categories": categories,
		})

	case "POST":
		var req struct {
			Name  string `json:"name"`
			Type  string `json:"type"`
			Icon  string `json:"icon"`
			Color string `json:"color"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "无效的请求", http.StatusBadRequest)
			return
		}

		category, err := categoryService.CreateCategory(r.Context(), userID, req.Name, req.Type, req.Icon, req.Color)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		jsonResponse(w, category)

	default:
		jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
	}
}

func handleCategoryByID(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/categories/")
	categoryID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, "无效的分类ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "PUT":
		var req struct {
			Name  string `json:"name"`
			Icon  string `json:"icon"`
			Color string `json:"color"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "无效的请求", http.StatusBadRequest)
			return
		}

		category, err := categoryService.UpdateCategory(r.Context(), userID, categoryID, req.Name, req.Icon, req.Color)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		jsonResponse(w, category)

	case "DELETE":
		if err := categoryService.DeleteCategory(r.Context(), userID, categoryID); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		jsonResponse(w, map[string]string{"message": "删除成功"})

	default:
		jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
	}
}

// ==================== 报表处理 ====================

func handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	userID := getUserID(r)
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	summary, categoryStats, dailyStats, err := reportService.GetStats(r.Context(), userID, startDate, endDate)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"summary":        summary,
		"category_stats": categoryStats,
		"daily_stats":    dailyStats,
	})
}

func handleMonthlyReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	userID := getUserID(r)
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	month, _ := strconv.Atoi(r.URL.Query().Get("month"))

	if year == 0 {
		year = time.Now().Year()
	}
	if month == 0 {
		month = int(time.Now().Month())
	}

	report, err := reportService.GetMonthlyReport(r.Context(), userID, year, month)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, report)
}

// ==================== 菜品管理处理 ====================

func handleDishes(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	switch r.Method {
	case "GET":
		category := r.URL.Query().Get("category")
		status := r.URL.Query().Get("status")
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

		dishes, total, err := dishService.ListDishes(r.Context(), userID, category, status, page, pageSize)
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, map[string]interface{}{
			"dishes": dishes,
			"total":  total,
		})

	case "POST":
		var req service.CreateDishReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "无效的请求", http.StatusBadRequest)
			return
		}

		if req.Name == "" {
			jsonError(w, "菜品名称不能为空", http.StatusBadRequest)
			return
		}
		if req.Price <= 0 {
			jsonError(w, "菜品价格必须大于0", http.StatusBadRequest)
			return
		}

		dish, err := dishService.CreateDish(r.Context(), userID, &req)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		jsonResponse(w, dish)

	default:
		jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
	}
}

func handleDishByID(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	// 处理 /api/v1/dishes/categories 路由
	if strings.HasSuffix(r.URL.Path, "/categories") {
		handleDishCategories(w, r)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/dishes/")
	dishID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, "无效的菜品ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		dish, err := dishService.GetDish(r.Context(), userID, dishID)
		if err != nil {
			jsonError(w, err.Error(), http.StatusNotFound)
			return
		}
		jsonResponse(w, dish)

	case "PUT":
		var req service.UpdateDishReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "无效的请求", http.StatusBadRequest)
			return
		}

		dish, err := dishService.UpdateDish(r.Context(), userID, dishID, &req)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		jsonResponse(w, dish)

	case "DELETE":
		if err := dishService.DeleteDish(r.Context(), userID, dishID); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		jsonResponse(w, map[string]string{"message": "删除成功"})

	default:
		jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
	}
}

func handleDishCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	userID := getUserID(r)
	categories, err := dishService.GetDishCategories(r.Context(), userID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"categories": categories,
	})
}

// ==================== 订单管理处理 ====================

func handleOrders(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	switch r.Method {
	case "GET":
		status := r.URL.Query().Get("status")
		orderSource := r.URL.Query().Get("order_source")
		tableID, _ := strconv.ParseInt(r.URL.Query().Get("table_id"), 10, 64)
		startDate := r.URL.Query().Get("start_date")
		endDate := r.URL.Query().Get("end_date")
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

		orders, total, err := orderService.ListOrdersWithFilter(r.Context(), userID, &service.ListOrdersReq{
			Status:      status,
			OrderSource: orderSource,
			TableID:     tableID,
			StartDate:   startDate,
			EndDate:     endDate,
			Page:        page,
			PageSize:    pageSize,
		})
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, map[string]interface{}{
			"orders": orders,
			"total":  total,
		})

	case "POST":
		var req service.CreateOrderReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "无效的请求", http.StatusBadRequest)
			return
		}

		order, err := orderService.CreateOrder(r.Context(), userID, &req)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		jsonResponse(w, order)

	default:
		jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
	}
}

func handleOrderByID(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/orders/")

	// 处理 /api/v1/orders/{id}/status
	if strings.Contains(idStr, "/status") {
		parts := strings.Split(idStr, "/")
		orderID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			jsonError(w, "无效的订单ID", http.StatusBadRequest)
			return
		}

		if r.Method == "PUT" {
			var req struct {
				Status string `json:"status"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				jsonError(w, "无效的请求", http.StatusBadRequest)
				return
			}

			if err := orderService.UpdateOrderStatus(r.Context(), userID, orderID, req.Status); err != nil {
				jsonError(w, err.Error(), http.StatusBadRequest)
				return
			}

			jsonResponse(w, map[string]string{"message": "状态更新成功"})
			return
		}

		jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	orderID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, "无效的订单ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		order, err := orderService.GetOrder(r.Context(), userID, orderID)
		if err != nil {
			jsonError(w, err.Error(), http.StatusNotFound)
			return
		}
		jsonResponse(w, order)

	case "PUT":
		var req service.UpdateOrderReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "无效的请求", http.StatusBadRequest)
			return
		}

		order, err := orderService.UpdateOrder(r.Context(), userID, orderID, &req)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		jsonResponse(w, order)

	case "DELETE":
		if err := orderService.DeleteOrder(r.Context(), userID, orderID); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		jsonResponse(w, map[string]string{"message": "删除成功"})

	default:
		jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
	}
}

// ==================== 图片上传处理 ====================

func handleUploadImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	// 检查 Content-Type
	contentType := r.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "multipart/form-data") {
		// 处理文件上传
		r.ParseMultipartForm(10 << 20) // 10 MB

		file, header, err := r.FormFile("image")
		if err != nil {
			jsonError(w, "请选择要上传的图片", http.StatusBadRequest)
			return
		}
		defer file.Close()

		imagePath, err := uploadService.UploadImage(file, header)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 如果是完整 URL（COS），直接返回；否则加上 /uploads/ 前缀
		imageURL := imagePath
		if !strings.HasPrefix(imagePath, "http://") && !strings.HasPrefix(imagePath, "https://") {
			imageURL = "/uploads/" + imagePath
		}

		jsonResponse(w, map[string]string{
			"path": imagePath,
			"url":  imageURL,
		})
	} else {
		// 处理 Base64 上传
		var req struct {
			Image string `json:"image"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "无效的请求", http.StatusBadRequest)
			return
		}

		if req.Image == "" {
			jsonError(w, "图片数据不能为空", http.StatusBadRequest)
			return
		}

		imagePath, err := uploadService.UploadBase64Image(req.Image)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 如果是完整 URL（COS），直接返回；否则加上 /uploads/ 前缀
		imageURL := imagePath
		if !strings.HasPrefix(imagePath, "http://") && !strings.HasPrefix(imagePath, "https://") {
			imageURL = "/uploads/" + imagePath
		}

		jsonResponse(w, map[string]string{
			"path": imagePath,
			"url":  imageURL,
		})
	}
}

// handleDishReports 处理菜品报表请求
func handleDishReports(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	userID := getUserID(r)
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "monthly"
	}

	// 支持自定义日期范围
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	var startDate, endDate time.Time
	if startDateStr != "" && endDateStr != "" {
		var err error
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			jsonError(w, "无效的开始日期格式", http.StatusBadRequest)
			return
		}
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			jsonError(w, "无效的结束日期格式", http.StatusBadRequest)
			return
		}
	} else {
		startDate, endDate = service.GetPeriodDates(period)
	}

	report, err := dishReportService.GetDishReport(userID, period, startDate, endDate)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, report)
}

// handleDishReportTrend 处理菜品报表趋势数据请求
func handleDishReportTrend(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	userID := getUserID(r)
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "daily"
	}

	countStr := r.URL.Query().Get("count")
	count := 7
	if countStr != "" {
		if c, err := strconv.Atoi(countStr); err == nil && c > 0 && c <= 90 {
			count = c
		}
	}

	trend, err := dishReportService.GetTrendData(userID, period, count)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"period": period,
		"count":  count,
		"data":   trend,
	})
}

// ==================== 餐桌管理处理 ====================

func handleTables(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	switch r.Method {
	case "GET":
		status := r.URL.Query().Get("status")
		tables, err := tableService.ListTables(r.Context(), userID, status)
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, map[string]interface{}{
			"tables": tables,
		})

	case "POST":
		var req struct {
			TableNo  string `json:"table_no"`
			Capacity int    `json:"capacity"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "无效的请求", http.StatusBadRequest)
			return
		}

		table, err := tableService.CreateTable(r.Context(), userID, req.TableNo, req.Capacity)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		jsonResponse(w, table)

	default:
		jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
	}
}

func handleTableByID(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/tables/")

	// 处理 /api/v1/tables/{id}/regenerate
	if strings.HasSuffix(path, "/regenerate") {
		idStr := strings.TrimSuffix(path, "/regenerate")
		tableID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			jsonError(w, "无效的餐桌ID", http.StatusBadRequest)
			return
		}

		if r.Method != "POST" {
			jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
			return
		}

		token, err := tableService.RegenerateToken(r.Context(), userID, tableID)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		jsonResponse(w, map[string]string{
			"token": token,
		})
		return
	}

	// 处理 /api/v1/tables/{id}/qrcode
	if strings.HasSuffix(path, "/qrcode") {
		idStr := strings.TrimSuffix(path, "/qrcode")
		tableID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			jsonError(w, "无效的餐桌ID", http.StatusBadRequest)
			return
		}

		if r.Method != "GET" {
			jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
			return
		}

		token, err := tableService.GetTableToken(r.Context(), userID, tableID)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		jsonResponse(w, map[string]string{
			"token": token,
		})
		return
	}

	// 处理 /api/v1/tables/{id}
	tableID, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		jsonError(w, "无效的餐桌ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		table, err := tableService.GetTableByID(tableID)
		if err != nil {
			jsonError(w, err.Error(), http.StatusNotFound)
			return
		}
		if table.UserID != userID {
			jsonError(w, "无权访问此餐桌", http.StatusForbidden)
			return
		}
		jsonResponse(w, table)

	case "PUT":
		var req struct {
			TableNo  string `json:"table_no"`
			Capacity int    `json:"capacity"`
			Status   string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "无效的请求", http.StatusBadRequest)
			return
		}

		table, err := tableService.UpdateTable(r.Context(), userID, tableID, req.TableNo, req.Capacity, req.Status)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		jsonResponse(w, table)

	case "DELETE":
		if err := tableService.DeleteTable(r.Context(), userID, tableID); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		jsonResponse(w, map[string]string{"message": "删除成功"})

	default:
		jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
	}
}

// ==================== 公开API处理（扫码点单） ====================

func handlePublicMenu(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	// 速率限制
	clientIP := getClientIP(r)
	if allowed, remaining := service.PublicAPIRateLimiter.Allow(clientIP); !allowed {
		jsonError(w, fmt.Sprintf("请求过于频繁，请 %d 秒后再试", remaining), http.StatusTooManyRequests)
		return
	}

	// 获取token
	token := strings.TrimPrefix(r.URL.Path, "/api/v1/public/menu/")
	if token == "" {
		jsonError(w, "缺少二维码令牌", http.StatusBadRequest)
		return
	}

	menu, _, err := publicService.GetPublicMenu(token)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonResponse(w, menu)
}

func handlePublicOrder(w http.ResponseWriter, r *http.Request) {
	// 速率限制
	clientIP := getClientIP(r)
	if allowed, remaining := service.PublicAPIRateLimiter.Allow(clientIP); !allowed {
		jsonError(w, fmt.Sprintf("请求过于频繁，请 %d 秒后再试", remaining), http.StatusTooManyRequests)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/public/order/")
	parts := strings.Split(path, "/")

	if len(parts) < 1 || parts[0] == "" {
		jsonError(w, "缺少二维码令牌", http.StatusBadRequest)
		return
	}

	token := parts[0]

	// POST /api/v1/public/order/{token} - 提交订单
	if r.Method == "POST" && len(parts) == 1 {
		var req service.PublicOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "无效的请求", http.StatusBadRequest)
			return
		}

		order, err := publicService.CreatePublicOrder(token, &req)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		jsonResponse(w, order)
		return
	}

	// GET /api/v1/public/order/{token}/{orderNo} - 查询订单状态
	if r.Method == "GET" && len(parts) == 2 {
		orderNo := parts[1]
		status, err := publicService.GetPublicOrderStatus(token, orderNo)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		jsonResponse(w, status)
		return
	}

	jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
}

// handlePublicTableOrders 获取本桌所有订单
func handlePublicTableOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	// 提取token: /api/v1/public/orders/{token}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/public/orders/")
	token := strings.TrimSuffix(path, "/")

	if token == "" {
		jsonError(w, "缺少token", http.StatusBadRequest)
		return
	}

	orders, err := publicService.GetTableOrders(token)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"orders": orders,
	})
}
