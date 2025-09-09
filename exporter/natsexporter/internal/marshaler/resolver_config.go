package marshaler

type resolverConfig struct {
	BuiltinMarshalerName  BuiltinMarshalerName `mapstructure:"marshaler"`
	EncodingExtensionName []byte               `mapstructure:"encoding_extension"`
}

func (c *resolverConfig) newResolver() (resolver, error) {
	if c.EncodingExtensionName != nil {
		return newEncodingExtensionResolver(c.EncodingExtensionName)
	} else {
		return newBuiltinMarshalerResolver(c.BuiltinMarshalerName)
	}
}

func newDefaultResolverConfig() resolverConfig {
	return resolverConfig{
		BuiltinMarshalerName:  OtlpProtoBuiltinMarshalerName,
		EncodingExtensionName: nil,
	}
}
