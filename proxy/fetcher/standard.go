package fetcher

import (
	"errors"
	"fmt"
	"go-proxy/logger"
	"go-proxy/logger/console"
	"go-proxy/util"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type StandardProxyFetcher struct {
	url      string
	username string
	password string
	splitter string
	logger   logger.Logger
	client   *http.Client
}

type StandardProxyFetcherConfig struct {
	Url      string
	Username string
	Password string
	Splitter string
	Logger   logger.Logger
}

// NewStandardProxyFetcher 创建StandardProxyFetcher
func NewStandardProxyFetcher(config *StandardProxyFetcherConfig) *StandardProxyFetcher {
	splitter := config.Splitter
	if splitter == "" {
		splitter = "\r\n"
	}
	log := config.Logger
	if log == nil {
		log = console.NewLogger()
	}
	return &StandardProxyFetcher{
		url:      config.Url,
		username: config.Username,
		password: config.Password,
		splitter: splitter,
		logger:   log,
		client:   &http.Client{Timeout: 5 * time.Second},
	}
}

// GetProxy 获取一个代理
func (f *StandardProxyFetcher) GetProxy(exitWhenError bool) (proxy string, err error) {
	proxies, err := f.GetProxiesSync(1, exitWhenError)
	if err != nil {
		return "", err
	}
	return proxies[0], nil
}

// GetProxiesSync 同步批量获取代理
//
// count 获取数量
//
// exitWhenError 当调用api失败时是否立刻结束
func (f *StandardProxyFetcher) GetProxiesSync(count int, exitWhenError bool) (proxies []string, err error) {
	for {
		if len(proxies) >= count {
			break
		}
		// 获取匹配获取数量的url
		apiUrl := f.replaceNumPlaceholder(count - len(proxies))
		body, err := f.callApi(apiUrl)
		if err != nil {
			if f.logger != nil {
				f.logger.Warn(fmt.Sprintf("[GetProxiesSync] 调用代理供应商API失败. %v", err))
			}
			if exitWhenError {
				return nil, err
			}
			time.Sleep(1 * time.Second)
			continue
		}
		if !util.IsContainsProxyOnly(body, f.splitter) {
			f.logger.Warn(fmt.Sprintf("[GetProxiesSync] 供应商API返回非法文本. 原文: %s", body))
			if exitWhenError {
				return nil, fmt.Errorf("供应商API返回非法文本. 原文: %s", body)
			}
			continue
		}
		tProxies := util.GetProxyFromBody(body, f.splitter)
		proxies = append(proxies, tProxies...)
	}
	return proxies, nil
}

// GetCheckedProxiesSync 同步批量获取已检查的代理
func (f *StandardProxyFetcher) GetCheckedProxiesSync(count int, exitWhenError bool) (proxies []string, err error) {
	for {
		tProxies, err := f.GetProxiesSync(count, exitWhenError)
		if err != nil {
			return nil, err
		}
		succ, _ := util.CheckProxiesConnSync(tProxies)
		proxies = append(proxies, succ...)
		if len(proxies) >= count {
			return proxies, nil
		}
	}
}

// GetProxiesAsync 异步批量获取代理
//
// chProxy 成功的代理通过此channel返回
//
// chErr 异常通过此channel返回
func (f *StandardProxyFetcher) GetProxiesAsync(count int, exitWhenError bool) (chProxy chan string, chErr chan error) {
	chProxy = make(chan string)
	chErr = make(chan error)

	go func() {
		var current atomic.Int64
		for {
			if int(current.Load()) >= count {
				close(chProxy)
				return
			}
			// 获取匹配获取数量的url
			apiUrl := f.replaceNumPlaceholder(count - int(current.Load()))
			body, err := f.callApi(apiUrl)
			if err != nil {
				if f.logger != nil {
					f.logger.Warn(fmt.Sprintf("[GetProxiesSync] 调用代理供应商API失败. %v", err))
				}
				if exitWhenError {
					chErr <- err
					return
				}
				time.Sleep(1 * time.Second)
				continue
			}
			if !util.IsContainsProxyOnly(body, f.splitter) {
				f.logger.Warn(fmt.Sprintf("[GetProxiesSync] 供应商API返回非法文本. 原文: %s", body))
				if exitWhenError {
					chErr <- fmt.Errorf("供应商API返回非法文本. 原文: %s", body)
					return
				}
				continue
			}
			proxies := util.GetProxyFromBody(body, f.splitter)
			for _, proxy := range proxies {
				chProxy <- proxy
				current.Add(1)
			}
		}
	}()

	return chProxy, chErr
}

// GetCheckedProxiesAsync 异步批量获取已检查的代理
//
// chProxy 成功的代理通过此channel返回
//
// chErr 异常通过此channel返回
func (f *StandardProxyFetcher) GetCheckedProxiesAsync(count int, exitWhenError bool) (chProxy chan string, chErr chan error) {
	chProxy = make(chan string)
	chErr = make(chan error)

	go func() {
		var current atomic.Int64
		for {
			if int(current.Load()) >= count {
				close(chProxy)
				return
			}
			// 获取匹配获取数量的url
			apiUrl := f.replaceNumPlaceholder(count - int(current.Load()))
			body, err := f.callApi(apiUrl)
			if err != nil {
				if f.logger != nil {
					f.logger.Warn(fmt.Sprintf("[GetProxiesSync] 调用代理供应商API失败. %v", err))
				}
				if exitWhenError {
					chErr <- err
					return
				}
				time.Sleep(1 * time.Second)
				continue
			}
			if !util.IsContainsProxyOnly(body, f.splitter) {
				f.logger.Warn(fmt.Sprintf("[GetProxiesSync] 供应商API返回非法文本. 原文: %s", body))
				if exitWhenError {
					chErr <- fmt.Errorf("供应商API返回非法文本. 原文: %s", body)
					return
				}
				continue
			}
			proxies := util.GetProxyFromBody(body, f.splitter)
			chResult := make(chan *util.CheckProxyConnAsyncResult)
			util.CheckProxiesConnAsync(proxies, chResult)
			var tcount int
			for tcount < len(proxies) {
				r := <-chResult
				tcount++
				if r.Success {
					chProxy <- r.Proxy
					current.Add(1)
				}
			}
		}
	}()

	return chProxy, chErr
}

// replaceNumPlaceholder 使用count替换配置url中的占位符${num}, 生成实际的代理获取url
func (f *StandardProxyFetcher) replaceNumPlaceholder(count int) string {
	if !strings.Contains(f.url, `${num}`) {
		return f.url
	}
	return strings.ReplaceAll(f.url, `${num}`, strconv.Itoa(count))
}

// callApi
func (f *StandardProxyFetcher) callApi(apiUrl string) (body string, err error) {
	req, _ := http.NewRequest("GET", apiUrl, nil)
	resp, err := f.client.Do(req)
	if err != nil {
		return "", err
	}
	buf, err := io.ReadAll(resp.Body)
	body = string(buf)
	if err != nil {
		return "", errors.New("调用代理API失败")
	}
	if resp.StatusCode != 200 {
		return body, fmt.Errorf("调用代理API返回状态码异常, StatusCode=%d", resp.StatusCode)
	}
	return body, nil
}
