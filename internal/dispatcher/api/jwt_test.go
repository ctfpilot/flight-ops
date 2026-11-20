package api

import (
	"fmt"
	"testing"
	"time"

	jwtV5 "github.com/golang-jwt/jwt/v5"
)

func TestJwtFlight_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		domain *jwtFlight
		want   bool
	}{
		{
			name: "valid domain",
			domain: &jwtFlight{
				Admin:      true,
				Owner:      defaultOwner,
				Terminal:   defaultTerminal,
				FlightPlan: defaultFlightPlan,
				TTL:        3600,
				Action:     JwtActionList,
			},
			want: true,
		},
		{
			name:   "invalid domain - nil",
			domain: nil,
			want:   false,
		},
		{
			name: "invalid domain - empty action",
			domain: &jwtFlight{
				Admin:      true,
				Owner:      defaultOwner,
				Terminal:   defaultTerminal,
				FlightPlan: defaultFlightPlan,
				TTL:        3600,
				Action:     "",
			},
			want: false,
		},
		{
			name: "invalid domain - invalid action",
			domain: &jwtFlight{
				Admin:      true,
				Owner:      defaultOwner,
				Terminal:   defaultTerminal,
				FlightPlan: defaultFlightPlan,
				TTL:        3600,
				Action:     constInvalid,
			},
			want: false,
		},
		{
			name: "invalid domain - Empty owner",
			domain: &jwtFlight{
				Admin:      true,
				Owner:      "",
				Terminal:   defaultTerminal,
				FlightPlan: defaultFlightPlan,
				TTL:        3600,
				Action:     JwtActionList,
			},
			want: false,
		},
		{
			name: "invalid domain - Empty terminal",
			domain: &jwtFlight{
				Admin:      true,
				Owner:      defaultOwner,
				Terminal:   "",
				FlightPlan: defaultFlightPlan,
				TTL:        3600,
				Action:     JwtActionList,
			},
			want: false,
		},
		{
			name: "valid domain - Empty flight plan with list action",
			domain: &jwtFlight{
				Admin:      true,
				Owner:      defaultOwner,
				Terminal:   defaultTerminal,
				FlightPlan: "",
				TTL:        3600,
				Action:     JwtActionList,
			},
			want: true,
		},
		{
			name: "invalid domain - Empty flight plan with depart action",
			domain: &jwtFlight{
				Admin:      true,
				Owner:      defaultOwner,
				Terminal:   defaultTerminal,
				FlightPlan: "",
				TTL:        3600,
				Action:     JwtActionDepart,
			},
			want: false,
		},
		{
			name: "valid domain - depart with empty ttl",
			domain: &jwtFlight{
				Admin:      true,
				Owner:      defaultOwner,
				Terminal:   defaultTerminal,
				FlightPlan: defaultFlightPlan,
				Action:     JwtActionDepart,
			},
			want: true,
		},
		{
			name: "invalid domain - depart with negative ttl",
			domain: &jwtFlight{
				Admin:      true,
				Owner:      defaultOwner,
				Terminal:   defaultTerminal,
				FlightPlan: defaultFlightPlan,
				TTL:        -3600,
				Action:     JwtActionDepart,
			},
			want: false,
		},
		{
			name: "valid domain - action list",
			domain: &jwtFlight{
				Admin:      true,
				Owner:      defaultOwner,
				Terminal:   defaultTerminal,
				FlightPlan: defaultFlightPlan,
				TTL:        3600,
				Action:     JwtActionList,
			},
			want: true,
		},
		{
			name: "valid domain - action read",
			domain: &jwtFlight{
				Admin:      true,
				Owner:      defaultOwner,
				Terminal:   defaultTerminal,
				FlightPlan: defaultFlightPlan,
				TTL:        3600,
				Action:     JwtActionRead,
			},
			want: true,
		},
		{
			name: "valid domain - action depart",
			domain: &jwtFlight{
				Admin:      true,
				Owner:      defaultOwner,
				Terminal:   defaultTerminal,
				FlightPlan: defaultFlightPlan,
				TTL:        3600,
				Action:     JwtActionDepart,
			},
			want: true,
		},
		{
			name: "valid domain - action land",
			domain: &jwtFlight{
				Admin:      true,
				Owner:      defaultOwner,
				Terminal:   defaultTerminal,
				FlightPlan: defaultFlightPlan,
				TTL:        3600,
				Action:     JwtActionLand,
			},
			want: true,
		},
		{
			name: "valid domain - action refresh",
			domain: &jwtFlight{
				Admin:      true,
				Owner:      defaultOwner,
				Terminal:   defaultTerminal,
				FlightPlan: defaultFlightPlan,
				TTL:        3600,
				Action:     JwtActionRefresh,
			},
			want: true,
		},
		{
			name: "invalid owner name - invalid characters",
			domain: &jwtFlight{
				Admin:      true,
				Owner:      "Invalid Owner Name!",
				Terminal:   defaultTerminal,
				FlightPlan: defaultFlightPlan,
				TTL:        3600,
				Action:     JwtActionList,
			},
			want: false,
		},
		{
			name: "invalid terminal name - invalid characters",
			domain: &jwtFlight{
				Admin:      true,
				Owner:      defaultOwner,
				Terminal:   "Invalid Terminal Name!",
				FlightPlan: defaultFlightPlan,
				TTL:        3600,
				Action:     JwtActionList,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.domain.IsValid(); (got == nil) != tt.want {
				t.Errorf("JwtFlight.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJwtConfig_IsValid(t *testing.T) {

	tests := []struct {
		name   string
		config *JwtConfig
		want   bool
	}{
		{
			name: "valid config",
			config: &JwtConfig{
				Secret:     defaultSecret,
				Issuers:    []string{defaultIssuer1, defaultIssuer2},
				Audience:   defaultAudience,
				Expiration: 3600,
			},
			want: true,
		},
		{
			name:   "invalid config - nil",
			config: nil,
			want:   false,
		},
		{
			name: "invalid config - empty secret",
			config: &JwtConfig{
				Secret:     "",
				Issuers:    []string{defaultIssuer1, defaultIssuer2},
				Audience:   defaultAudience,
				Expiration: 3600,
			},
			want: false,
		},
		{
			name: "invalid config - empty issuers",
			config: &JwtConfig{
				Secret:     defaultSecret,
				Issuers:    []string{},
				Audience:   defaultAudience,
				Expiration: 3600,
			},
			want: false,
		},
		{
			name: "invalid config - empty audience",
			config: &JwtConfig{
				Secret:     defaultSecret,
				Issuers:    []string{defaultIssuer1, defaultIssuer2},
				Audience:   "",
				Expiration: 3600,
			},
			want: false,
		},
		{
			name: "invalid config - non-positive expiration",
			config: &JwtConfig{
				Secret:     defaultSecret,
				Issuers:    []string{defaultIssuer1, defaultIssuer2},
				Audience:   defaultAudience,
				Expiration: 0,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsValid(); (got == nil) != tt.want {
				t.Errorf("jwtConfig.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewJwt(t *testing.T) {
	config := &JwtConfig{
		Secret:     defaultSecret,
		Issuers:    []string{defaultIssuer1, defaultIssuer2},
		Audience:   defaultAudience,
		Expiration: 3600,
	}

	jwt := newJwt(config)
	if jwt.config != config {
		t.Errorf("NewJwt() config = %v, want %v", jwt.config, config)
	}
}

func TestJwt_isValid(t *testing.T) {
	tests := []struct {
		name   string
		domain *jwt
		want   bool
	}{
		{
			name: "valid jwt",
			domain: &jwt{
				config: &JwtConfig{
					Secret:     defaultSecret,
					Issuers:    []string{defaultIssuer1, defaultIssuer2},
					Audience:   defaultAudience,
					Expiration: 3600,
				},
				Expiration: time.Now().Add(time.Minute).Unix(),
				IssuedAt:   time.Now().Unix(),
				NotBefore:  time.Now().Unix(),
				Issuer:     defaultIssuer1,
				Audience:   []string{defaultAudience},
				JTI:        defaultJti,
				Flight: &jwtFlight{
					Admin:      true,
					Owner:      defaultOwner,
					Terminal:   defaultTerminal,
					FlightPlan: defaultFlightPlan,
					TTL:        3600,
					Action:     JwtActionList,
				},
			},
			want: true,
		},
		{
			name: "invalid jwt - nil config",
			domain: &jwt{
				config:     nil,
				Expiration: time.Now().Add(time.Minute).Unix(),
				IssuedAt:   time.Now().Unix(),
				NotBefore:  time.Now().Unix(),
				Issuer:     defaultIssuer1,
				Audience:   []string{defaultAudience},
				JTI:        defaultJti,
				Flight: &jwtFlight{
					Admin:      true,
					Owner:      defaultOwner,
					Terminal:   defaultTerminal,
					FlightPlan: defaultFlightPlan,
					TTL:        3600,
					Action:     JwtActionList,
				},
			},
			want: false,
		},
		{
			name: "invalid jwt - nil domain",
			domain: &jwt{
				config: &JwtConfig{
					Secret:     defaultSecret,
					Issuers:    []string{defaultIssuer1, defaultIssuer2},
					Audience:   defaultAudience,
					Expiration: 3600,
				},
				Expiration: time.Now().Add(time.Minute).Unix(),
				IssuedAt:   time.Now().Unix(),
				NotBefore:  time.Now().Unix(),
				Issuer:     defaultIssuer1,
				Audience:   []string{defaultAudience},
				JTI:        defaultJti,
				Flight:     nil,
			},
			want: false,
		},
		{
			name: "invalid jwt - empty expiration",
			domain: &jwt{
				config: &JwtConfig{
					Secret:     defaultSecret,
					Issuers:    []string{defaultIssuer1, defaultIssuer2},
					Audience:   defaultAudience,
					Expiration: 3600,
				},
				IssuedAt:  time.Now().Unix(),
				NotBefore: time.Now().Unix(),
				Issuer:    defaultIssuer1,
				Audience:  []string{defaultAudience},
				JTI:       defaultJti,
				Flight: &jwtFlight{
					Admin:      true,
					Owner:      defaultOwner,
					Terminal:   defaultTerminal,
					FlightPlan: defaultFlightPlan,
					TTL:        3600,
					Action:     JwtActionList,
				},
			},
			want: false,
		},
		{
			name: "invalid jwt - empty issuedAt",
			domain: &jwt{
				config: &JwtConfig{
					Secret:     defaultSecret,
					Issuers:    []string{defaultIssuer1, defaultIssuer2},
					Audience:   defaultAudience,
					Expiration: 3600,
				},
				Expiration: time.Now().Add(time.Minute).Unix(),
				NotBefore:  time.Now().Unix(),
				Issuer:     defaultIssuer1,
				Audience:   []string{defaultAudience},
				JTI:        defaultJti,
				Flight: &jwtFlight{
					Admin:      true,
					Owner:      defaultOwner,
					Terminal:   defaultTerminal,
					FlightPlan: defaultFlightPlan,
					TTL:        3600,
					Action:     JwtActionList,
				},
			},
			want: false,
		},
		{
			name: "invalid jwt - empty notBefore",
			domain: &jwt{
				config: &JwtConfig{
					Secret:     defaultSecret,
					Issuers:    []string{defaultIssuer1, defaultIssuer2},
					Audience:   defaultAudience,
					Expiration: 3600,
				},
				Expiration: time.Now().Add(time.Minute).Unix(),
				IssuedAt:   time.Now().Unix(),
				Issuer:     defaultIssuer1,
				Audience:   []string{defaultAudience},
				JTI:        defaultJti,
				Flight: &jwtFlight{
					Admin:      true,
					Owner:      defaultOwner,
					Terminal:   defaultTerminal,
					FlightPlan: defaultFlightPlan,
					TTL:        3600,
					Action:     JwtActionList,
				},
			},
			want: false,
		},
		{
			name: "invalid jwt - empty issuer",
			domain: &jwt{
				config: &JwtConfig{
					Secret:     defaultSecret,
					Issuers:    []string{defaultIssuer1, defaultIssuer2},
					Audience:   defaultAudience,
					Expiration: 3600,
				},
				Expiration: time.Now().Add(time.Minute).Unix(),
				IssuedAt:   time.Now().Unix(),
				NotBefore:  time.Now().Unix(),
				Audience:   []string{defaultAudience},
				JTI:        defaultJti,
				Flight: &jwtFlight{
					Admin:      true,
					Owner:      defaultOwner,
					Terminal:   defaultTerminal,
					FlightPlan: defaultFlightPlan,
					TTL:        3600,
					Action:     JwtActionList,
				},
			},
			want: false,
		},
		{
			name: "invalid jwt - empty audience",
			domain: &jwt{
				config: &JwtConfig{
					Secret:     defaultSecret,
					Issuers:    []string{defaultIssuer1, defaultIssuer2},
					Audience:   defaultAudience,
					Expiration: 3600,
				},
				Expiration: time.Now().Add(time.Minute).Unix(),
				IssuedAt:   time.Now().Unix(),
				NotBefore:  time.Now().Unix(),
				Issuer:     defaultIssuer1,
				JTI:        defaultJti,
				Flight: &jwtFlight{
					Admin:      true,
					Owner:      defaultOwner,
					Terminal:   defaultTerminal,
					FlightPlan: defaultFlightPlan,
					TTL:        3600,
					Action:     JwtActionList,
				},
			},
			want: false,
		},
		{
			name: "invalid jwt - empty jti",
			domain: &jwt{
				config: &JwtConfig{
					Secret:     defaultSecret,
					Issuers:    []string{defaultIssuer1, defaultIssuer2},
					Audience:   defaultAudience,
					Expiration: 3600,
				},
				Expiration: time.Now().Add(time.Minute).Unix(),
				IssuedAt:   time.Now().Unix(),
				NotBefore:  time.Now().Unix(),
				Issuer:     defaultIssuer1,
				Audience:   []string{defaultAudience},
				Flight: &jwtFlight{
					Admin:      true,
					Owner:      defaultOwner,
					Terminal:   defaultTerminal,
					FlightPlan: defaultFlightPlan,
					TTL:        3600,
					Action:     JwtActionList,
				},
			},
			want: false,
		},
		{
			name: "invalid jwt - too long experation",
			domain: &jwt{
				config: &JwtConfig{
					Secret:     defaultSecret,
					Issuers:    []string{defaultIssuer1, defaultIssuer2},
					Audience:   defaultAudience,
					Expiration: 3600,
				},
				Expiration: time.Now().Add(time.Hour * 2).Unix(),
				IssuedAt:   time.Now().Unix(),
				NotBefore:  time.Now().Unix(),
				Issuer:     defaultIssuer1,
				Audience:   []string{defaultAudience},
				JTI:        defaultJti,
				Flight: &jwtFlight{
					Admin:      true,
					Owner:      defaultOwner,
					Terminal:   defaultTerminal,
					FlightPlan: defaultFlightPlan,
					TTL:        3600,
					Action:     JwtActionList,
				},
			},
			want: false,
		},
		{
			name: "invalid jwt - expired",
			domain: &jwt{
				config: &JwtConfig{
					Secret:     defaultSecret,
					Issuers:    []string{defaultIssuer1, defaultIssuer2},
					Audience:   defaultAudience,
					Expiration: 3600,
				},
				Expiration: time.Now().Add(-time.Minute).Unix(),
				IssuedAt:   time.Now().Add(-time.Hour).Unix(),
				NotBefore:  time.Now().Add(-time.Hour).Unix(),
				Issuer:     defaultIssuer1,
				Audience:   []string{defaultAudience},
				JTI:        defaultJti,
				Flight: &jwtFlight{
					Admin:      true,
					Owner:      defaultOwner,
					Terminal:   defaultTerminal,
					FlightPlan: defaultFlightPlan,
					TTL:        3600,
					Action:     JwtActionList,
				},
			},
			want: false,
		},
		{
			name: "invalid jwt - future token",
			domain: &jwt{
				config: &JwtConfig{
					Secret:     defaultSecret,
					Issuers:    []string{defaultIssuer1, defaultIssuer2},
					Audience:   defaultAudience,
					Expiration: 3600,
				},
				Expiration: time.Now().Add(time.Hour).Unix(),
				IssuedAt:   time.Now().Add(time.Minute).Unix(),
				NotBefore:  time.Now().Add(time.Minute).Unix(),
				Issuer:     defaultIssuer1,
				Audience:   []string{defaultAudience},
				JTI:        defaultJti,
				Flight: &jwtFlight{
					Admin:      true,
					Owner:      defaultOwner,
					Terminal:   defaultTerminal,
					FlightPlan: defaultFlightPlan,
					TTL:        3600,
					Action:     JwtActionList,
				},
			},
			want: false,
		},
		{
			name: "invalid jwt - future token not before",
			domain: &jwt{
				config: &JwtConfig{
					Secret:     defaultSecret,
					Issuers:    []string{defaultIssuer1, defaultIssuer2},
					Audience:   defaultAudience,
					Expiration: 3600,
				},
				Expiration: time.Now().Add(time.Hour).Unix(),
				IssuedAt:   time.Now().Unix(),
				NotBefore:  time.Now().Add(time.Minute).Unix(),
				Issuer:     defaultIssuer1,
				Audience:   []string{defaultAudience},
				JTI:        defaultJti,
				Flight: &jwtFlight{
					Admin:      true,
					Owner:      defaultOwner,
					Terminal:   defaultTerminal,
					FlightPlan: defaultFlightPlan,
					TTL:        3600,
					Action:     JwtActionList,
				},
			},
			want: false,
		},
		{
			name: "invalid jwt - invalid issuer",
			domain: &jwt{
				config: &JwtConfig{
					Secret:     defaultSecret,
					Issuers:    []string{defaultIssuer1, defaultIssuer2},
					Audience:   defaultAudience,
					Expiration: 3600,
				},
				Expiration: time.Now().Add(time.Minute).Unix(),
				IssuedAt:   time.Now().Unix(),
				NotBefore:  time.Now().Unix(),
				Issuer:     "invalid-issuer",
				Audience:   []string{defaultAudience},
				JTI:        defaultJti,
				Flight: &jwtFlight{
					Admin:      true,
					Owner:      defaultOwner,
					Terminal:   defaultTerminal,
					FlightPlan: defaultFlightPlan,
					TTL:        3600,
					Action:     JwtActionList,
				},
			},
			want: false,
		},
		{
			name: "invalid jwt - invalid audience",
			domain: &jwt{
				config: &JwtConfig{
					Secret:     defaultSecret,
					Issuers:    []string{defaultIssuer1, defaultIssuer2},
					Audience:   defaultAudience,
					Expiration: 3600,
				},
				Expiration: time.Now().Add(time.Minute).Unix(),
				IssuedAt:   time.Now().Unix(),
				NotBefore:  time.Now().Unix(),
				Issuer:     defaultIssuer1,
				Audience:   []string{constInvalid},
				JTI:        defaultJti,
				Flight: &jwtFlight{
					Admin:      true,
					Owner:      defaultOwner,
					Terminal:   defaultTerminal,
					FlightPlan: defaultFlightPlan,
					TTL:        3600,
					Action:     JwtActionList,
				},
			},
			want: false,
		},
		{
			name: "valid jwt - multiple audience",
			domain: &jwt{
				config: &JwtConfig{
					Secret:     defaultSecret,
					Issuers:    []string{defaultIssuer1, defaultIssuer2},
					Audience:   defaultAudience,
					Expiration: 3600,
				},
				Expiration: time.Now().Add(time.Minute).Unix(),
				IssuedAt:   time.Now().Unix(),
				NotBefore:  time.Now().Unix(),
				Issuer:     defaultIssuer1,
				Audience:   []string{constInvalid, defaultAudience},
				JTI:        defaultJti,
				Flight: &jwtFlight{
					Admin:      true,
					Owner:      defaultOwner,
					Terminal:   defaultTerminal,
					FlightPlan: defaultFlightPlan,
					TTL:        3600,
					Action:     JwtActionList,
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.domain.IsValid(); (got == nil) != tt.want {
				t.Errorf("Jwt.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJwt_ParseToken(t *testing.T) {
	t.Run("invalid format token", func(t *testing.T) {
		config := &JwtConfig{
			Secret:     defaultSecret,
			Issuers:    []string{defaultIssuer1, defaultIssuer2},
			Audience:   defaultAudience,
			Expiration: 3600,
		}

		jwt := newJwt(config)

		err := jwt.ParseToken("test-token")
		if err == nil {
			t.Errorf("ParseToken() expected error for invalid token format, got nil")
		}
	})

	t.Run("invalid token method", func(t *testing.T) {
		config := &JwtConfig{
			Secret:     defaultSecret,
			Issuers:    []string{defaultIssuer1, defaultIssuer2},
			Audience:   defaultAudience,
			Expiration: 3600,
		}

		jwt := newJwt(config)

		err := jwt.ParseToken("eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.")
		if err == nil {
			t.Errorf("ParseToken() expected error for invalid token method, got nil")
		}
	})

	t.Run("invalid token content", func(t *testing.T) {
		config := &JwtConfig{
			Secret:     tokenSecret,
			Issuers:    []string{defaultIssuer1, defaultIssuer2},
			Audience:   defaultAudience,
			Expiration: 3600,
		}

		jwt := newJwt(config)

		err := jwt.ParseToken("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.x0IrNyyeWXFUc20uHF3uRZuPLRvucyCQzC2hLkSCK5A")
		if err == nil {
			t.Errorf("ParseToken() expected error for invalid token signature, got nil")
		}
	})

	t.Run("valid token content", func(t *testing.T) {
		config := &JwtConfig{
			Secret:     tokenSecret,
			Issuers:    []string{defaultIssuer1, defaultIssuer2},
			Audience:   defaultAudience,
			Expiration: 3600,
		}

		claim := jwtClaim{
			RegisteredClaims: jwtV5.RegisteredClaims{
				Issuer:    defaultIssuer1,
				Audience:  []string{defaultAudience},
				ExpiresAt: jwtV5.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwtV5.NewNumericDate(time.Now()),
				NotBefore: jwtV5.NewNumericDate(time.Now()),
				ID:        defaultJti,
			},
			Flight: jwtFlight{
				Admin:      true,
				Owner:      "1",
				Terminal:   "2",
				FlightPlan: "3",
				TTL:        123,
				Action:     JwtActionList,
			},
		}
		token, err := generateTestToken(config.Secret, claim)
		if err != nil {
			t.Errorf("generateTestToken() error = %v", err)
		}

		jwt := newJwt(config)

		err = jwt.ParseToken(token)
		if err != nil {
			t.Errorf("ParseToken() unexpected error = %v", err)
		}
	})

	t.Run("invalid token signature", func(t *testing.T) {
		config := &JwtConfig{
			Secret:     tokenSecret,
			Issuers:    []string{defaultIssuer1, defaultIssuer2},
			Audience:   defaultAudience,
			Expiration: 3600,
		}

		claim := jwtClaim{
			RegisteredClaims: jwtV5.RegisteredClaims{
				Issuer:    defaultIssuer1,
				Audience:  []string{defaultAudience},
				ExpiresAt: jwtV5.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwtV5.NewNumericDate(time.Now()),
				NotBefore: jwtV5.NewNumericDate(time.Now()),
				ID:        defaultJti,
			},
			Flight: jwtFlight{
				Admin:      true,
				Owner:      "1",
				Terminal:   "2",
				FlightPlan: "3",
				TTL:        123,
				Action:     JwtActionList,
			},
		}
		token, err := generateTestToken("super-duper-different-token-which-fails", claim)
		if err != nil {
			t.Errorf("generateTestToken() error = %v", err)
		}

		jwt := newJwt(config)

		err = jwt.ParseToken(token)
		if err == nil {
			t.Errorf("ParseToken() expected error for invalid token signature, got nil")
		}
	})

	t.Run("incorrect algorithm", func(t *testing.T) {
		config := &JwtConfig{
			Secret:     tokenSecret,
			Issuers:    []string{defaultIssuer1, defaultIssuer2},
			Audience:   defaultAudience,
			Expiration: 3600,
		}

		claim := jwtClaim{
			RegisteredClaims: jwtV5.RegisteredClaims{
				Issuer:    defaultIssuer1,
				Audience:  []string{defaultAudience},
				ExpiresAt: jwtV5.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwtV5.NewNumericDate(time.Now()),
				NotBefore: jwtV5.NewNumericDate(time.Now()),
				ID:        defaultJti,
			},
			Flight: jwtFlight{
				Admin:      true,
				Owner:      "1",
				Terminal:   "2",
				FlightPlan: "3",
				TTL:        123,
				Action:     JwtActionList,
			},
		}
		token, err := generateUnsignedTestToken(claim)
		if err != nil {
			t.Errorf("generateTestToken() error = %v", err)
		}

		jwt := newJwt(config)

		err = jwt.ParseToken(token)
		if err == nil {
			t.Errorf("ParseToken() expected error for invalid token signature, got nil")
		}
	})

	t.Run("invalid flight", func(t *testing.T) {
		config := &JwtConfig{
			Secret:     tokenSecret,
			Issuers:    []string{defaultIssuer1, defaultIssuer2},
			Audience:   defaultAudience,
			Expiration: 3600,
		}

		claim := jwtClaim{
			RegisteredClaims: jwtV5.RegisteredClaims{
				Issuer:    defaultIssuer1,
				Audience:  []string{defaultAudience},
				ExpiresAt: jwtV5.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwtV5.NewNumericDate(time.Now()),
				NotBefore: jwtV5.NewNumericDate(time.Now()),
				ID:        defaultJti,
			},
			Flight: jwtFlight{
				Admin:      true,
				Owner:      "",
				Terminal:   "",
				FlightPlan: "",
				TTL:        123,
				Action:     JwtActionDepart,
			},
		}
		token, err := generateUnsignedTestToken(claim)
		if err != nil {
			t.Errorf("generateTestToken() error = %v", err)
		}

		jwt := newJwt(config)

		err = jwt.ParseToken(token)
		if err == nil {
			t.Errorf("ParseToken() expected error for invalid flight, got nil")
		}
	})

}

func generateTestToken(secret string, claims jwtClaim) (string, error) {
	token := jwtV5.NewWithClaims(jwtV5.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign test token: %w", err)
	}
	return tokenStr, nil
}

func generateUnsignedTestToken(claims jwtClaim) (string, error) {
	token := jwtV5.NewWithClaims(jwtV5.SigningMethodNone, claims)
	tokenStr, err := token.SignedString(jwtV5.UnsafeAllowNoneSignatureType)
	if err != nil {
		return "", fmt.Errorf("failed to create unsigned test token: %w", err)
	}
	return tokenStr, nil
}
