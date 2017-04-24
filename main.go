package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/elazarl/goproxy"
)

func proxyOn() {
	log.Printf("setting up network settings for: %s", runtime.GOOS)
	if strings.Contains(runtime.GOOS, "darwin") {
		cmd := exec.Command("./proxy_mac_start.sh")
		if err := cmd.Run(); err != nil {
			log.Fatal(err)
		}
	}
}

func proxyOff() {
	log.Printf("clearing network settings for: %s", runtime.GOOS)
	if strings.Contains(runtime.GOOS, "darwin") {
		cmd := exec.Command("./proxy_mac_end.sh")
		cmd.Run()
	}
}

func main() {
	goproxyCa, err := tls.LoadX509KeyPair("proxy.pem", "proxy.key")
	if err != nil {
		log.Fatal("Failed to read certificate")
	}

	goproxy.GoproxyCa = goproxyCa
	goproxy.OkConnect = &goproxy.ConnectAction{Action: goproxy.ConnectAccept, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}
	goproxy.MitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectMitm, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}
	goproxy.HTTPMitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectHTTPMitm, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}
	goproxy.RejectConnect = &goproxy.ConnectAction{Action: goproxy.ConnectReject, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}

	proxy := goproxy.NewProxyHttpServer()

	proxy.OnRequest(goproxy.UrlMatches(regexp.MustCompile("cloudfront.net"))).HandleConnect(goproxy.AlwaysMitm)

	proxy.OnResponse().DoFunc(func(r *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		r.Header.Set("X-GoProxy", "yxorPoG-X")
		return r
	})

	proxy.OnRequest(goproxy.UrlMatches(regexp.MustCompile("example.com"))).DoFunc(func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		if h, _, _ := time.Now().Clock(); h >= 8 && h <= 17 {
			return r, goproxy.NewResponse(r,
				goproxy.ContentTypeText, http.StatusForbidden,
				"Don't waste your time!")
		}
		return r, nil
	})

	proxy.Verbose = false

	proxyOn()
	defer proxyOff()
	log.Fatal(http.ListenAndServe(":8888", proxy))
}
