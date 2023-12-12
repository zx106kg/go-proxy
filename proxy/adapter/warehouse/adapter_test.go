package warehouse

import (
	"context"
	"errors"
	"fmt"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"
	"github.com/zx106kg/go-proxy/test"
	"github.com/zx106kg/go-proxy/util"
	"net/http"
	"reflect"
	"testing"
)

func TestWarehouse_replaceNumPlaceholder(t *testing.T) {
	convey.Convey("replaceNumPlaceholder", t, func() {

		convey.Convey("ApiUrl contains ${num}", func() {
			replaced := NewWarehouse(&CreateConfig{
				Url: "http://localhost?qty=${num}&type=",
			}).replaceNumPlaceholder(5)
			convey.So(replaced, convey.ShouldEqual, "http://localhost?qty=5&type=")
		})

		convey.Convey("ApiUrl does not contain ${num}", func() {
			replaced := NewWarehouse(&CreateConfig{
				Url: "http://localhost?qty=1&type=",
			}).replaceNumPlaceholder(5)
			convey.So(replaced, convey.ShouldEqual, "http://localhost?qty=1&type=")
		})

	})
}

func TestWarehouse_GetProxiesSync(t *testing.T) {
	convey.Convey("GetProxiesSync", t, func() {
		fetcher := NewWarehouse(&CreateConfig{
			Url: "http://proxy-agent.com?qty=${num}",
		})

		convey.Convey("Exit when error occurs", func() {
			patch := gomonkey.ApplyMethod(reflect.TypeOf(&http.Client{}), "Do", func(client *http.Client, req *http.Request) (*http.Response, error) {
				return nil, errors.New("mock http request failed")
			})
			defer patch.Reset()

			proxies, err := fetcher.GetProxiesSync(context.TODO(), 2, true)
			convey.So(proxies, convey.ShouldBeNil)
			convey.So(err, convey.ShouldBeError)
		})

		convey.Convey("Fetch several times to get enough proxies", func() {
			r1, e1 := test.MockHttpClientReturn(true, "192.168.0.1:8888\r\n")
			r2, e2 := test.MockHttpClientReturn(false, "")
			r3, e3 := test.MockHttpClientReturn(true, "192.168.0.2:8888\r\n192.168.0.3:8888")
			outputs := []gomonkey.OutputCell{
				{Values: gomonkey.Params{r1, e1}, Times: 1},
				{Values: gomonkey.Params{r2, e2}, Times: 1},
				{Values: gomonkey.Params{r3, e3}, Times: 1},
			}
			patches := gomonkey.ApplyMethodSeq(reflect.TypeOf(&http.Client{}), "Do", outputs)
			defer patches.Reset()
			proxies, err := fetcher.GetProxiesSync(context.TODO(), 2, false)
			convey.So(len(proxies), convey.ShouldEqual, 3)
			convey.So(err, convey.ShouldBeNil)

		})
	})
}

func TestWarehouse_GetCheckedProxiesSync(t *testing.T) {
	convey.Convey("GetCheckedProxiesSync", t, func() {
		fetcher := NewWarehouse(&CreateConfig{
			Url: "http://proxy-agent.com?qty=${num}",
		})

		convey.Convey("Exit when error occurs", func() {
			patch := gomonkey.ApplyMethodReturn(fetcher, "GetProxiesSync", nil, errors.New(""))
			defer patch.Reset()
			proxies, err := fetcher.GetCheckedProxiesSync(context.TODO(), 2, true)
			convey.So(proxies, convey.ShouldBeNil)
			convey.So(err, convey.ShouldBeError)
		})

		convey.Convey("Fetch several times to get enough proxies", func() {
			outputs1 := []gomonkey.OutputCell{
				{Values: gomonkey.Params{[]string{"192.168.0.1:8888", "192.168.0.3:8888"}, nil}, Times: 1},
				{Values: gomonkey.Params{[]string{"192.168.0.3:8888"}, nil}, Times: 1},
			}
			outputs2 := []gomonkey.OutputCell{
				{Values: gomonkey.Params{true, nil}, Times: 1},
				{Values: gomonkey.Params{false, nil}, Times: 1},
				{Values: gomonkey.Params{true, nil}, Times: 1},
			}
			patches1 := gomonkey.ApplyMethodSeq(fetcher, "GetProxiesSync", outputs1)
			patches2 := gomonkey.ApplyFuncSeq(util.CheckProxyConn, outputs2)
			defer patches1.Reset()
			defer patches2.Reset()
			proxies, err := fetcher.GetCheckedProxiesSync(context.TODO(), 2, false)
			convey.So(len(proxies), convey.ShouldEqual, 2)
			convey.So(err, convey.ShouldBeNil)

		})
	})
}

func TestWarehouse_GetProxy(t *testing.T) {
	convey.Convey("GetProxy", t, func() {
		fetcher := NewWarehouse(&CreateConfig{
			Url: "http://proxy-agent.com?qty=${num}",
		})

		convey.Convey("Exit when error occurs", func() {
			patch := gomonkey.ApplyMethodReturn(fetcher, "GetProxiesSync", nil, errors.New(""))
			defer patch.Reset()
			proxy, err := fetcher.GetProxy(context.TODO(), true)
			convey.So(proxy, convey.ShouldBeEmpty)
			convey.So(err, convey.ShouldBeError)
		})

		convey.Convey("Successfully get a proxy", func() {
			patch := gomonkey.ApplyMethodReturn(fetcher, "GetProxiesSync", []string{"http://192.168.0.1"}, nil)
			defer patch.Reset()
			proxy, err := fetcher.GetProxy(context.TODO(), true)
			convey.So(proxy, convey.ShouldEqual, "http://192.168.0.1")
			convey.So(err, convey.ShouldBeNil)
		})
	})
}

func TestWarehouse_GetCheckedProxiesAsync(t *testing.T) {
	convey.Convey("GetCheckedProxiesAsync", t, func() {
		fetcher := NewWarehouse(&CreateConfig{
			Url: "http://proxy-agent.com?qty=${num}",
		})

		convey.Convey("Exit when error occurs", func() {
			patch := gomonkey.ApplyPrivateMethod(fetcher, "callApi", func(ctx context.Context, apiUrl string) (body string, err error) {
				return "", errors.New("")
			})
			defer patch.Reset()
			_, chErr := fetcher.GetCheckedProxiesAsync(context.TODO(), 10, true)
			err := <-chErr
			convey.So(err, convey.ShouldBeError)
		})

		convey.Convey("Get 2 proxies, one of them is invalid, then retry.", func() {
			var t int
			pCallApi := gomonkey.ApplyPrivateMethod(fetcher, "callApi", func(ctx context.Context, apiUrl string) (body string, err error) {
				t++
				return fmt.Sprintf("192.168.50.%d:8888", t), nil
			})
			defer pCallApi.Reset()

			pConn := gomonkey.ApplyFunc(util.CheckProxyConn, func(ctx context.Context, proxy string) (ok bool, err error) {
				switch proxy {
				case "http://192.168.50.1:8888":
					ok = true
				case "http://192.168.50.2:8888":
					ok = false
				case "http://192.168.50.3:8888":
					ok = true
				}
				return ok, nil
			})
			defer pConn.Reset()

			var proxies []string
			chProxy, chErr := fetcher.GetCheckedProxiesAsync(context.TODO(), 2, false)
			var done bool
			for {
				select {
				case proxy, open := <-chProxy:
					if !open {
						done = true
						break
					}
					proxies = append(proxies, proxy)
				case err := <-chErr:
					convey.So(err, convey.ShouldBeNil)
					return
				}
				if done {
					break
				}
			}
			convey.So(len(proxies), convey.ShouldBeGreaterThanOrEqualTo, 2)
			convey.So(proxies, convey.ShouldContain, "http://192.168.50.1:8888")
			convey.So(proxies, convey.ShouldContain, "http://192.168.50.3:8888")
		})
	})
}

func TestWarehouse_GetProxiesAsync(t *testing.T) {
	convey.Convey("GetProxiesAsync", t, func() {
		fetcher := NewWarehouse(&CreateConfig{
			Url: "http://proxy-agent.com?qty=${num}",
		})

		convey.Convey("Exit when error occurs", func() {
			patch := gomonkey.ApplyPrivateMethod(fetcher, "callApi", func(ctx context.Context, apiUrl string) (body string, err error) {
				return "", errors.New("")
			})
			defer patch.Reset()
			_, chErr := fetcher.GetCheckedProxiesAsync(context.TODO(), 10, true)
			err := <-chErr
			convey.So(err, convey.ShouldBeError)
		})

		convey.Convey("Get 3 proxies in 2 times.", func() {
			var t int
			pCallApi := gomonkey.ApplyPrivateMethod(fetcher, "callApi", func(ctx context.Context, apiUrl string) (body string, err error) {
				if t == 0 {
					body = "192.168.50.1:8888\r\n192.168.50.2:8888"
				} else if t == 1 {
					body = "192.168.50.3:8888"
				}
				t++
				return body, nil
			})
			defer pCallApi.Reset()

			pConn := gomonkey.ApplyFunc(util.CheckProxyConn, func(ctx context.Context, proxy string) (ok bool, err error) {
				switch proxy {
				case "192.168.50.1:8888":
					ok = true
				case "192.168.50.2:8888":
					ok = false
				case "192.168.50.3:8888":
					ok = true
				}
				return ok, nil
			})
			defer pConn.Reset()

			var proxies []string
			chProxy, chErr := fetcher.GetProxiesAsync(context.TODO(), 3, false)
			var done bool
			for {
				select {
				case proxy, open := <-chProxy:
					if !open {
						done = true
						break
					}
					proxies = append(proxies, proxy)
				case err := <-chErr:
					convey.So(err, convey.ShouldBeNil)
					return
				}
				if done {
					break
				}
			}
			convey.So(len(proxies), convey.ShouldBeGreaterThanOrEqualTo, 3)
		})
	})
}
