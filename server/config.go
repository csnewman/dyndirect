package server

type Config struct {
	RootDomain    string                  `mapstructure:"root_domain"`
	StaticRecords map[string]StaticRecord `mapstructure:"static_records"`
}

type StaticRecord struct {
	A []string
}
