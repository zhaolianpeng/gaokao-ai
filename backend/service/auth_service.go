package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"gaokao-ai/backend/model"
	"gaokao-ai/backend/repository"
)

type AuthService struct {
	appID     string
	appSecret string
	repo      *repository.AuthRepository
	client    *http.Client

	tokenMu          sync.Mutex
	accessToken      string
	accessTokenUntil time.Time
}

func NewAuthService(appID, appSecret string, repo *repository.AuthRepository) *AuthService {
	return &AuthService{
		appID:     strings.TrimSpace(appID),
		appSecret: strings.TrimSpace(appSecret),
		repo:      repo,
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (s *AuthService) Login(ctx context.Context, req model.WechatLoginRequest) (*model.AuthUser, error) {
	openid, err := s.exchangeLoginCode(ctx, req.LoginCode)
	if err != nil {
		return nil, err
	}
	phone, err := s.fetchPhoneNumber(ctx, req.Code)
	if err != nil {
		return nil, err
	}
	userRecord, created, err := s.repo.UpsertWechatUser(ctx, openid, phone)
	if err != nil {
		return nil, err
	}
	return toAuthUser(userRecord, created), nil
}

func (s *AuthService) UpdateProfile(ctx context.Context, req model.WechatProfileUpdateRequest) (*model.AuthUser, error) {
	userID, err := strconv.Atoi(strings.TrimSpace(req.UserID))
	if err != nil || userID <= 0 {
		return nil, fmt.Errorf("invalid user id")
	}
	if req.Phone != nil {
		trimmed := strings.TrimSpace(*req.Phone)
		req.Phone = &trimmed
	}
	req.Nickname = strings.TrimSpace(req.Nickname)
	if req.AvatarURL != nil {
		trimmed := strings.TrimSpace(*req.AvatarURL)
		req.AvatarURL = &trimmed
	}
	if req.IDCard != nil {
		trimmed := strings.ToUpper(strings.TrimSpace(*req.IDCard))
		req.IDCard = &trimmed
	}
	if req.SchoolName != nil {
		trimmed := strings.TrimSpace(*req.SchoolName)
		req.SchoolName = &trimmed
	}
	if req.SchoolYear != nil {
		trimmed := strings.TrimSpace(*req.SchoolYear)
		req.SchoolYear = &trimmed
	}
	if req.ClassName != nil {
		trimmed := strings.TrimSpace(*req.ClassName)
		req.ClassName = &trimmed
	}
	if req.StudentNo != nil {
		trimmed := strings.TrimSpace(*req.StudentNo)
		req.StudentNo = &trimmed
	}
	userRecord, err := s.repo.UpdateProfile(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	return toAuthUser(userRecord, false), nil
}

func (s *AuthService) GetUserByID(ctx context.Context, userID string) (*model.AuthUser, error) {
	parsedID, err := strconv.Atoi(strings.TrimSpace(userID))
	if err != nil || parsedID <= 0 {
		return nil, fmt.Errorf("invalid user id")
	}
	record, err := s.repo.GetUserByID(ctx, parsedID)
	if err != nil {
		return nil, err
	}
	return toAuthUser(record, false), nil
}

func (s *AuthService) exchangeLoginCode(ctx context.Context, loginCode string) (string, error) {
	if s.appID == "" || s.appSecret == "" {
		return "", fmt.Errorf("未配置微信登录参数")
	}
	endpoint := "https://api.weixin.qq.com/sns/jscode2session?appid=" + url.QueryEscape(s.appID) + "&secret=" + url.QueryEscape(s.appSecret) + "&js_code=" + url.QueryEscape(loginCode) + "&grant_type=authorization_code"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	response, err := s.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("微信登录失败：%w", err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	var payload struct {
		OpenID  string `json:"openid"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("微信登录失败：%w", err)
	}
	if payload.ErrCode != 0 {
		return "", fmt.Errorf("微信登录失败：%s", payload.ErrMsg)
	}
	if strings.TrimSpace(payload.OpenID) == "" {
		return "", fmt.Errorf("微信登录失败：missing openid")
	}
	return payload.OpenID, nil
}

func (s *AuthService) fetchPhoneNumber(ctx context.Context, code string) (string, error) {
	accessToken, err := s.getAccessToken(ctx)
	if err != nil {
		return "", err
	}
	body, err := json.Marshal(map[string]string{"code": code})
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.weixin.qq.com/wxa/business/getuserphonenumber?access_token="+url.QueryEscape(accessToken), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := s.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("获取微信手机号失败：%w", err)
	}
	defer response.Body.Close()
	responseBody, _ := io.ReadAll(response.Body)
	var payload struct {
		PhoneInfo struct {
			PurePhoneNumber string `json:"purePhoneNumber"`
			PhoneNumber     string `json:"phoneNumber"`
		} `json:"phone_info"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return "", fmt.Errorf("获取微信手机号失败：%w", err)
	}
	if payload.ErrCode != 0 {
		return "", fmt.Errorf("获取微信手机号失败：%s", payload.ErrMsg)
	}
	phone := strings.TrimSpace(payload.PhoneInfo.PurePhoneNumber)
	if phone == "" {
		phone = strings.TrimSpace(payload.PhoneInfo.PhoneNumber)
	}
	return phone, nil
}

func (s *AuthService) getAccessToken(ctx context.Context) (string, error) {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()
	if s.accessToken != "" && time.Now().Before(s.accessTokenUntil) {
		return s.accessToken, nil
	}
	endpoint := "https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=" + url.QueryEscape(s.appID) + "&secret=" + url.QueryEscape(s.appSecret)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	response, err := s.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("获取微信 access_token 失败：%w", err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	var payload struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("获取微信 access_token 失败：%w", err)
	}
	if payload.ErrCode != 0 {
		return "", fmt.Errorf("获取微信 access_token 失败：%s", payload.ErrMsg)
	}
	s.accessToken = strings.TrimSpace(payload.AccessToken)
	expiresIn := payload.ExpiresIn
	if expiresIn <= 300 {
		expiresIn = 300
	}
	s.accessTokenUntil = time.Now().Add(time.Duration(expiresIn-120) * time.Second)
	return s.accessToken, nil
}

func toAuthUser(record *model.AuthUserRecord, created bool) *model.AuthUser {
	if record == nil {
		return nil
	}
	return &model.AuthUser{
		ID:            strconv.Itoa(record.ID),
		OpenID:        record.OpenID,
		Phone:         record.Phone,
		Nickname:      record.Nickname,
		AvatarURL:     record.AvatarURL,
		IDCard:        record.IDCard,
		SchoolName:    record.SchoolName,
		SchoolYear:    record.SchoolYear,
		ClassName:     record.ClassName,
		StudentNo:     record.StudentNo,
		FromRecommend: record.FromRecommend,
		LoginType:     record.LoginType,
		StorageMode:   "server",
		Created:       created,
		CreatedAt:     record.CreatedAt.UnixMilli(),
		UpdatedAt:     record.UpdatedAt.UnixMilli(),
		LastLoginAt:   record.LastLoginAt.UnixMilli(),
	}
}
