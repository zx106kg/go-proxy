package util

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// CheckProxyConn 检查代理连通性
//
// proxy必须完整带有scheme
func CheckProxyConn(ctx context.Context, proxy string) (ok bool, err error) {
	urlProxy, err := url.Parse(proxy)
	if err != nil {
		return false, fmt.Errorf("代理字符串格式错误. %+v", err)
	}
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(urlProxy),
		},
		Timeout: 3 * time.Second,
	}
	req, err := http.NewRequest("GET", "http://www.baidu.com", nil)
	if ctx != nil {
		req = req.WithContext(ctx)
	}
	if err != nil {
		return false, err
	}
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36")
	req.Header.Add("Accept-Encoding", "gzip, deflate, br")
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	_ = resp.Body.Close()
	return resp.StatusCode == 200, nil
}

// IsContainsProxyOnly 检查文本中是否只包含代理连接串.
//
// splitter 分隔符.
func IsContainsProxyOnly(body string, splitter string) bool {
	if body == "" {
		return false
	}
	vec := strings.Split(body, splitter)
	reg, _ := regexp.Compile(`^\d+\.\d+\.\d+\.\d+:\d+$`)
	for _, v := range vec {
		if v == "" {
			continue
		}
		match := reg.MatchString(v)
		if !match {
			return false
		}
	}
	return true
}

// GetProxyFromBody 从body中获取代理连接串
//
// splitter 分隔符.
func GetProxyFromBody(body string, splitter string) []string {
	var proxies []string
	for _, v := range strings.Split(body, splitter) {
		vTrim := strings.TrimSpace(v)
		if vTrim == "" {
			continue
		}
		proxies = append(proxies, vTrim)
	}
	return proxies
}

// FormatRawProxy 格式化原始代理连接串
func FormatRawProxy(proxy string, usr string, pwd string) (formatted string, err error) {
	if !strings.Contains(proxy, "http://") {
		proxy = "http://" + proxy
	}
	parsed, err := url.Parse(proxy)
	if err != nil {
		return "", errors.New("format is incorrect")
	}
	ip := net.ParseIP(parsed.Hostname())
	if ip == nil || ip.To4() == nil {
		return "", errors.New("format is incorrect")
	}
	if parsed.Port() == "" {
		return "", errors.New("format is incorrect")
	}
	if usr != "" && pwd != "" {
		parsed.User = url.UserPassword(usr, pwd)
	}
	return parsed.String(), nil
}

// CheckProxiesConnSync 批量检查代理连通性, 同步返回结果
//
// succ为测试成功的代理组, fail为测试失败的代理组.
func CheckProxiesConnSync(ctx context.Context, proxies []string) (succ, fail []string) {
	c := sync.NewCond(&sync.Mutex{})
	for _, proxy := range proxies {
		go func(proxy string) {
			ok, _ := CheckProxyConn(ctx, proxy)
			c.L.Lock()
			if ok {
				succ = append(succ, proxy)
			} else {
				fail = append(fail, proxy)
			}
			c.L.Unlock()
			c.Broadcast()
		}(proxy)
	}
	c.L.Lock()
	for len(succ)+len(fail) != len(proxies) {
		c.Wait()
	}
	c.L.Unlock()
	return succ, fail
}

// CheckProxiesConnAsync 异步批量检查代理连通性
//
// 检查完成时, 立刻通过ch返回结果
func CheckProxiesConnAsync(ctx context.Context, proxies []string, ch chan *CheckProxyConnAsyncResult) {
	c := sync.NewCond(&sync.Mutex{})
	total := len(proxies)
	for _, proxy := range proxies {
		go func(proxy string) {
			ok, _ := CheckProxyConn(ctx, proxy)
			c.L.Lock()
			total--
			c.L.Unlock()
			ch <- &CheckProxyConnAsyncResult{Proxy: proxy, Success: ok}
			c.Broadcast()
		}(proxy)
	}
}
