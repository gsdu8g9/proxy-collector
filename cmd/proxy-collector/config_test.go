package main

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	type spec struct {
		Content string
		Error   string
	}

	specs := []spec{
		{
			Content: `{"target_list":[],"body_fallback": 0}`,
			Error:   "target_list is empty",
		},
		{
			Content: `{"target_list":["http://example.com"],"body_fallback": 2}`,
			Error:   "not support body fallback mode:2",
		},
		{
			Content: `{"target_list":["http://example.com"],"body_fallback": 1}`,
			Error:   "",
		},
	}

	for _, spec := range specs {
		f, err := ioutil.TempFile("", "proxy-collector")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			f.Close()
			os.Remove(f.Name())
		}()
		f.WriteString(spec.Content)
		f.Close()
		_, err = LoadConfig(f.Name())
		if err == nil {
			if spec.Error != "" {
				t.Errorf("got %v but should empty", err)
			}
		} else {
			if e, g := spec.Error, err; e != g.Error() {
				t.Errorf("got %v but should %v", e, g)
			}
		}
	}
}
