// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Reverse proxy tests.

package proxy

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestReverseProxy(t *testing.T) {
	const backendResponse = "I am the backend"
	const backendStatus = 404
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.TransferEncoding) > 0 {
			t.Errorf("backend got unexpected TransferEncoding: %v", r.TransferEncoding)
		}
		if r.Header.Get("X-Forwarded-For") == "" {
			t.Errorf("didn't get X-Forwarded-For header")
		}
		if c := r.Header.Get("Connection"); c != "" {
			t.Errorf("handler got Connection header value %q", c)
		}
		if c := r.Header.Get("Upgrade"); c != "" {
			t.Errorf("handler got Upgrade header value %q", c)
		}
		if g, e := r.Host, "some-name"; g != e {
			t.Errorf("backend got Host header %q, want %q", g, e)
		}
		w.WriteHeader(backendStatus)
		w.Write([]byte(backendResponse))
	}))
	defer backend.Close()
	backendURL, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	proxy := NewProxy([]*url.URL{backendURL})
	proxy.BodyFallback = BodyFallbackNone
	frontend := httptest.NewServer(proxy)
	defer frontend.Close()

	getReq, _ := http.NewRequest("GET", frontend.URL, nil)
	getReq.Host = "some-name"
	getReq.Header.Set("Connection", "close")
	getReq.Header.Set("Upgrade", "foo")
	getReq.Close = true
	res, err := http.DefaultClient.Do(getReq)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if g, e := res.StatusCode, http.StatusOK; g != e {
		t.Errorf("got res.StatusCode %d; expected %d", g, e)
	}
}

func TestXForwardedFor(t *testing.T) {
	const prevForwardedFor = "client ip"
	const backendResponse = "I am the backend"
	const backendStatus = 404
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Forwarded-For") == "" {
			t.Errorf("didn't get X-Forwarded-For header")
		}
		if !strings.Contains(r.Header.Get("X-Forwarded-For"), prevForwardedFor) {
			t.Errorf("X-Forwarded-For didn't contain prior data")
		}
		w.WriteHeader(backendStatus)
		w.Write([]byte(backendResponse))
	}))
	defer backend.Close()
	backendURL, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	proxy := NewProxy([]*url.URL{backendURL})
	proxy.BodyFallback = BodyFallbackNone
	frontend := httptest.NewServer(proxy)
	defer frontend.Close()

	getReq, _ := http.NewRequest("GET", frontend.URL, nil)
	getReq.Host = "some-name"
	getReq.Header.Set("Connection", "close")
	getReq.Header.Set("X-Forwarded-For", prevForwardedFor)
	getReq.Close = true
	res, err := http.DefaultClient.Do(getReq)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if g, e := res.StatusCode, http.StatusOK; g != e {
		t.Errorf("got res.StatusCode %d; expected %d", g, e)
	}
}

var proxyQueryTests = []struct {
	baseSuffix string // suffix to add to backend URL
	reqSuffix  string // suffix to add to frontend's request URL
	want       string // what backend should see for final request URL (without ?)
}{
	{"", "", ""},
	{"?sta=tic", "?us=er", "sta=tic&us=er"},
	{"", "?us=er", "us=er"},
	{"?sta=tic", "", "sta=tic"},
}

func TestReverseProxyQuery(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if g, e := r.URL.RawQuery, r.Header.Get("X-Want-Query"); g != e {
			t.Errorf("got r.URL.RawQuery %v. but expected %v", g, e)
		}
		w.Write([]byte("hi"))
	}))
	defer backend.Close()

	for i, tt := range proxyQueryTests {
		backendURL, err := url.Parse(backend.URL + tt.baseSuffix)
		if err != nil {
			t.Fatal(err)
		}
		proxy := NewProxy([]*url.URL{backendURL})
		proxy.BodyFallback = BodyFallbackNone
		frontend := httptest.NewServer(proxy)
		req, _ := http.NewRequest("GET", frontend.URL+tt.reqSuffix, nil)
		req.Header.Set("X-Want-Query", tt.want)
		req.Close = true
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%d. Get: %v", i, err)
		}
		res.Body.Close()
		frontend.Close()
	}
}
