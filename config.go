package httpmiddleware

type Config struct {
	ExcludeOpt        *ExcludeOption
	DisableIngressLog bool // true: add important info to context and disable default ingress log (usecase: custom logging implementation), default value: false
	FieldOpt          *FieldOption
}

type ExcludeOption struct {
	RequestHeader       bool
	RequestBody         bool
	ResponseHeader      bool
	ResponseBody        bool
	SuccessResponseBody bool
	SuccessRequest      bool
	RequestHeaderKeys   []string
}

type FieldOption struct {
	EventPrefix string
}

func defaultConfig() *Config {
	return &Config{
		ExcludeOpt: &ExcludeOption{},
	}
}

func NewConfig(c *Config) *Config {
	if c.ExcludeOpt == nil {
		c.ExcludeOpt = &ExcludeOption{}
	}

	return c
}

func (c *Config) LogRequestHeader() bool {
	if c.ExcludeOpt == nil {
		return IncludeLog
	}

	return c.ExcludeOpt.RequestHeader == IncludeLog
}

func (c *Config) LogRequestBody() bool {
	if c.ExcludeOpt == nil {
		return IncludeLog
	}

	return c.ExcludeOpt.RequestBody == IncludeLog
}

func (c *Config) LogResponseHeader() bool {
	if c.ExcludeOpt == nil {
		return IncludeLog
	}

	return c.ExcludeOpt.ResponseHeader == IncludeLog
}

func (c *Config) LogResponseBody() bool {
	if c.ExcludeOpt == nil {
		return IncludeLog
	}

	return c.ExcludeOpt.ResponseBody == IncludeLog
}

func (c *Config) LogSuccessResponseBody() bool {
	if c.ExcludeOpt == nil {
		return IncludeLog
	}

	return c.ExcludeOpt.SuccessResponseBody == IncludeLog
}

func (c *Config) LogFailedRequestOnly() bool {
	if c.ExcludeOpt == nil {
		return IncludeLog
	}

	return c.ExcludeOpt.SuccessRequest == ExcludeLog
}

func (c *Config) GetEventPrefix() string {
	if c.FieldOpt == nil || len(c.FieldOpt.EventPrefix) == 0 {
		return EventPrefix + URLSeparator
	}

	return c.FieldOpt.EventPrefix + URLSeparator
}
