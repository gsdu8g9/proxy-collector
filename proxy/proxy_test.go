package proxy

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"testing"
)

type bodyFallbackSpec struct {
	BodyFallback BodyFallback
	Expected     string
}

func TestProxyValidJson(t *testing.T) {
	targetList := []*url.URL{}

	content := []byte(`{"ping":"pong"}`)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer backend.Close()

	url, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	targetList = append(targetList, url)

	specs := []bodyFallbackSpec{
		{
			BodyFallback: BodyFallbackNone,
			Expected:     fmt.Sprintf("[{\"target\":\"%v\",\"body\":%v,\"status_code\":%v}]\n", backend.URL, string(content), 200),
		},
		{
			BodyFallback: BodyFallbackJsonEncode,
			Expected:     fmt.Sprintf("[{\"target\":\"%v\",\"body\":%v,\"status_code\":%v}]\n", backend.URL, string(content), 200),
		},
	}

	for _, spec := range specs {
		proxy := NewProxy(targetList)
		proxy.BodyFallback = spec.BodyFallback

		frontend := httptest.NewServer(proxy)
		defer frontend.Close()

		req, _ := http.NewRequest("GET", frontend.URL, nil)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		if spec.Expected != string(body) {
			t.Errorf("should %v but got %v", spec.Expected, string(body))
		}
	}
}

func TestProxyInvalidJson(t *testing.T) {
	targetList := []*url.URL{}

	invalidJson := []byte(`{"ping":"pong"`)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(invalidJson)
	}))
	defer backend.Close()

	url, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	targetList = append(targetList, url)

	specs := []bodyFallbackSpec{
		{
			BodyFallback: BodyFallbackNone,
			Expected:     fmt.Sprintf("[{\"target\":\"%v\",\"body\":\"%v\",\"status_code\":%v}]\n", backend.URL, "", 200),
		},
		{
			BodyFallback: BodyFallbackJsonEncode,
			Expected:     fmt.Sprintf("[{\"target\":\"%v\",\"body\":\"%v\",\"status_code\":%v}]\n", backend.URL, base64.StdEncoding.EncodeToString(invalidJson), 200),
		},
	}

	for _, spec := range specs {
		proxy := NewProxy(targetList)
		proxy.BodyFallback = spec.BodyFallback

		frontend := httptest.NewServer(proxy)
		defer frontend.Close()

		req, _ := http.NewRequest("GET", frontend.URL, nil)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		if spec.Expected != string(body) {
			t.Errorf("should %v but got %v", spec.Expected, string(body))
		}
	}
}

func TestProxyEmptyBody(t *testing.T) {
	targetList := []*url.URL{}

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	url, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	targetList = append(targetList, url)

	specs := []bodyFallbackSpec{
		{
			BodyFallback: BodyFallbackNone,
			Expected:     fmt.Sprintf("[{\"target\":\"%v\",\"body\":\"%v\",\"status_code\":%v}]\n", backend.URL, "", 200),
		},
		{
			BodyFallback: BodyFallbackJsonEncode,
			Expected:     fmt.Sprintf("[{\"target\":\"%v\",\"body\":\"%v\",\"status_code\":%v}]\n", backend.URL, "", 200),
		},
	}

	for _, spec := range specs {
		proxy := NewProxy(targetList)
		proxy.BodyFallback = spec.BodyFallback

		frontend := httptest.NewServer(proxy)
		defer frontend.Close()

		req, _ := http.NewRequest("GET", frontend.URL, nil)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		if spec.Expected != string(body) {
			t.Errorf("should %v but got %v", spec.Expected, string(body))
		}
	}
}

func TestProxyNotJsonContent(t *testing.T) {
	targetList := []*url.URL{}

	content := []byte("not found")

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write(content)
	}))
	defer backend.Close()

	url, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	targetList = append(targetList, url)

	specs := []bodyFallbackSpec{
		{
			BodyFallback: BodyFallbackNone,
			Expected:     fmt.Sprintf("[{\"target\":\"%v\",\"body\":\"%v\",\"status_code\":%v}]\n", backend.URL, "", 404),
		},
		{
			BodyFallback: BodyFallbackJsonEncode,
			Expected:     fmt.Sprintf("[{\"target\":\"%v\",\"body\":\"%v\",\"status_code\":%v}]\n", backend.URL, base64.StdEncoding.EncodeToString(content), 404),
		},
	}

	for _, spec := range specs {
		proxy := NewProxy(targetList)
		proxy.BodyFallback = spec.BodyFallback

		frontend := httptest.NewServer(proxy)
		defer frontend.Close()

		req, _ := http.NewRequest("GET", frontend.URL, nil)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		if spec.Expected != string(body) {
			t.Errorf("should %v but got %v", spec.Expected, string(body))
		}
	}
}

func TestProxyDownHost(t *testing.T) {
	targetList := []*url.URL{}

	content := []byte("not found")

	backend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write(content)
	}))
	defer backend1.Close()

	{
		url, err := url.Parse(backend1.URL)
		if err != nil {
			t.Fatal(err)
		}
		targetList = append(targetList, url)
	}

	backend2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write(content)
	}))
	backend2.Close()

	{
		url, err := url.Parse(backend2.URL)
		if err != nil {
			t.Fatal(err)
		}
		targetList = append(targetList, url)
	}

	specs := []bodyFallbackSpec{
		{
			BodyFallback: BodyFallbackNone,
			Expected:     fmt.Sprintf("[{\"target\":\"%v\",\"body\":\"%v\",\"status_code\":%v}]\n", backend1.URL, "", 404),
		},
	}

	for _, spec := range specs {
		proxy := NewProxy(targetList)
		proxy.BodyFallback = spec.BodyFallback

		frontend := httptest.NewServer(proxy)
		defer frontend.Close()

		req, _ := http.NewRequest("GET", frontend.URL, nil)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		if spec.Expected != string(body) {
			t.Errorf("should %v but got %v", spec.Expected, string(body))
		}
	}
}

type JsonItems []JsonItem

func (a JsonItems) Len() int           { return len(a) }
func (a JsonItems) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a JsonItems) Less(i, j int) bool { return a[i].StatusCode < a[j].StatusCode }

func TestProxyMultipleHost(t *testing.T) {
	targetList := []*url.URL{}

	content := []byte("not found")

	backend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer backend1.Close()

	{
		url, err := url.Parse(backend1.URL)
		if err != nil {
			t.Fatal(err)
		}
		targetList = append(targetList, url)
	}

	backend2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write(content)
	}))
	defer backend2.Close()

	{
		url, err := url.Parse(backend2.URL)
		if err != nil {
			t.Fatal(err)
		}
		targetList = append(targetList, url)
	}

	proxy := NewProxy(targetList)
	proxy.BodyFallback = BodyFallbackNone

	frontend := httptest.NewServer(proxy)
	defer frontend.Close()

	req, _ := http.NewRequest("GET", frontend.URL, nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	var jsonItems []JsonItem
	if err := json.NewDecoder(res.Body).Decode(&jsonItems); err != nil {
		t.Fatal(err)
	}
	sort.Sort(JsonItems(jsonItems))

	expecteds := []JsonItem{
		{
			Target:     backend1.URL,
			StatusCode: 200,
			Body:       []byte(`""`),
		},
		{
			Target:     backend2.URL,
			StatusCode: 404,
			Body:       []byte(`""`),
		},
	}

	if len(jsonItems) != len(expecteds) {
		t.Errorf("should %v but got %v", len(expecteds), len(jsonItems))
	}

	for i, expected := range expecteds {
		if jsonItems[i].Target != expected.Target {
			t.Errorf("should %v but got %v", expected.Target, jsonItems[i].Target)
		}
		if jsonItems[i].StatusCode != expected.StatusCode {
			t.Errorf("should %v but got %v", expected.StatusCode, jsonItems[i].StatusCode)
		}
		if !bytes.Equal(jsonItems[i].Body, expected.Body) {
			t.Errorf("should %v but got %v", expected.Body, jsonItems[i].Body)
		}
	}
}
