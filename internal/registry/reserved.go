package registry

import (
	"fmt"
	"regexp"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

type ReservedConfig struct {
	Words []string `yaml:"reserved"`
}

func ParseReserved(src []byte) (ReservedConfig, error) {
	var cfg ReservedConfig
	if err := yaml.Unmarshal(src, &cfg); err != nil {
		return ReservedConfig{}, fmt.Errorf("registry: %w", err)
	}
	return cfg, nil
}

var previewPrefixPattern = regexp.MustCompile(`^(?i)pr-[0-9]+$`)

const minSubdomainLength = 3

func (cfg ReservedConfig) IsReserved(subdomain string) bool {
	if len(subdomain) < minSubdomainLength {
		return true
	}
	if previewPrefixPattern.MatchString(subdomain) {
		return true
	}
	for _, w := range cfg.Words {
		if strings.EqualFold(w, subdomain) {
			return true
		}
	}
	return false
}
