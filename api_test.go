package bitmex

import (
	"github.com/micro/go-config"
	"github.com/micro/go-config/source/file"
	"testing"
)

var (
	_proxyURL string
	_testnet  bool
	_key      string
	_secret   string
)

func loadConfig() {
	err := config.Load(file.NewSource(file.WithPath("./testdata/config.yaml")))
	if err != nil {
		return
	}
	cfg := config.Map()

	_proxyURL = cfg["proxy_url"].(string)
	_testnet = cfg["testnet"].(bool)
	_key = cfg["key"].(string)
	_secret = cfg["secret"].(string)
}

func TestConfig(t *testing.T) {
	err := config.Load(file.NewSource(file.WithPath("./testdata/config.yaml")))
	if err != nil {
		t.Error(err)
	}
	cfg := config.Map()

	proxy := cfg["proxy"].(string)
	testnet := cfg["testnet"].(bool)
	key := cfg["key"].(string)
	secret := cfg["secret"].(string)

	t.Log(proxy)
	t.Log(testnet)
	t.Log(key)
	t.Log(secret)
}

func newBitmexForTest() *BitMEX {
	loadConfig()
	var host string
	if _testnet {
		host = HostTestnet
	} else {
		host = HostReal
	}
	bitmex := New(host, _key, _secret, "XBTUSD")
	if _proxyURL != "" {
		bitmex.SetHttpProxy(_proxyURL)
	}
	return bitmex
}
