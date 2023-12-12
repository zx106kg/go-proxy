package tunnel

import "context"

type Tunnel struct {
	url string
}

type CreateConfig struct {
	Url string
}

func NewTunnel(config *CreateConfig) *Tunnel {
	return &Tunnel{
		url: config.Url,
	}
}

func (t *Tunnel) GetProxy(ctx context.Context, _ bool) (proxy string, err error) {
	return t.url, nil
}

func (t *Tunnel) GetProxiesSync(ctx context.Context, count int, _ bool) (proxies []string, err error) {
	for i := 0; i < count; i++ {
		proxies = append(proxies, t.url)
	}
	return proxies, nil
}

func (t *Tunnel) GetCheckedProxiesSync(ctx context.Context, count int, exitWhenError bool) (proxies []string, err error) {
	return t.GetProxiesSync(ctx, count, exitWhenError)
}

func (t *Tunnel) GetProxiesAsync(ctx context.Context, count int, _ bool) (chProxy chan string, chErr chan error) {
	chProxy = make(chan string)
	chErr = make(chan error)
	go func() {
		for i := 0; i < count; i++ {
			chProxy <- t.url
		}
		close(chProxy)
	}()
	return chProxy, chErr
}

func (t *Tunnel) GetCheckedProxiesAsync(ctx context.Context, count int, exitWhenError bool) (chProxy chan string, chErr chan error) {
	return t.GetProxiesAsync(ctx, count, exitWhenError)
}
