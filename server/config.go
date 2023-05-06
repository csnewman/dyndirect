package server

type Config struct {
	RootDomain     string                  `mapstructure:"root_domain"`
	APIHost        string                  `mapstructure:"api_host"`
	APIListenHTTP  string                  `mapstructure:"api_listen_http"`
	APIListenHTTPS string                  `mapstructure:"api_listen_https"`
	APIBehindProxy bool                    `mapstructure:"api_behind_proxy"`
	CertFile       string                  `mapstructure:"tls_cert"`
	KeyFile        string                  `mapstructure:"tls_key"`
	ACMEEnabled    bool                    `mapstructure:"acme_enabled"`
	ACMEContact    string                  `mapstructure:"acme_contact"`
	StaticRecords  map[string]StaticRecord `mapstructure:"static_records"`
	TokenKey       string                  `mapstructure:"token_key"`
	Store          string                  `mapstructure:"store"`
	RedisAddr      string                  `mapstructure:"redis_addr"`
	RedisUser      string                  `mapstructure:"redis_user"`
	RedisPass      string                  `mapstructure:"redis_pass"`
	RedisDB        int                     `mapstructure:"redis_db"`
}

type StaticRecord struct {
	A []string
}
