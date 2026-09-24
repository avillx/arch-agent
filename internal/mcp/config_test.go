package mcp

import (
	"context"
	"errors"
	"testing"

	"arch-agent/internal/types"
)

func validationProblem(t *testing.T, err error) map[string]string {
	t.Helper()

	var validationErr *types.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *types.ValidationError, got %T: %v", err, err)
	}
	return validationErr.Problems()
}

func TestServerGatewayConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		cfg         ServerGatewayConfig
		wantProblem string
	}{
		{
			name:        "empty config",
			cfg:         ServerGatewayConfig{},
			wantProblem: "config",
		},
		{
			name: "both gateways",
			cfg: ServerGatewayConfig{
				HTTPGateway:    &HTTPGatewayConfig{URL: "http://example.com"},
				CommandGateway: &CommandGatewayConfig{Command: "npm"},
			},
			wantProblem: "config",
		},
		{
			name:        "http url empty",
			cfg:         ServerGatewayConfig{HTTPGateway: &HTTPGatewayConfig{}},
			wantProblem: "http_gateway",
		},
		{
			name:        "command empty",
			cfg:         ServerGatewayConfig{CommandGateway: &CommandGatewayConfig{}},
			wantProblem: "command_gateway",
		},
		{
			name: "http valid",
			cfg:  ServerGatewayConfig{HTTPGateway: &HTTPGatewayConfig{URL: "http://example.com"}},
		},
		{
			name: "command valid",
			cfg:  ServerGatewayConfig{CommandGateway: &CommandGatewayConfig{Command: "npm"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate(context.Background())
			if tt.wantProblem == "" {
				if err != nil {
					t.Fatalf("expected valid config, got %v", err)
				}
				return
			}

			problems := validationProblem(t, err)
			if _, ok := problems[tt.wantProblem]; !ok {
				t.Fatalf("expected problem field %q, got %v", tt.wantProblem, problems)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ServerGatewayConfig
		wantErr bool
	}{
		{
			name:    "both gateways",
			cfg:     ServerGatewayConfig{HTTPGateway: &HTTPGatewayConfig{}, CommandGateway: &CommandGatewayConfig{}},
			wantErr: true,
		},
		{
			name:    "no gateways",
			cfg:     ServerGatewayConfig{},
			wantErr: true,
		},
		{
			name: "http only",
			cfg:  ServerGatewayConfig{HTTPGateway: &HTTPGatewayConfig{}},
		},
		{
			name: "command only",
			cfg:  ServerGatewayConfig{CommandGateway: &CommandGatewayConfig{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.cfg)
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("expected valid config, got %v", err)
				}
				return
			}

			problems := validationProblem(t, err)
			if _, ok := problems["config"]; !ok {
				t.Fatalf("expected problem field %q, got %v", "config", problems)
			}
		})
	}
}

func TestServerGatewayConfigEquals(t *testing.T) {
	http := &HTTPGatewayConfig{URL: "http://example.com", Token: "token"}
	httpSame := &HTTPGatewayConfig{URL: "http://example.com", Token: "token"}
	httpOtherToken := &HTTPGatewayConfig{URL: "http://example.com", Token: "other"}
	httpEmptyToken := &HTTPGatewayConfig{URL: "http://example.com"}

	command := &CommandGatewayConfig{
		Command: "npm",
		Args:    []string{"run", "stdio"},
		Env:     map[string]string{"KEY": "value"},
	}
	commandSame := &CommandGatewayConfig{
		Command: "npm",
		Args:    []string{"run", "stdio"},
		Env:     map[string]string{"KEY": "value"},
	}
	commandOtherArgs := &CommandGatewayConfig{
		Command: "npm",
		Args:    []string{"run"},
		Env:     map[string]string{"KEY": "value"},
	}
	commandOtherEnv := &CommandGatewayConfig{
		Command: "npm",
		Args:    []string{"run", "stdio"},
		Env:     map[string]string{"KEY": "other"},
	}

	tests := []struct {
		name string
		a    ServerGatewayConfig
		b    ServerGatewayConfig
		want bool
	}{
		{
			name: "empty",
			a:    ServerGatewayConfig{},
			b:    ServerGatewayConfig{},
			want: true,
		},
		{
			name: "same http",
			a:    ServerGatewayConfig{HTTPGateway: http},
			b:    ServerGatewayConfig{HTTPGateway: httpSame},
			want: true,
		},
		{
			name: "different http token",
			a:    ServerGatewayConfig{HTTPGateway: http},
			b:    ServerGatewayConfig{HTTPGateway: httpOtherToken},
		},
		{
			name: "set token vs empty token",
			a:    ServerGatewayConfig{HTTPGateway: http},
			b:    ServerGatewayConfig{HTTPGateway: httpEmptyToken},
		},
		{
			name: "same command",
			a:    ServerGatewayConfig{CommandGateway: command},
			b:    ServerGatewayConfig{CommandGateway: commandSame},
			want: true,
		},
		{
			name: "different command args",
			a:    ServerGatewayConfig{CommandGateway: command},
			b:    ServerGatewayConfig{CommandGateway: commandOtherArgs},
		},
		{
			name: "different command env",
			a:    ServerGatewayConfig{CommandGateway: command},
			b:    ServerGatewayConfig{CommandGateway: commandOtherEnv},
		},
		{
			name: "http vs command",
			a:    ServerGatewayConfig{HTTPGateway: http},
			b:    ServerGatewayConfig{CommandGateway: command},
		},
		{
			name: "nil http vs set http",
			a:    ServerGatewayConfig{},
			b:    ServerGatewayConfig{HTTPGateway: http},
		},
		{
			name: "nil command vs set command",
			a:    ServerGatewayConfig{},
			b:    ServerGatewayConfig{CommandGateway: command},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Equals(tt.b); got != tt.want {
				t.Fatalf("Equals() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHTTPGatewayConfigEquals(t *testing.T) {
	a := &HTTPGatewayConfig{URL: "http://example.com", Token: "token"}
	b := &HTTPGatewayConfig{URL: "http://example.com", Token: "token"}
	c := &HTTPGatewayConfig{URL: "http://example.com", Token: "other"}
	d := &HTTPGatewayConfig{URL: "http://other.com", Token: "token"}

	checks := []struct {
		name string
		a    *HTTPGatewayConfig
		b    *HTTPGatewayConfig
		want bool
	}{
		{"nil and nil", nil, nil, true},
		{"nil and set", nil, a, false},
		{"set and nil", a, nil, false},
		{"equal", a, b, true},
		{"different token", a, c, false},
		{"different url", a, d, false},
	}

	for _, tt := range checks {
		if got := tt.a.Equals(tt.b); got != tt.want {
			t.Fatalf("%s: Equals() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestCommandGatewayConfigEquals(t *testing.T) {
	a := &CommandGatewayConfig{
		Command: "npm",
		Args:    []string{"run", "stdio"},
		Env:     map[string]string{"KEY": "value"},
	}
	b := &CommandGatewayConfig{
		Command: "npm",
		Args:    []string{"run", "stdio"},
		Env:     map[string]string{"KEY": "value"},
	}
	c := &CommandGatewayConfig{
		Command: "npm",
		Args:    []string{"run"},
		Env:     map[string]string{"KEY": "value"},
	}
	d := &CommandGatewayConfig{
		Command: "npm",
		Args:    []string{"run", "stdio"},
		Env:     map[string]string{"KEY": "other"},
	}
	e := &CommandGatewayConfig{
		Command: "other",
		Args:    []string{"run", "stdio"},
		Env:     map[string]string{"KEY": "value"},
	}

	checks := []struct {
		name string
		a    *CommandGatewayConfig
		b    *CommandGatewayConfig
		want bool
	}{
		{"nil and nil", nil, nil, true},
		{"nil and set", nil, a, false},
		{"set and nil", a, nil, false},
		{"equal", a, b, true},
		{"different args", a, c, false},
		{"different env", a, d, false},
		{"different command", a, e, false},
	}

	for _, tt := range checks {
		if got := tt.a.Equals(tt.b); got != tt.want {
			t.Fatalf("%s: Equals() = %v, want %v", tt.name, got, tt.want)
		}
	}
}
