package core

import (
	"net/http"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"
)

func offlineUuid(name string) string {
	h := md5.Sum([]byte("OfflinePlayer:" + name))
	b := h[:]
	b[6] = (b[6] & 0x0F) | 0x30
	b[8] = (b[8] & 0x3F) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func loginOffline(name string) *Account {
	if name == "" {
		name = "Player"
	}
	uuid := offlineUuid(name)
	return &Account{Type: "offline", Name: name, UUID: uuid, AccessToken: strings.ReplaceAll(uuid, "-", "")}
}

func loginMicrosoft(verbose bool, clientId string) (*Account, error) {
	if clientId == "" {
		clientId = os.Getenv("MC_MS_CLIENT_ID")
	}
	if clientId == "" {
		clientId = "00000000402b5328"
	}

	// 步骤 1: 获取设备码 (使用 form-urlencoded)
	deviceBody := url.Values{}
	deviceBody.Set("client_id", clientId)
	deviceBody.Set("scope", "XboxLive.signin offline_access")
	
	b, code, err := httpPostForm("https://login.microsoftonline.com/consumers/oauth2/v2.0/devicecode", deviceBody.Encode())
	if err != nil || code != 200 {
		return nil, fmt.Errorf("获取设备码失败: %v", err)
	}
	var dc struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationUri string `json:"verification_uri"`
		Interval        int    `json:"interval"`
	}
	if err := json.Unmarshal(b, &dc); err != nil {
		return nil, fmt.Errorf("解析设备码失败: %v", err)
	}
	if dc.DeviceCode == "" {
		return nil, fmt.Errorf("未获取到设备码")
	}
	if verbose {
		fmt.Printf("请访问: %s\n输入代码: %s\n", dc.VerificationUri, dc.UserCode)
		fmt.Println("等待授权...")
	}

	// 步骤 2: 轮询 token (使用 form-urlencoded)
	tokenBody := url.Values{}
	tokenBody.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	tokenBody.Set("client_id", clientId)
	tokenBody.Set("device_code", dc.DeviceCode)
	
	interval := dc.Interval
	if interval <= 0 {
		interval = 5
	}
	deadline := time.Now().Add(15 * time.Minute)
	var msToken string
	for time.Now().Before(deadline) {
		time.Sleep(time.Duration(interval) * time.Second)
		b, _, _ = httpPostForm("https://login.microsoftonline.com/consumers/oauth2/v2.0/token", tokenBody.Encode())
		var tj map[string]any
		if err := json.Unmarshal(b, &tj); err != nil {
			continue
		}
		if t, ok := tj["access_token"].(string); ok && t != "" {
			msToken = t
			break
		}
		errCode, _ := tj["error"].(string)
		switch errCode {
		case "authorization_pending":
			// 继续等待
		case "slow_down":
			interval += 5
		case "":
			return nil, fmt.Errorf("微软登录返回未知响应")
		case "authorization_declined":
			return nil, fmt.Errorf("用户拒绝了授权")
		case "expired_token":
			return nil, fmt.Errorf("设备码已过期，请重新登录")
		default:
			desc, _ := tj["error_description"].(string)
			if desc == "" {
				desc = errCode
			}
			return nil, fmt.Errorf("微软登录失败: %s", desc)
		}
	}
	if msToken == "" {
		return nil, fmt.Errorf("微软登录超时")
	}
	if verbose {
		fmt.Println("✓ 微软授权成功")
	}

	// 步骤 3: Xbox Live 认证
	xblBody := `{"Properties":{"AuthMethod":"RPS","SiteName":"user.auth.xboxlive.com","RpsTicket":"d=` + msToken + `"},"RelyingParty":"http://auth.xboxlive.com","TokenType":"JWT"}`
	b, _, _ = httpPostJSON("https://user.auth.xboxlive.com/user/authenticate", xblBody)
	var xbl map[string]any
	json.Unmarshal(b, &xbl)
	xblToken, _ := xbl["Token"].(string)
	if xblToken == "" {
		return nil, fmt.Errorf("Xbox Live 认证失败")
	}
	if verbose {
		fmt.Println("✓ Xbox Live 认证成功")
	}

	// 步骤 4: XSTS 认证
	xstsBody := `{"Properties":{"SandboxId":"RETAIL","UserTokens":["` + xblToken + `"]},"RelyingParty":"rp://api.minecraftservices.com/","TokenType":"JWT"}`
	b, _, _ = httpPostJSON("https://xsts.auth.xboxlive.com/xsts/authorize", xstsBody)
	var sx map[string]any
	json.Unmarshal(b, &sx)
	xstsTok, _ := sx["Token"].(string)
	
	// DisplayClaims 实际是对象 {"xui":[{"uhs":"..."}]}，同时兼容数组形式
	uhs := extractUHS(sx)
	if xstsTok == "" || uhs == "" {
		if xe, ok := sx["XErr"].(float64); ok {
			return nil, fmt.Errorf("XSTS 认证失败: %s", xboxErrText(int(xe)))
		}
		return nil, fmt.Errorf("XSTS 认证失败")
	}
	if verbose {
		fmt.Println("✓ XSTS 认证成功")
	}

	// 步骤 5: Minecraft 认证
	mcBody := `{"identityToken":"XBL3.0 x=` + uhs + `;` + xstsTok + `"}`
	b, _, _ = httpPostJSON("https://api.minecraftservices.com/authentication/login_with_xbox", mcBody)
	var mj map[string]any
	json.Unmarshal(b, &mj)
	mcToken, _ := mj["access_token"].(string)
	if mcToken == "" {
		return nil, fmt.Errorf("Minecraft 认证失败")
	}
	if verbose {
		fmt.Println("✓ Minecraft 认证成功")
	}

	// 步骤 6: 获取玩家信息
	b, _, _ = httpGet("https://api.minecraftservices.com/minecraft/profile", "Authorization", "Bearer "+mcToken)
	var profile map[string]any
	json.Unmarshal(b, &profile)
	name2, _ := profile["name"].(string)
	id2, _ := profile["id"].(string)
	if name2 == "" {
		name2 = "MicrosoftPlayer"
	}
	if id2 == "" {
		id2 = offlineUuid("MicrosoftPlayer")
	}
	// 格式化 UUID
	if len(id2) == 32 {
		id2 = id2[:8] + "-" + id2[8:12] + "-" + id2[12:16] + "-" + id2[16:20] + "-" + id2[20:]
	}
	if verbose {
		fmt.Printf("✓ 玩家: %s\n", name2)
	}
	return &Account{Type: "microsoft", Name: name2, UUID: id2, AccessToken: mcToken}, nil
}

func loginAuthlib(server, user, pass string) (*Account, error) {
	if server == "" || user == "" {
		return nil, fmt.Errorf("服务器地址和用户名不能为空")
	}
	server = strings.TrimRight(server, "/")
	body := `{"agent":{"name":"Minecraft","version":1},"username":"` + user + `","password":"` + pass + `","requestUser":true}`
	b, code, err := httpPostJSON(server+"/authserver/authenticate", body)
	if err != nil || code != 200 {
		return nil, fmt.Errorf("认证失败: %v", err)
	}
	var mj map[string]any
	json.Unmarshal(b, &mj)
	token, _ := mj["accessToken"].(string)
	if token == "" {
		msg, _ := mj["errorMessage"].(string)
		return nil, fmt.Errorf("认证失败: %s", msg)
	}
	selected, _ := mj["selectedProfile"].(map[string]any)
	uname, _ := selected["name"].(string)
	uid, _ := selected["id"].(string)
	if uname == "" {
		uname = user
	}
	if len(uid) == 32 {
		uid = uid[:8] + "-" + uid[8:12] + "-" + uid[12:16] + "-" + uid[16:20] + "-" + uid[20:]
	}
	return &Account{Type: "authlib", Name: uname, UUID: uid, AccessToken: token, Server: server}, nil
}

// httpPostForm 发送 form-urlencoded 请求
func httpPostForm(fullURL, body string) ([]byte, int, error) {
	req, err := http.NewRequest("POST", fullURL, strings.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "MC-Launcher/1.0")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	return b, resp.StatusCode, err
}

func LoginOffline(name string) *Account {
	return loginOffline(name)
}

func LoginMicrosoft(verbose bool, clientId string) (*Account, error) {
	return loginMicrosoft(verbose, clientId)
}

func LoginAuthlib(server, user, pass string) (*Account, error) {
	return loginAuthlib(server, user, pass)
}

// extractUHS 从 XSTS 响应中取出 uhs，兼容对象与数组两种 DisplayClaims 结构
func extractUHS(resp map[string]any) string {
	dc, ok := resp["DisplayClaims"]
	if !ok {
		return ""
	}
	// 形式一: {"xui":[{"uhs":"..."}]}
	if m, ok := dc.(map[string]any); ok {
		if xui, ok := m["xui"].([]any); ok && len(xui) > 0 {
			if first, ok := xui[0].(map[string]any); ok {
				if u, ok := first["uhs"].(string); ok {
					return u
				}
			}
		}
	}
	// 形式二: [{"uhs":"..."}]
	if arr, ok := dc.([]any); ok && len(arr) > 0 {
		if first, ok := arr[0].(map[string]any); ok {
			if u, ok := first["uhs"].(string); ok {
				return u
			}
		}
	}
	return ""
}

// xboxErrText 把 XSTS 错误码转成可读文本
func xboxErrText(code int) string {
	switch code {
	case 2148916227:
		return "该账号已被 Xbox 封禁"
	case 2148916233:
		return "该账号没有 Xbox 档案，请先在 xbox.com 创建"
	case 2148916235:
		return "该账号所在地区不支持 Xbox Live"
	case 2148916236, 2148916237:
		return "需要成人验证（地区限制）"
	case 2148916238:
		return "未成年账号需要家长同意并加入家庭组"
	default:
		return fmt.Sprintf("未知错误码 %d", code)
	}
}
