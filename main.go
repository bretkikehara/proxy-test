package main

import (
	"bytes"
	"crypto/tls"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"golang.org/x/net/html"

	"github.com/ditashi/jsbeautifier-go/jsbeautifier"
	"github.com/elazarl/goproxy"
)

var beautifyOpts = map[string]interface{}{
	"indent_size":           4,
	"indent_char":           " ",
	"indent_with_tabs":      false,
	"preserve_newlines":     true,
	"max_preserve_newlines": 10,
	"space_in_paren":        false,
	"space_in_empty_paren":  false,
	"e4x":                       true,
	"jslint_happy":              false,
	"space_after_anon_function": false,
	"brace_style":               "collapse",
	"keep_array_indentation":    false,
	"keep_function_indentation": false,
	"eval_code":                 false,
	"unescape_strings":          false,
	"wrap_line_length":          0,
	"break_chained_methods":     false,
	"end_with_newline":          false,
}

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

func parseNode(n *html.Node) {
	if n.Type == html.ElementNode && n.Data == "script" {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.TextNode {
				if body, err := jsbeautifier.Beautify(&c.Data, beautifyOpts); err == nil {
					c.Data = "\n" + body + "\n"
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		parseNode(c)
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

	proxy.OnResponse().DoFunc(func(r *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		cType := strings.Split(r.Header.Get("Content-Type"), ";")[0]
		switch cType {
		case "application/javascript":
			if body, err := ioutil.ReadAll(r.Body); err == nil {
				bodyS := string(body)
				if body, err := jsbeautifier.Beautify(&bodyS, beautifyOpts); err == nil {
					return goproxy.NewResponse(r.Request, r.Header.Get("Content-Type"), r.StatusCode, body)
				}
			}
			break
		case "text/html":
			if doc, err := html.Parse(r.Body); err == nil {
				parseNode(doc)
				var buf bytes.Buffer
				w := io.Writer(&buf)
				html.Render(w, doc)
				return goproxy.NewResponse(r.Request, r.Header.Get("Content-Type"), r.StatusCode, buf.String())
			}
			break
		}
		return r
	})

	proxy.Verbose = false

	// proxyOn()

	// c := make(chan os.Signal, 2)
	// signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	// go func() {
	// 	<-c
	// 	proxyOff()
	// 	os.Exit(1)
	// }()

	log.Println("Proxy now listening on port :8888")
	log.Fatal(http.ListenAndServe(":8888", proxy))
}
