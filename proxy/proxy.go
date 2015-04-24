// original code is taken from 1.4.2/libexec/src/net/http/httputil/reverseproxy.go

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// HTTP reverse proxy handler

package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"
)

type BodyFallback int

const (
	BodyFallbackNone BodyFallback = iota
	BodyFallbackJsonEncode
)

type Proxy struct {
	TargetList   []*url.URL
	Transport    http.RoundTripper
	BodyFallback BodyFallback
	M            sync.RWMutex
}

func NewProxy(targetList []*url.URL) *Proxy {
	return &Proxy{
		TargetList:   targetList,
		BodyFallback: BodyFallbackNone,
		Transport:    http.DefaultTransport,
	}
}

type JsonItem struct {
	Target     string          `json:"target"`
	Body       json.RawMessage `json:"body"`
	StatusCode int             `json:"status_code"`
}

func (p *Proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	p.M.RLock()

	itemChan := make(chan *JsonItem, len(p.TargetList))
	var wg sync.WaitGroup

	targetReqMap := map[string]*http.Request{}

	for _, target := range p.TargetList {
		outreq := cloneRequest(req)
		outreq.URL = director(target, req)
		targetReqMap[target.String()] = outreq
	}

	p.M.RUnlock()

	for target, req := range targetReqMap {
		wg.Add(1)
		go func(target string, req *http.Request) {
			defer wg.Done()

			log.Debugf("target:%v request url:%v", target, req.URL)

			res, err := p.Transport.RoundTrip(req)
			if err != nil {
				log.Errorf("round trip err:%v target:%v", err, target)
				return
			}
			defer res.Body.Close()

			body, err := p.responseBodyToJsonBody(res)

			if err != nil {
				log.Errorf("%v target:%v", err, target)
			}

			item := &JsonItem{
				Target:     target,
				StatusCode: res.StatusCode,
				Body:       body,
			}

			itemChan <- item
		}(target, req)
	}

	wg.Wait()
	close(itemChan)

	items := make([]*JsonItem, 0, len(itemChan))
	for item := range itemChan {
		items = append(items, item)
	}

	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(items); err != nil {
		log.Errorf("json encode err:%v", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	rw.Write(b.Bytes())
}

func (p *Proxy) responseBodyToJsonBody(res *http.Response) (body []byte, err error) {

	var contentType string

	if res.ContentLength <= 0 {
		goto fallback
	}

	contentType = res.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "application/json"):
		body, err = ioutil.ReadAll(res.Body)
		if err != nil {
			err = fmt.Errorf("read body err:%v", err)
			return
		}
		var tmp interface{}
		if err := json.Unmarshal(body, &tmp); err != nil {
			goto fallback
		}
		return
	default:
		goto fallback
	}

fallback:
	switch p.BodyFallback {
	case BodyFallbackNone:
		body = []byte(`""`)
		return
	case BodyFallbackJsonEncode:
		if body == nil {
			body, err = ioutil.ReadAll(res.Body)
		}
		if err != nil {
			return nil, fmt.Errorf("read body err:%v", err)
		}
		var b bytes.Buffer
		if _err := json.NewEncoder(&b).Encode(body); _err != nil {
			err = fmt.Errorf("fallback json encode err:%v", _err)
			return
		}
		body = b.Bytes()
		return
	default:
		err = fmt.Errorf("not supported fallback type:%v", p.BodyFallback)
		return
	}
}

func cloneRequest(req *http.Request) *http.Request {
	outreq := new(http.Request)
	*outreq = *req // includes shallow copies of maps, but okay

	outreq.Proto = "HTTP/1.1"
	outreq.ProtoMajor = 1
	outreq.ProtoMinor = 1
	outreq.Close = false

	// Remove hop-by-hop headers to the backend.  Especially
	// important is "Connection" because we want a persistent
	// connection, regardless of what the client sent to us.  This
	// is modifying the same underlying map from req (shallow
	// copied above) so we only copy it if necessary.
	copiedHeaders := false
	for _, h := range hopHeaders {
		if outreq.Header.Get(h) != "" {
			if !copiedHeaders {
				outreq.Header = make(http.Header)
				copyHeader(outreq.Header, req.Header)
				copiedHeaders = true
			}
			outreq.Header.Del(h)
		}
	}

	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		// If we aren't the first proxy retain prior
		// X-Forwarded-For information as a comma+space
		// separated list and fold multiple headers into one.
		if prior, ok := outreq.Header["X-Forwarded-For"]; ok {
			clientIP = strings.Join(prior, ", ") + ", " + clientIP
		}
		outreq.Header.Set("X-Forwarded-For", clientIP)
	}

	return outreq
}

// create url from oridinal request and target
func director(target *url.URL, req *http.Request) *url.URL {
	targetQuery := target.RawQuery

	var url url.URL
	url = *req.URL

	url.Scheme = target.Scheme
	url.Host = target.Host
	url.Path = singleJoiningSlash(target.Path, url.Path)
	if targetQuery == "" || url.RawQuery == "" {
		url.RawQuery = targetQuery + url.RawQuery
	} else {
		url.RawQuery = targetQuery + "&" + url.RawQuery
	}

	return &url
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

// Hop-by-hop headers. These are removed when sent to the backend.
// http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
var hopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te", // canonicalized version of "TE"
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}
