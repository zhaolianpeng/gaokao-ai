package service

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
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
}

func NewPayService(appID, mchID, certSerial, privateKeyPath, notifyURL string, authRepo *repository.AuthRepository) (*PayService, error) {
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
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}, nil
}

func (s *PayService) CreatePayment(ctx context.Context, req model.WechatPayRequest) (*model.WechatPayResponse, error) {
	product, ok := vipProducts[strings.TrimSpace(req.ProductID)]
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

func (s *PayService) ConfirmPayment(_ context.Context, req model.WechatPayConfirmRequest) map[string]any {
	return map[string]any{
		"confirmedAt":    time.Now().UnixMilli(),
		"ok":             true,
		"orderId":        req.OrderID,
		"paymentChannel": "wechat-pay",
		"productId":      req.ProductID,
	}
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
