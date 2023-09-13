package adapter

type ProxyVendorAdapter interface {
	GetProxy(exitWhenError bool) (proxy string, err error)
	GetProxiesSync(count int, exitWhenError bool) (proxies []string, err error)
	GetCheckedProxiesSync(count int, exitWhenError bool) (proxies []string, err error)
	GetProxiesAsync(count int, exitWhenError bool) (chProxy chan string, chErr chan error)
	GetCheckedProxiesAsync(count int, exitWhenError bool) (chProxy chan string, chErr chan error)
}
