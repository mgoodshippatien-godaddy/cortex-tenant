package main

import (
	"fmt"
	"os"
	"time"

	"github.com/caarlos0/env/v8"
	"github.com/pkg/errors"
	fhu "github.com/valyala/fasthttp/fasthttputil"
	"gopkg.in/yaml.v2"
)

type config struct {
	Listen               string `env:"CT_LISTEN"`
	ListenPprof          string `env:"CT_LISTEN_PPROF"           yaml:"listen_pprof"`
	ListenMetricsAddress string `env:"CT_LISTEN_METRICS_ADDRESS" yaml:"listen_metrics_address"`
	MetricsIncludeTenant bool   `env:"CT_METRICS_INCLUDE_TENANT" yaml:"metrics_include_tenant"`

	Target struct {
		Endpoint           string `yaml:"endpoint" env:"CT_TARGET_ENDPOINT"`
		CertFile           string `yaml:"cert_file" env:"CT_TARGET_CERT_FILE"`
		KeyFile            string `yaml:"key_file" env:"CT_TARGET_KEY_FILE"`
		CAFile             string `yaml:"ca_file" env:"CT_TARGET_CA_FILE"`
		InsecureSkipVerify bool   `yaml:"insecure_skip_verify" env:"CT_TARGET_INSECURE_SKIP_VERIFY"`
	} `yaml:"target"`

	EnableIPv6 bool `yaml:"enable_ipv6" env:"CT_ENABLE_IPV6"`

	LogLevel          string        `yaml:"log_level"               env:"CT_LOG_LEVEL"`
	Timeout           time.Duration `                               env:"CT_TIMEOUT"`
	TimeoutShutdown   time.Duration `yaml:"timeout_shutdown"        env:"CT_TIMEOUT_SHUTDOWN"`
	Concurrency       int           `                               env:"CT_CONCURRENCY"`
	Metadata          bool          `                               env:"CT_METADATA"`
	LogResponseErrors bool          `yaml:"log_response_errors"     env:"CT_LOG_RESPONSE_ERRORS"`
	MaxConnDuration   time.Duration `yaml:"max_connection_duration" env:"CT_MAX_CONN_DURATION"`
	MaxConnsPerHost   int           `yaml:"max_conns_per_host"      env:"CT_MAX_CONNS_PER_HOST"`

	Auth struct {
		Egress struct {
			Username string `env:"CT_AUTH_EGRESS_USERNAME"`
			Password string `env:"CT_AUTH_EGRESS_PASSWORD"`
		}
	}

	Tenant struct {
		Label              string   `env:"CT_TENANT_LABEL"`
		LabelList          []string `yaml:"label_list" env:"CT_TENANT_LABEL_LIST" envSeparator:","`
		Prefix             string   `yaml:"prefix" env:"CT_TENANT_PREFIX"`
		PrefixPreferSource bool     `yaml:"prefix_prefer_source" env:"CT_TENANT_PREFIX_PREFER_SOURCE"`
		LabelRemove        bool     `yaml:"label_remove" env:"CT_TENANT_LABEL_REMOVE"`
		Header             string   `env:"CT_TENANT_HEADER"`
		Default            string   `env:"CT_TENANT_DEFAULT"`
		AcceptAll          bool     `yaml:"accept_all" env:"CT_TENANT_ACCEPT_ALL"`
	}

	pipeIn  *fhu.InmemoryListener
	pipeOut *fhu.InmemoryListener
}

func configLoad(file string) (*config, error) {
	cfg := &config{}

	if file != "" {
		y, err := os.ReadFile(file)
		if err != nil {
			return nil, errors.Wrap(err, "Unable to read config")
		}

		if err := yaml.UnmarshalStrict(y, cfg); err != nil {
			return nil, errors.Wrap(err, "Unable to parse config")
		}
	}

	if err := env.Parse(cfg); err != nil {
		return nil, errors.Wrap(err, "Unable to parse env vars")
	}

	if cfg.Listen == "" {
		cfg.Listen = "127.0.0.1:8081"
	}

	if cfg.ListenMetricsAddress == "" {
		cfg.ListenMetricsAddress = "0.0.0.0:9090"
	}

	if cfg.LogLevel == "" {
		cfg.LogLevel = "warn"
	}

	if cfg.Target.Endpoint == "" {
		cfg.Target.Endpoint = "127.0.0.1:9090"
	}

	if cfg.Target.CertFile != "" {
		_, err := os.Stat(cfg.Target.CertFile)
		if err != nil {
			return nil, errors.Wrap(err, "Unable to find cert file")
		}
	}

	if cfg.Target.KeyFile != "" {
		_, err := os.Stat(cfg.Target.KeyFile)
		if err != nil {
			return nil, errors.Wrap(err, "Unable to find key file")
		}
	}

	if cfg.Target.CAFile != "" {
		_, err := os.Stat(cfg.Target.CAFile)
		if err != nil {
			return nil, errors.Wrap(err, "Unable to find CA file")
		}
	}

	if cfg.Target.InsecureSkipVerify {
		cfg.Target.InsecureSkipVerify = false
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}

	if cfg.Concurrency == 0 {
		cfg.Concurrency = 512
	}

	if cfg.Tenant.Header == "" {
		cfg.Tenant.Header = "X-Scope-OrgID"
	}

	if cfg.Tenant.Label == "" {
		cfg.Tenant.Label = "__tenant__"
	}

	// Default to the Label if list is empty
	if len(cfg.Tenant.LabelList) == 0 {
		cfg.Tenant.LabelList = append(cfg.Tenant.LabelList, cfg.Tenant.Label)
	}

	if cfg.Auth.Egress.Username != "" {
		if cfg.Auth.Egress.Password == "" {
			return nil, fmt.Errorf("egress auth user specified, but the password is not")
		}
	}

	if cfg.MaxConnsPerHost == 0 {
		cfg.MaxConnsPerHost = 64
	}

	return cfg, nil
}
