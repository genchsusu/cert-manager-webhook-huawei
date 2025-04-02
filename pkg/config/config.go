package config

import (
	"encoding/json"
	"fmt"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// HuaweiDNSProviderConfig 用于存放华为云 DNS 的配置参数
type HuaweiDNSProviderConfig struct {
	Region    string `json:"region"`
	ZoneID    string `json:"zoneId"`
	AccessKey string `json:"appKey"`
	SecretKey string `json:"appSecret"`
}

// LoadConfig 从 CRD 的 JSON 数据中解析出配置
func LoadConfig(cfgJSON *extapi.JSON) (*HuaweiDNSProviderConfig, error) {
	cfg := &HuaweiDNSProviderConfig{}
	if cfgJSON == nil {
		return cfg, nil
	}
	if err := json.Unmarshal(cfgJSON.Raw, cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse solver config: %v", err)
	}
	return cfg, nil
}
