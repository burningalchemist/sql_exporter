package config

// Secret special type for storing secrets.
type Secret string

// MarshalYAML implements the yaml.Marshaler interface for Secrets.
func (s Secret) MarshalYAML() (any, error) {
	if s != "" {
		return "<secret>", nil
	}
	return nil, nil
}
