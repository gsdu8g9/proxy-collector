package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/soh335/proxy-collector/proxy"
)

type Config struct {
	TargetList   []string           `json:"target_list"`
	BodyFallback proxy.BodyFallback `json:"body_fallback"`
}

func (c *Config) validate() error {
	if len(c.TargetList) <= 0 {
		return fmt.Errorf("target_list is empty")
	}
	switch c.BodyFallback {
	case proxy.BodyFallbackNone, proxy.BodyFallbackJsonEncode:
		break
	default:
		return fmt.Errorf("not support body fallback mode:%v", c.BodyFallback)
	}

	return nil
}

func (c *Config) TargetListAsURL() ([]*url.URL, error) {
	targetList := make([]*url.URL, 0, len(c.TargetList))
	for _, target := range c.TargetList {
		u, err := url.Parse(target)
		if err != nil {
			return nil, err
		}
		targetList = append(targetList, u)
	}
	return targetList, nil
}

func LoadConfig(p string) (*Config, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}

	var c Config
	if err := json.NewDecoder(f).Decode(&c); err != nil {
		return nil, err
	}

	if err := c.validate(); err != nil {
		return nil, err
	}

	return &c, nil
}
