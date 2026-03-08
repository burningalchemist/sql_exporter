package config

import "context"

// Secret special type for storing secrets.
type Secret string

// UnmarshalYAML implements the yaml.Unmarshaler interface for Secrets.
func (s *Secret) UnmarshalYAML(unmarshal func(any) error) error {
	type plain Secret
	if err := unmarshal((*plain)(s)); err != nil {
		return err
	}

	resolved, err := resolveSecretDSN(context.TODO(), string(*s))
	if err != nil {
		return err
	}
	*s = Secret(resolved)
	return nil
}

// MarshalYAML implements the yaml.Marshaler interface for Secrets.
func (s Secret) MarshalYAML() (any, error) {
	if s != "" {
		return "<secret>", nil
	}
	return nil, nil
}
