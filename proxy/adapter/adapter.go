package adapter

import "context"

type ProxyVendorAdapter interface {
	GetProxy(ctx context.Context, exitWhenError bool) (proxy string, err error)
	GetProxiesSync(ctx context.Context, count int, exitWhenError bool) (proxies []string, err error)
	GetCheckedProxiesSync(ctx context.Context, count int, exitWhenError bool) (proxies []string, err error)
	GetProxiesAsync(ctx context.Context, count int, exitWhenError bool) (chProxy chan string, chErr chan error)
	GetCheckedProxiesAsync(ctx context.Context, count int, exitWhenError bool) (chProxy chan string, chErr chan error)
}
