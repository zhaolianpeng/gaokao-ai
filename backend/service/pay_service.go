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

	"gaokao-ai/backend/model"
	"gaokao-ai/backend/repository"
)

type vipProduct struct {
	Description string
	AmountFen   int
}

var vipProducts = map[string]vipProduct{
	"vip_single": {Description: "VIP 次卡", AmountFen: 1},
	"vip_day":    {Description: "VIP 天卡", AmountFen: 1},
	"vip_month":  {Description: "VIP 月卡", AmountFen: 1},
	"vip_season": {Description: "VIP 季卡", AmountFen: 1},
}

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

func (s *PayService) CreatePayment(ctx context.Context, req model.WechatPayRequest) (*model.WechatPayResponse, error) {
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
	userID, err := strconv.Atoi(strings.TrimSpace(req.UserID))
	if err != nil || userID <= 0 {
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
		return nil, fmt.Errorf("create wechat prepay failed: %w", err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode >= 300 {
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
		})
	}
	return &model.WechatPayResponse{
		AmountFen: product.AmountFen,
		Debug: model.WechatPayDebug{
			Nonce:     nonce,
			PrepayID:  payload.PrepayID,
			TimeStamp: timestamp,
		},
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

func (s *PayService) ConfirmPayment(ctx context.Context, req model.WechatPayConfirmRequest) map[string]any {
	if s.adminRepo != nil {
		paidAt := time.Now()
		_ = s.adminRepo.UpsertPaymentOrder(ctx, model.AdminOrder{
			OrderID:        strings.TrimSpace(req.OrderID),
			UserID:         parseUserID(req.UserID),
			ProductID:      strings.TrimSpace(req.ProductID),
			Status:         "paid",
			PaymentChannel: "wechat-pay",
			PaidAt:         &paidAt,
		})
	}
	return map[string]any{
		"confirmedAt":    time.Now().UnixMilli(),
		"ok":             true,
		"orderId":        req.OrderID,
		"paymentChannel": "wechat-pay",
		"productId":      req.ProductID,
	}
}

func parseUserID(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return 0
	}
	return value
}

func (s *PayService) BackfillOrders(ctx context.Context, startDate, endDate string) (*OrderBackfillResult, error) {
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
			result.SkippedDays = append(result.SkippedDays, day.Format("2006-01-02"))
			result.SkippedNotes = append(result.SkippedNotes, fetchErr.Error())
			continue
		}
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
			userID := 0
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
	return result, nil
}

func (s *PayService) fetchTradeBillRows(ctx context.Context, day time.Time) ([]tradeBillRow, error) {
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
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch trade bill %s failed: %s", day.Format("2006-01-02"), strings.TrimSpace(string(body)))
	}
	var payload struct {
		DownloadURL string `json:"download_url"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
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
		return nil, err
	}
	defer downloadResp.Body.Close()
	downloadBody, _ := io.ReadAll(downloadResp.Body)
	if downloadResp.StatusCode >= 300 {
		return nil, fmt.Errorf("download trade bill %s failed: %s", day.Format("2006-01-02"), strings.TrimSpace(string(downloadBody)))
	}
	return parseTradeBillCSV(downloadBody)
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
