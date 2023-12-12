package util

import (
	"bytes"
	"context"
	"errors"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"
	"github.com/zx106kg/go-proxy/test"
	"io"
	"log"
	"net/http"
	"reflect"
	"testing"
)

func TestCheckProxyConn(t *testing.T) {
	convey.Convey("CheckProxyConn", t, func() {

		convey.Convey("Proxy fail.", func() {
			patches := gomonkey.ApplyMethod(reflect.TypeOf(&http.Client{}), "Do", func(client *http.Client, req *http.Request) (*http.Response, error) {
				log.Println("http.Client -> Do has been mocked")
				return nil, errors.New("mock http request failed")
			})
			defer patches.Reset()
			ok, err := CheckProxyConn(context.TODO(), "http://127.0.0.1:8080")
			convey.So(ok, convey.ShouldBeFalse)
			convey.So(err, convey.ShouldBeError)
		})

		convey.Convey("Proxy success.", func() {
			patches := gomonkey.ApplyMethod(reflect.TypeOf(&http.Client{}), "Do", func(client *http.Client, req *http.Request) (*http.Response, error) {
				log.Println("http.Client -> Do has been mocked")
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(""))}, nil
			})
			defer patches.Reset()
			ok, err := CheckProxyConn(context.TODO(), "http://127.0.0.1:8080")
			convey.So(ok, convey.ShouldBeTrue)
			convey.So(err, convey.ShouldBeNil)
		})
	})
}

func TestIsContainsProxyOnly(t *testing.T) {
	convey.Convey("IsContainsProxyOnly", t, func() {
		convey.Convey("Body only contains proxy.", func() {
			body := "192.168.50.1:8888\r\n192.168.50.2:8888"
			ok := IsContainsProxyOnly(body, "\r\n")
			convey.So(ok, convey.ShouldBeTrue)
		})

		convey.Convey("Body contains wrong splitter.", func() {
			body := "192.168.50.1:8888\r192.168.50.2:8888"
			ok := IsContainsProxyOnly(body, "\n")
			convey.So(ok, convey.ShouldBeFalse)
		})

		convey.Convey("Body contains invalid proxy.", func() {
			body := "192.168.50.1:888\r192.168.50.2:8888"
			ok := IsContainsProxyOnly(body, "\n")
			convey.So(ok, convey.ShouldBeFalse)
		})
	})
}

func TestFormatRawProxy(t *testing.T) {
	convey.Convey("FormatRawProxy", t, func() {
		convey.Convey("Proxy is valid.", func() {
			proxy := "http://127.0.0.1:8888"
			proxy, err := FormatRawProxy(proxy, "", "")
			convey.So(err, convey.ShouldBeNil)
			convey.So(proxy, convey.ShouldEqual, "http://127.0.0.1:8888")
		})

		convey.Convey("Scheme prefix should be added to original string when it does not have it.", func() {
			proxy := "127.0.0.1:8888"
			proxy, err := FormatRawProxy(proxy, "", "")
			convey.So(err, convey.ShouldBeNil)
			convey.So(proxy, convey.ShouldEqual, "http://127.0.0.1:8888")
		})

		convey.Convey("Proxy is invalid", func() {
			proxy := "127.0.0:8888"
			proxy, err := FormatRawProxy(proxy, "", "")
			convey.So(err, convey.ShouldBeError)
			convey.So(proxy, convey.ShouldBeEmpty)
		})

		convey.Convey("Proxy with usr and pwd.", func() {
			proxy := "127.0.0.1:8888"
			proxy, err := FormatRawProxy(proxy, "usr", "pwd")
			convey.So(err, convey.ShouldBeNil)
			convey.So(proxy, convey.ShouldEqual, "http://usr:pwd@127.0.0.1:8888")
		})

		convey.Convey("Proxy with empty usr and pwd.", func() {
			proxy := "127.0.0.1:8888"
			proxy, err := FormatRawProxy(proxy, "", "")
			convey.So(err, convey.ShouldBeNil)
			convey.So(proxy, convey.ShouldEqual, "http://127.0.0.1:8888")
		})
	})
}

func TestCheckProxiesConnSync(t *testing.T) {
	convey.Convey("CheckProxiesConnSync", t, func() {
		convey.Convey("Check several proxies.", func() {
			proxies := []string{"http://192.168.0.1:8888", "http://192.168.0.2:8888", "http://192.168.0.3:8888"}
			r1, e1 := test.MockHttpClientReturn(true, "")
			r2, e2 := test.MockHttpClientReturn(false, "")
			outputs := []gomonkey.OutputCell{
				{Values: gomonkey.Params{r1, e1}, Times: 2},
				{Values: gomonkey.Params{r2, e2}, Times: 1},
			}
			patches := gomonkey.ApplyMethodSeq(reflect.TypeOf(&http.Client{}), "Do", outputs)
			defer patches.Reset()
			succ, fail := CheckProxiesConnSync(context.TODO(), proxies)
			convey.So(len(succ), convey.ShouldEqual, 2)
			convey.So(len(fail), convey.ShouldEqual, 1)
		})
	})
}

func TestCheckProxiesConnAsync(t *testing.T) {
	convey.Convey("CheckProxiesConnAsync", t, func() {

		convey.Convey("Check several proxies async.", func() {
			proxies := []string{"http://192.168.0.1:8888", "http://192.168.0.2:8888", "http://192.168.0.3:8888"}
			r1, e1 := test.MockHttpClientReturn(true, "")
			r2, e2 := test.MockHttpClientReturn(false, "")
			outputs := []gomonkey.OutputCell{
				{Values: gomonkey.Params{r1, e1}, Times: 2},
				{Values: gomonkey.Params{r2, e2}, Times: 1},
			}
			patches := gomonkey.ApplyMethodSeq(reflect.TypeOf(&http.Client{}), "Do", outputs)
			defer patches.Reset()
			ch := make(chan *CheckProxyConnAsyncResult)
			CheckProxiesConnAsync(context.TODO(), proxies, ch)
			var results []*CheckProxyConnAsyncResult
			for {
				result := <-ch
				results = append(results, result)
				if len(results) == 3 {
					break
				}
			}
			convey.So(len(results), convey.ShouldEqual, 3)
			var (
				succCount int
				failCount int
			)
			for _, r := range results {
				if r.Success {
					succCount++
				} else {
					failCount++
				}
			}
			convey.So(succCount, convey.ShouldEqual, 2)
			convey.So(failCount, convey.ShouldEqual, 1)
		})
	})
}
