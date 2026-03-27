// Package requests 提供简化的 HTTP 请求工具，供平台模块使用
package requests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var defaultClient = &http.Client{Timeout: 15 * time.Second}

// Response HTTP 响应封装
type Response struct {
	body   []byte
	status int
}

// Get 发送 GET 请求
func Get(url string) (*Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := defaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	return &Response{body: body, status: resp.StatusCode}, nil
}

// Text 返回响应体文本
func (r *Response) Text() string {
	return string(r.body)
}

// JSON 解析 JSON 响应，返回 JSON 访问器
func (r *Response) JSON() (JSON, bool) {
	var raw map[string]interface{}
	if err := json.Unmarshal(r.body, &raw); err != nil {
		return JSON{}, false
	}
	return JSON{data: raw}, true
}

// Status 返回 HTTP 状态码
func (r *Response) Status() int {
	return r.status
}

// JSON 简单的 JSON 访问器（链式调用）
type JSON struct {
	data interface{}
}

// Get 获取指定 key 的值
func (j JSON) Get(key string) JSON {
	if m, ok := j.data.(map[string]interface{}); ok {
		return JSON{data: m[key]}
	}
	return JSON{}
}

// ToInt 转为 int64
func (j JSON) ToInt() int64 {
	switch v := j.data.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	}
	return 0
}

// ToString 转为字符串
func (j JSON) ToString() string {
	if v, ok := j.data.(string); ok {
		return v
	}
	return fmt.Sprintf("%v", j.data)
}

// ToArray 转为 JSON 数组
func (j JSON) ToArray() []JSON {
	arr, ok := j.data.([]interface{})
	if !ok {
		return nil
	}
	result := make([]JSON, len(arr))
	for i, v := range arr {
		result[i] = JSON{data: v}
	}
	return result
}
