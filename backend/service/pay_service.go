package service

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gaokao-ai/backend/logging"
	"gaokao-ai/backend/model"
	"gaokao-ai/backend/repository"
)

type vipProduct struct {
	Description string
	AmountFen   int
}

type membershipWindow struct {
	effectiveFrom  *time.Time
	effectiveUntil *time.Time
	validityType   string
}

var vipProducts = map[string]vipProduct{
	"vip_single": {Description: "VIP 次卡", AmountFen: 1},
	"vip_day":    {Description: "VIP 天卡", AmountFen: 1},
	"vip_month":  {Description: "VIP 月卡", AmountFen: 1},
	"vip_season": {Description: "VIP 季卡", AmountFen: 1},
}

const pendingOrderTTL = 10 * time.Minute
const pendingOrderCloseInterval = time.Second

type PayService struct {
	appID      string
	mchID      string
	certSerial string
	notifyURL  string
	privateKey *rsa.PrivateKey
	httpClient *http.Client
	authRepo   *repository.AuthRepository
	adminRepo  *repository.AdminRepository
}

type OrderBackfillResult struct {
	StartDate    string   `json:"startDate"`
	EndDate      string   `json:"endDate"`
	BillDays     int      `json:"billDays"`
	Processed    int      `json:"processed"`
	SkippedDays  []string `json:"skippedDays"`
	SkippedNotes []string `json:"skippedNotes"`
}

type tradeBillRow struct {
	OrderID       string
	OpenID        string
	TransactionID string
	Status        string
	ProductName   string
	AmountFen     int
	PaidAt        *time.Time
}

var orderIDProductPattern = regexp.MustCompile(`^(.+)_\d{10,}$`)

func NewPayService(appID, mchID, certSerial, privateKeyPath, notifyURL string, authRepo *repository.AuthRepository, adminRepo *repository.AdminRepository) (*PayService, error) {
	privateKey, err := loadMerchantPrivateKey(privateKeyPath)
	if err != nil {
		return nil, err
	}
	return &PayService{
		appID:      strings.TrimSpace(appID),
		mchID:      strings.TrimSpace(mchID),
		certSerial: strings.TrimSpace(certSerial),
		notifyURL:  strings.TrimSpace(notifyURL),
		privateKey: privateKey,
		authRepo:   authRepo,
		adminRepo:  adminRepo,
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}, nil
}

func (s *PayService) StartPendingOrderCloser(ctx context.Context) {
	if s == nil || s.adminRepo == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(pendingOrderCloseInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.adminRepo.CloseExpiredCreatedOrders(context.Background(), time.Now()); err != nil {
					logging.LogEvent("pay_close_expired_orders", map[string]any{"status": "failed", "error": err.Error()})
				}
			}
		}
	}()
}

func (s *PayService) CreatePayment(ctx context.Context, req model.WechatPayRequest) (*model.WechatPayResponse, error) {
	logging.LogEvent("pay_create_start", map[string]any{"orderId": strings.TrimSpace(req.OrderID), "userId": strings.TrimSpace(req.UserID), "productId": strings.TrimSpace(req.ProductID), "hasOpenID": strings.TrimSpace(req.OpenID) != ""})
	if s.adminRepo != nil {
		if err := s.adminRepo.CloseExpiredCreatedOrders(ctx, time.Now()); err != nil {
			return nil, fmt.Errorf("close expired pending orders failed: %w", err)
		}
	}
	product, ok := vipProducts[strings.TrimSpace(req.ProductID)]
	if s.adminRepo != nil {
		configured, err := s.adminRepo.GetVIPProductByProductID(ctx, strings.TrimSpace(req.ProductID))
		if err == nil && configured != nil {
			if !configured.Enabled {
				return nil, fmt.Errorf("vip product disabled")
			}
			product = vipProduct{Description: configured.Description, AmountFen: configured.AmountFen}
			ok = true
		}
	}
	if !ok {
		return nil, fmt.Errorf("invalid vip product")
	}
	userID, err := s.authRepo.ResolveUserID(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id")
	}
	userRecord, err := s.authRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load user failed: %w", err)
	}
	openid := strings.TrimSpace(req.OpenID)
	if openid == "" {
		openid = strings.TrimSpace(userRecord.OpenID)
	}
	if openid == "" {
		return nil, fmt.Errorf("missing openid")
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := randomAlphaNum(24)
	requestBody, err := json.Marshal(map[string]any{
		"appid":        s.appID,
		"mchid":        s.mchID,
		"description":  product.Description,
		"notify_url":   s.notifyURL,
		"out_trade_no": req.OrderID,
		"amount": map[string]any{
			"total":    product.AmountFen,
			"currency": "CNY",
		},
		"payer": map[string]any{
			"openid": openid,
		},
	})
	if err != nil {
		return nil, err
	}
	authorization, err := s.buildAuthorization(http.MethodPost, "/v3/pay/transactions/jsapi", timestamp, nonce, string(requestBody))
	if err != nil {
		return nil, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.mch.weixin.qq.com/v3/pay/transactions/jsapi", bytes.NewReader(requestBody))
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Authorization", authorization)
	httpRequest.Header.Set("User-Agent", "gaokao-api/1.0")
	response, err := s.httpClient.Do(httpRequest)
	if err != nil {
		logging.LogEvent("pay_wechat_request", map[string]any{"orderId": strings.TrimSpace(req.OrderID), "productId": strings.TrimSpace(req.ProductID), "status": "failed", "error": err.Error()})
		return nil, fmt.Errorf("create wechat prepay failed: %w", err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode >= 300 {
		logging.LogEvent("pay_wechat_request", map[string]any{"orderId": strings.TrimSpace(req.OrderID), "productId": strings.TrimSpace(req.ProductID), "statusCode": response.StatusCode, "status": "failed", "responsePreview": logging.PreviewString(strings.TrimSpace(string(body)), 512)})
		return nil, fmt.Errorf("create wechat prepay failed: %s", strings.TrimSpace(string(body)))
	}
	var payload struct {
		PrepayID string `json:"prepay_id"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse wechat prepay failed: %w", err)
	}
	if strings.TrimSpace(payload.PrepayID) == "" {
		return nil, fmt.Errorf("create wechat prepay failed: missing prepay_id")
	}
	paymentNonce := randomAlphaNum(24)
	paymentPackage := "prepay_id=" + payload.PrepayID
	paymentSign, err := s.signPayment(timestamp, paymentNonce, paymentPackage)
	if err != nil {
		return nil, err
	}
	if s.adminRepo != nil {
		expiresAt := time.Now().Add(pendingOrderTTL)
		_ = s.adminRepo.UpsertPaymentOrder(ctx, model.AdminOrder{
			OrderID:        strings.TrimSpace(req.OrderID),
			UserID:         userID,
			OpenID:         openid,
			ProductID:      strings.TrimSpace(req.ProductID),
			ProductName:    product.Description,
			Content:        product.Description,
			AmountFen:      product.AmountFen,
			Status:         "created",
			PaymentChannel: "wechat-pay",
			PrepayID:       strings.TrimSpace(payload.PrepayID),
			ExpiresAt:      &expiresAt,
		})
	}
	logging.LogEvent("pay_create_complete", map[string]any{"orderId": strings.TrimSpace(req.OrderID), "userId": userID, "productId": strings.TrimSpace(req.ProductID), "amountFen": product.AmountFen, "prepayId": strings.TrimSpace(payload.PrepayID), "status": "created"})
	return &model.WechatPayResponse{
		AmountFen: product.AmountFen,
		Debug: model.WechatPayDebug{
			Nonce:     nonce,
			PrepayID:  payload.PrepayID,
			TimeStamp: timestamp,
		},
		ExpiresAt: time.Now().Add(pendingOrderTTL).UnixMilli(),
		OrderID:   req.OrderID,
		ProductID: req.ProductID,
		Payment: model.WechatPaymentParams{
			AppID:     s.appID,
			TimeStamp: timestamp,
			NonceStr:  paymentNonce,
			Package:   paymentPackage,
			SignType:  "RSA",
			PaySign:   paymentSign,
		},
	}, nil
}

func (s *PayService) ConfirmPayment(ctx context.Context, req model.WechatPayConfirmRequest) (map[string]any, error) {
	logging.LogEvent("pay_confirm", map[string]any{"orderId": strings.TrimSpace(req.OrderID), "userId": strings.TrimSpace(req.UserID), "productId": strings.TrimSpace(req.ProductID), "status": "start"})
	resolvedUserID := strings.TrimSpace(req.UserID)
	if s.authRepo != nil {
		if value, err := s.authRepo.ResolveUserID(ctx, req.UserID); err == nil {
			resolvedUserID = value
		}
	}
	if s.adminRepo == nil {
		return nil, fmt.Errorf("pay service unavailable")
	}
	if err := s.adminRepo.CloseExpiredCreatedOrders(ctx, time.Now()); err != nil {
		return nil, fmt.Errorf("close expired pending orders failed: %w", err)
	}
	order, err := s.adminRepo.GetPaymentOrderByOrderID(ctx, req.OrderID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("order not found")
		}
		return nil, err
	}
	if strings.TrimSpace(order.UserID) != "" && strings.TrimSpace(order.UserID) != resolvedUserID {
		return nil, fmt.Errorf("order user mismatch")
	}
	if order.Status == "closed" {
		return nil, fmt.Errorf("当前订单已超时关闭，请重新创建订单后支付")
	}
	if order.Status == "paid" {
		return map[string]any{
			"confirmedAt":    time.Now().UnixMilli(),
			"ok":             true,
			"orderId":        req.OrderID,
			"paymentChannel": "wechat-pay",
			"productId":      req.ProductID,
		}, nil
	}
	if s.adminRepo != nil {
		paidAt := time.Now()
		configuredProduct, _ := s.adminRepo.GetVIPProductByProductID(ctx, strings.TrimSpace(req.ProductID))
		window := s.buildMembershipWindow(ctx, resolvedUserID, strings.TrimSpace(req.ProductID), configuredProduct, paidAt)
		_ = s.adminRepo.UpsertPaymentOrder(ctx, model.AdminOrder{
			OrderID:        strings.TrimSpace(req.OrderID),
			UserID:         resolvedUserID,
			ProductID:      strings.TrimSpace(req.ProductID),
			Status:         "paid",
			PaymentChannel: "wechat-pay",
			PaidAt:         &paidAt,
			ExpiresAt:      order.ExpiresAt,
			EffectiveFrom:  window.effectiveFrom,
			EffectiveUntil: window.effectiveUntil,
		})
	}
	logging.LogEvent("pay_confirm", map[string]any{"orderId": strings.TrimSpace(req.OrderID), "userId": resolvedUserID, "productId": strings.TrimSpace(req.ProductID), "status": "paid"})
	return map[string]any{
		"confirmedAt":    time.Now().UnixMilli(),
		"ok":             true,
		"orderId":        req.OrderID,
		"paymentChannel": "wechat-pay",
		"productId":      req.ProductID,
	}, nil
}

func (s *PayService) GetMembership(ctx context.Context, userID string) (*model.VIPMembershipStatusResponse, error) {
	if s.adminRepo == nil {
		return nil, fmt.Errorf("membership service unavailable")
	}
	resolvedUserID := strings.TrimSpace(userID)
	if s.authRepo != nil {
		value, err := s.authRepo.ResolveUserID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("invalid user id")
		}
		resolvedUserID = value
	}
	if resolvedUserID == "" {
		return nil, fmt.Errorf("invalid user id")
	}
	order, err := s.adminRepo.GetLatestPaidOrderByUserID(ctx, resolvedUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return &model.VIPMembershipStatusResponse{}, nil
		}
		return nil, err
	}
	var product *model.VIPProductConfig
	if strings.TrimSpace(order.ProductID) != "" {
		configured, productErr := s.adminRepo.GetVIPProductByProductID(ctx, strings.TrimSpace(order.ProductID))
		if productErr == nil {
			product = configured
		}
	}
	status := s.adminRepo.InferVIPMembership(*order, product, time.Now())
	return &status, nil
}

func (s *PayService) buildMembershipWindow(ctx context.Context, userID, productID string, product *model.VIPProductConfig, paidAt time.Time) membershipWindow {
	validityType := ""
	if product != nil {
		validityType = strings.TrimSpace(product.ValidityType)
	}
	if validityType == "" || validityType == "unlimited" {
		switch strings.TrimSpace(productID) {
		case "vip_day", "vip_month", "vip_season":
			validityType = "range"
		case "vip_single":
			validityType = "times"
		default:
			validityType = "unlimited"
		}
	}
	window := membershipWindow{validityType: validityType}
	if validityType != "range" {
		return window
	}
	baseStart := paidAt
	baseEnd := paidAt
	currentMembership, err := s.GetMembership(ctx, userID)
	if err == nil && currentMembership != nil && currentMembership.Active && currentMembership.EndAt > paidAt.UnixMilli() {
		baseStart = time.UnixMilli(currentMembership.StartAt)
		baseEnd = time.UnixMilli(currentMembership.EndAt)
	}
	duration := s.resolveProductRangeDuration(productID, product, paidAt)
	if duration <= 0 {
		return window
	}
	effectiveFrom := baseStart
	effectiveUntil := baseEnd.Add(duration)
	if baseEnd.Before(paidAt) {
		effectiveFrom = paidAt
		effectiveUntil = paidAt.Add(duration)
	}
	window.effectiveFrom = &effectiveFrom
	window.effectiveUntil = &effectiveUntil
	return window
}

func (s *PayService) resolveProductRangeDuration(productID string, product *model.VIPProductConfig, reference time.Time) time.Duration {
	if product != nil && strings.TrimSpace(product.ValidityType) == "range" && product.ValidFrom != nil && product.ValidUntil != nil && product.ValidUntil.After(*product.ValidFrom) {
		return product.ValidUntil.Sub(*product.ValidFrom)
	}
	switch strings.TrimSpace(productID) {
	case "vip_day":
		return 24 * time.Hour
	case "vip_month":
		return time.Time{}.AddDate(0, 0, 30).Sub(time.Time{})
	case "vip_season":
		return time.Time{}.AddDate(0, 0, 90).Sub(time.Time{})
	default:
		_ = reference
		return 0
	}
}

func (s *PayService) BackfillOrders(ctx context.Context, startDate, endDate string) (*OrderBackfillResult, error) {
	logging.LogEvent("pay_backfill_start", map[string]any{"startDate": strings.TrimSpace(startDate), "endDate": strings.TrimSpace(endDate)})
	if s.adminRepo == nil || s.authRepo == nil {
		return nil, fmt.Errorf("order backfill unavailable")
	}
	start, err := parseBillDate(startDate)
	if err != nil {
		return nil, err
	}
	end, err := parseBillDate(endDate)
	if err != nil {
		return nil, err
	}
	if end.Before(start) {
		return nil, fmt.Errorf("end date must be greater than or equal to start date")
	}
	result := &OrderBackfillResult{
		StartDate:   start.Format("2006-01-02"),
		EndDate:     end.Format("2006-01-02"),
		SkippedDays: make([]string, 0),
	}
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		rows, fetchErr := s.fetchTradeBillRows(ctx, day)
		if fetchErr != nil {
			logging.LogEvent("pay_backfill_day", map[string]any{"billDate": day.Format("2006-01-02"), "status": "skipped", "error": fetchErr.Error()})
			result.SkippedDays = append(result.SkippedDays, day.Format("2006-01-02"))
			result.SkippedNotes = append(result.SkippedNotes, fetchErr.Error())
			continue
		}
		logging.LogEvent("pay_backfill_day", map[string]any{"billDate": day.Format("2006-01-02"), "status": "fetched", "rowCount": len(rows)})
		result.BillDays++
		for _, row := range rows {
			productID := parseProductIDFromOrderID(row.OrderID)
			if productID == "" {
				continue
			}
			productName := strings.TrimSpace(row.ProductName)
			amountFen := row.AmountFen
			if configured, configErr := s.adminRepo.GetVIPProductByProductID(ctx, productID); configErr == nil && configured != nil {
				if productName == "" {
					productName = configured.Name
				}
				if amountFen <= 0 {
					amountFen = configured.AmountFen
				}
			}
			userID := ""
			if strings.TrimSpace(row.OpenID) != "" {
				if user, userErr := s.authRepo.GetUserByOpenID(ctx, strings.TrimSpace(row.OpenID)); userErr == nil && user != nil {
					userID = user.ID
				} else if userErr != nil && userErr != sql.ErrNoRows {
					return nil, userErr
				}
			}
			if err := s.adminRepo.UpsertPaymentOrder(ctx, model.AdminOrder{
				OrderID:        row.OrderID,
				UserID:         userID,
				OpenID:         strings.TrimSpace(row.OpenID),
				ProductID:      productID,
				ProductName:    firstNonEmpty(productName, productID),
				Content:        firstNonEmpty(productName, productID),
				AmountFen:      amountFen,
				Status:         row.Status,
				PaymentChannel: "wechat-pay",
				TransactionID:  strings.TrimSpace(row.TransactionID),
				PaidAt:         row.PaidAt,
			}); err != nil {
				return nil, err
			}
			result.Processed++
		}
	}
	logging.LogEvent("pay_backfill_complete", map[string]any{"startDate": result.StartDate, "endDate": result.EndDate, "billDays": result.BillDays, "processed": result.Processed, "skippedDays": result.SkippedDays})
	return result, nil
}

func (s *PayService) fetchTradeBillRows(ctx context.Context, day time.Time) ([]tradeBillRow, error) {
	logging.LogEvent("pay_bill_fetch", map[string]any{"billDate": day.Format("2006-01-02"), "status": "start"})
	query := url.Values{}
	query.Set("bill_date", day.Format("2006-01-02"))
	query.Set("bill_type", "ALL")
	path := "/v3/bill/tradebill?" + query.Encode()
	authorization, err := s.buildAuthorization(http.MethodGet, path, strconv.FormatInt(time.Now().Unix(), 10), randomAlphaNum(24), "")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.mch.weixin.qq.com"+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", authorization)
	req.Header.Set("User-Agent", "gaokao-api/1.0")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		logging.LogEvent("pay_bill_fetch", map[string]any{"billDate": day.Format("2006-01-02"), "status": "failed", "error": err.Error()})
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		logging.LogEvent("pay_bill_fetch", map[string]any{"billDate": day.Format("2006-01-02"), "status": "failed", "statusCode": resp.StatusCode, "responsePreview": logging.PreviewString(strings.TrimSpace(string(body)), 512)})
		return nil, fmt.Errorf("fetch trade bill %s failed: %s", day.Format("2006-01-02"), strings.TrimSpace(string(body)))
	}
	var payload struct {
		DownloadURL string `json:"download_url"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		logging.LogEvent("pay_bill_fetch", map[string]any{"billDate": day.Format("2006-01-02"), "status": "failed", "error": err.Error(), "responsePreview": logging.PreviewString(strings.TrimSpace(string(body)), 512)})
		return nil, fmt.Errorf("parse trade bill %s failed: %w", day.Format("2006-01-02"), err)
	}
	if strings.TrimSpace(payload.DownloadURL) == "" {
		return nil, fmt.Errorf("trade bill %s missing download url", day.Format("2006-01-02"))
	}
	downloadReq, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSpace(payload.DownloadURL), nil)
	if err != nil {
		return nil, err
	}
	downloadResp, err := s.httpClient.Do(downloadReq)
	if err != nil {
		logging.LogEvent("pay_bill_download", map[string]any{"billDate": day.Format("2006-01-02"), "status": "failed", "error": err.Error()})
		return nil, err
	}
	defer downloadResp.Body.Close()
	downloadBody, _ := io.ReadAll(downloadResp.Body)
	if downloadResp.StatusCode >= 300 {
		logging.LogEvent("pay_bill_download", map[string]any{"billDate": day.Format("2006-01-02"), "status": "failed", "statusCode": downloadResp.StatusCode, "responsePreview": logging.PreviewString(strings.TrimSpace(string(downloadBody)), 512)})
		return nil, fmt.Errorf("download trade bill %s failed: %s", day.Format("2006-01-02"), strings.TrimSpace(string(downloadBody)))
	}
	rows, parseErr := parseTradeBillCSV(downloadBody)
	if parseErr != nil {
		logging.LogEvent("pay_bill_download", map[string]any{"billDate": day.Format("2006-01-02"), "status": "failed", "error": parseErr.Error()})
		return nil, parseErr
	}
	logging.LogEvent("pay_bill_download", map[string]any{"billDate": day.Format("2006-01-02"), "status": "success", "rowCount": len(rows)})
	return rows, nil
}

func parseTradeBillCSV(body []byte) ([]tradeBillRow, error) {
	reader := csv.NewReader(strings.NewReader(strings.TrimSpace(strings.TrimPrefix(string(body), "\ufeff"))))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	headerIdx := -1
	for index, record := range records {
		if len(record) == 0 {
			continue
		}
		joined := strings.Join(record, ",")
		if strings.Contains(joined, "商户订单号") && strings.Contains(joined, "用户标识") {
			headerIdx = index
			break
		}
	}
	if headerIdx < 0 || headerIdx >= len(records)-1 {
		return []tradeBillRow{}, nil
	}
	headers := records[headerIdx]
	indexMap := map[string]int{}
	for idx, header := range headers {
		indexMap[strings.TrimSpace(header)] = idx
	}
	items := make([]tradeBillRow, 0)
	for _, record := range records[headerIdx+1:] {
		if len(record) == 0 {
			continue
		}
		first := strings.TrimSpace(record[0])
		if first == "" || strings.Contains(first, "总交易单数") || strings.Contains(first, "总退款金额") {
			continue
		}
		orderID := csvValue(record, indexMap, "商户订单号")
		if orderID == "" {
			continue
		}
		status := normalizeTradeState(csvValue(record, indexMap, "交易状态"))
		amountFen := parseAmountFen(csvValue(record, indexMap, "总金额"))
		paidAt := parseTradeBillTime(csvValue(record, indexMap, "交易时间"))
		items = append(items, tradeBillRow{
			OrderID:       orderID,
			OpenID:        csvValue(record, indexMap, "用户标识"),
			TransactionID: csvValue(record, indexMap, "微信订单号"),
			Status:        status,
			ProductName:   csvValue(record, indexMap, "商品名称"),
			AmountFen:     amountFen,
			PaidAt:        paidAt,
		})
	}
	return items, nil
}

func csvValue(record []string, indexMap map[string]int, key string) string {
	idx, ok := indexMap[key]
	if !ok || idx < 0 || idx >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[idx])
}

func parseBillDate(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, fmt.Errorf("missing date")
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q", value)
	}
	return parsed, nil
}

func parseTradeBillTime(raw string) *time.Time {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	for _, layout := range []string{"2006-01-02 15:04:05", time.RFC3339} {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return &parsed
		}
	}
	return nil
}

func parseAmountFen(raw string) int {
	value := strings.ReplaceAll(strings.TrimSpace(raw), ",", "")
	if value == "" {
		return 0
	}
	amount, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return int(math.Round(amount * 100))
}

func normalizeTradeState(raw string) string {
	value := strings.TrimSpace(raw)
	switch {
	case strings.Contains(value, "成功"):
		return "paid"
	case strings.Contains(value, "退款"):
		return "refunded"
	case strings.Contains(value, "关闭"), strings.Contains(value, "撤销"):
		return "closed"
	default:
		return "created"
	}
}

func parseProductIDFromOrderID(orderID string) string {
	matched := orderIDProductPattern.FindStringSubmatch(strings.TrimSpace(orderID))
	if len(matched) != 2 {
		return ""
	}
	return strings.TrimSpace(matched[1])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (s *PayService) buildAuthorization(method, path, timestamp, nonce, body string) (string, error) {
	message := method + "\n" + path + "\n" + timestamp + "\n" + nonce + "\n" + body + "\n"
	digest := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.privateKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign wechat pay request failed: %w", err)
	}
	return fmt.Sprintf(`WECHATPAY2-SHA256-RSA2048 mchid="%s",nonce_str="%s",timestamp="%s",serial_no="%s",signature="%s"`, s.mchID, nonce, timestamp, s.certSerial, base64.StdEncoding.EncodeToString(signature)), nil
}

func (s *PayService) signPayment(timestamp, nonce, pkg string) (string, error) {
	message := s.appID + "\n" + timestamp + "\n" + nonce + "\n" + pkg + "\n"
	digest := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.privateKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign wechat pay params failed: %w", err)
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func loadMerchantPrivateKey(path string) (*rsa.PrivateKey, error) {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return nil, fmt.Errorf("missing wechat merchant private key path")
	}
	body, err := os.ReadFile(trimmedPath)
	if err != nil {
		return nil, fmt.Errorf("read wechat merchant private key failed: %w", err)
	}
	block, _ := pem.Decode(body)
	if block == nil {
		return nil, fmt.Errorf("decode wechat merchant private key failed")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		parsedKey, pkcs1Err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if pkcs1Err == nil {
			return parsedKey, nil
		}
		return nil, fmt.Errorf("parse wechat merchant private key failed: %w", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("wechat merchant private key is not rsa")
	}
	return rsaKey, nil
}

func randomAlphaNum(length int) string {
	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	if length <= 0 {
		return ""
	}
	buffer := make([]byte, length)
	randomBytes := make([]byte, length)
	_, _ = rand.Read(randomBytes)
	for index := 0; index < length; index++ {
		buffer[index] = alphabet[int(randomBytes[index])%len(alphabet)]
	}
	return string(buffer)
}
