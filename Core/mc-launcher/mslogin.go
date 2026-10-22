package core

// mslogin.go — 微软设备码登录的两步接口（给 Web 前端用）
import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
)

// ErrAuthPending 用户尚未完成授权，需要继续轮询
var ErrAuthPending = errors.New("authorization_pending")

// DeviceCode 设备码信息
type DeviceCode struct {
	DeviceCode      string `json:"deviceCode"`
	UserCode        string `json:"userCode"`
	VerificationURI string `json:"verificationUri"`
	ExpiresIn       int    `json:"expiresIn"`
	Interval        int    `json:"interval"`
	ClientID        string `json:"clientId"`
}

func msClientID(id string) string {
	if id != "" {
		return id
	}
	if v := os.Getenv("MC_MS_CLIENT_ID"); v != "" {
		return v
	}
	return "00000000402b5328"
}

// StartMicrosoftLogin 申请设备码，前端把 userCode 展示给用户
func StartMicrosoftLogin(clientID string) (*DeviceCode, error) {
	cid := msClientID(clientID)

	form := url.Values{}
	form.Set("client_id", cid)
	form.Set("scope", "XboxLive.signin offline_access")

	b, code, err := httpPostForm("https://login.microsoftonline.com/consumers/oauth2/v2.0/devicecode", form.Encode())
	if err != nil {
		return nil, fmt.Errorf("获取设备码失败: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("获取设备码失败: HTTP %d", code)
	}

	var raw struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		ExpiresIn       int    `json:"expires_in"`
		Interval        int    `json:"interval"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("解析设备码失败: %v", err)
	}
	if raw.DeviceCode == "" {
		return nil, fmt.Errorf("未获取到设备码")
	}

	interval := raw.Interval
	if interval <= 0 {
		interval = 5
	}
	expires := raw.ExpiresIn
	if expires <= 0 {
		expires = 900
	}

	return &DeviceCode{
		DeviceCode:      raw.DeviceCode,
		UserCode:        raw.UserCode,
		VerificationURI: raw.VerificationURI,
		ExpiresIn:       expires,
		Interval:        interval,
		ClientID:        cid,
	}, nil
}

// PollMicrosoftLogin 轮询一次。
// 返回 (nil, ErrAuthPending) 表示用户还没授权，调用方应继续等待后重试。
func PollMicrosoftLogin(deviceCode, clientID string) (*Account, error) {
	if deviceCode == "" {
		return nil, fmt.Errorf("deviceCode 不能为空")
	}
	cid := msClientID(clientID)

	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	form.Set("client_id", cid)
	form.Set("device_code", deviceCode)

	b, _, err := httpPostForm("https://login.microsoftonline.com/consumers/oauth2/v2.0/token", form.Encode())
	if err != nil {
		return nil, fmt.Errorf("轮询授权状态失败: %v", err)
	}

	var res map[string]any
	if err := json.Unmarshal(b, &res); err != nil {
		return nil, fmt.Errorf("解析授权响应失败: %v", err)
	}

	if tok, ok := res["access_token"].(string); ok && tok != "" {
		return exchangeForAccount(tok, false)
	}

	errCode, _ := res["error"].(string)
	switch errCode {
	case "authorization_pending":
		return nil, ErrAuthPending
	case "slow_down":
		return nil, ErrAuthPending
	case "":
		return nil, fmt.Errorf("微软返回未知响应")
	case "authorization_declined":
		return nil, fmt.Errorf("用户拒绝了授权")
	case "expired_token":
		return nil, fmt.Errorf("设备码已过期，请重新开始登录")
	default:
		desc, _ := res["error_description"].(string)
		if desc == "" {
			desc = errCode
		}
		return nil, fmt.Errorf("微软登录失败: %s", desc)
	}
}
