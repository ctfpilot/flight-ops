package api

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ctfpilot/flight-ops/internal/dispatcher/lib"
	jwtV5 "github.com/golang-jwt/jwt/v5"
)

const (
	// JwtActionList indicates that the user is performing a list action on a flight or flight plan.
	JwtActionList string = "list"
	// JwtActionRead indicates that the user is performing a read action on a flight or flight plan.
	JwtActionRead string = "read"
	// JwtActionDepart indicates that the user is performing a depart action on a flight.
	JwtActionDepart string = "depart"
	// JwtActionLand indicates that the user is performing a land action on a flight.
	JwtActionLand string = "land"
	// JwtActionRefresh indicates that the user is performing a refresh action on a flight or flight plan.
	JwtActionRefresh string = "refresh"
)

type jwtFlight struct {
	// Admin indicates if the user associated with the JWT token has admin privileges
	Admin bool `json:"admin"`
	// Owner is the owner associated with the JWT token.
	Owner string `json:"owner"`
	// Terminal indicates from what terminal instance the user is accessing the API.
	Terminal string `json:"terminal"`
	// FlightPlan indicates what flight plan to act upon. May be empty if not operating on a flight or flight plan.
	FlightPlan string `json:"flight_plan"`
	// Flight indicates what flight to act upon. May be empty if not operating on a flight or flight plan.
	Flight string `json:"flight"`
	// TTL indicates the TTL for the Flight. May be empty if using default TTL. If set, it must be a positive integer.
	TTL int64 `json:"ttl"`
	// Action indicates what action the user is performing. May be empty if not operating on a flight or flight plan.
	Action string `json:"action"`
}

func (d *jwtFlight) IsValid() error {
	if d == nil {
		return fmt.Errorf("jwtFlight is nil")
	}
	if d.Action == "" {
		return fmt.Errorf("jwtFlight action is empty")
	}

	// Validate that the action is one of the allowed actions
	allowedActions := []string{JwtActionList, JwtActionRead, JwtActionDepart, JwtActionLand, JwtActionRefresh}
	if !slices.Contains(allowedActions, d.Action) {
		return fmt.Errorf("jwtFlight action '%s' is not valid. Allowed actions are: %v", d.Action, allowedActions)
	}

	// Owner and Terminal must be non-empty strings
	if strings.TrimSpace(d.Owner) == "" || strings.TrimSpace(d.Terminal) == "" {
		return fmt.Errorf("jwtFlight owner and terminal must be non-empty strings. Owner: '%s', Terminal: '%s'", d.Owner, d.Terminal)
	}
	if !lib.ValidK8sName(d.Owner) {
		return fmt.Errorf("jwtFlight owner '%s' is not a valid Kubernetes resource name", d.Owner)
	}
	if !lib.ValidK8sName(d.Terminal) {
		return fmt.Errorf("jwtFlight terminal '%s' is not a valid Kubernetes resource name", d.Terminal)
	}

	// FlightPlan must not be empty departing or landing
	if d.Action == JwtActionDepart || d.Action == JwtActionLand || d.Action == JwtActionRefresh {
		if strings.TrimSpace(d.FlightPlan) == "" && strings.TrimSpace(d.Flight) == "" {
			return fmt.Errorf("jwtFlight flightPlan or flight must be non-empty for action '%s'. FlightPlan: '%s', Flight: '%s'", d.Action, d.FlightPlan, d.Flight)
		}
	}

	// TTL must be a positive integer if set
	if d.TTL < 0 {
		return fmt.Errorf("jwtFlight TTL must be a positive integer. TTL: %d", d.TTL)
	}

	return nil
}

type JwtConfig struct {
	// Secret is the secret key used to sign the JWT token
	Secret string `json:"-"`
	// Issuers is a list of valid issuers for the JWT token
	Issuers []string `json:"issuers"`
	// Audience is the intended audience for the JWT token
	Audience string `json:"audience"`
	// Expiration is the maximum expiration time for the JWT token in seconds
	Expiration int64 `json:"expiration"`
}

func (c *JwtConfig) IsValid() error {
	if c == nil {
		return fmt.Errorf("JWT Config is nil")
	}
	if c.Secret == "" || c.Audience == "" || len(c.Issuers) == 0 || c.Expiration <= 0 {
		return fmt.Errorf("JWT Config is invalid")
	}
	return nil
}

type jwt struct {
	// config is the configuration used to generate the JWT token
	config *JwtConfig

	// Expiration is the expiration time of the JWT token in Unix timestamp format
	Expiration int64 `json:"exp"`
	// IssuedAt is the time when the JWT token was issued in Unix timestamp format
	IssuedAt int64 `json:"iat"`
	// NotBefore is the time before which the JWT token is not valid in Unix timestamp format
	NotBefore int64 `json:"nbf"`
	// Issuer is the issuer of the JWT token
	Issuer string `json:"iss"`
	// Audience is the intended audience of the JWT token
	Audience []string `json:"aud"`
	// JTI is the unique identifier for the JWT token
	JTI string `json:"jti"`

	Flight *jwtFlight `json:"flight"`
}

func newJwt(config *JwtConfig) *jwt {
	return &jwt{
		config: config,
	}
}

func (j *jwt) IsValid() error {
	if j == nil {
		return fmt.Errorf("jwt is nil")
	}
	if j.config == nil {
		return fmt.Errorf("jwt config is nil")
	}
	if err := j.config.IsValid(); err != nil {
		return errors.New("jwt config is invalid: " + err.Error())
	}

	// Validate required fields
	if j.Expiration <= 0 || j.IssuedAt <= 0 || j.NotBefore <= 0 || j.Issuer == "" || len(j.Audience) == 0 || j.JTI == "" || j.Flight == nil {
		return fmt.Errorf("jwt is missing required fields")
	}

	// Validate tha the experation time is not greater than the configured maximum expiration time
	if j.Expiration-j.IssuedAt > j.config.Expiration {
		return fmt.Errorf("jwt expiration time exceeds maximum allowed expiration time. Expiration: %d, IssuedAt: %d, MaxExpiration: %d", j.Expiration, j.IssuedAt, j.config.Expiration)
	}

	// Validate expiration, issuedAt, and notBefore timestamps
	currentTime := time.Now().Unix()
	if j.Expiration < currentTime || j.IssuedAt > currentTime || j.NotBefore > currentTime {
		return fmt.Errorf("jwt timestamps are invalid. Expiration: %d, IssuedAt: %d, NotBefore: %d, CurrentTime: %d", j.Expiration, j.IssuedAt, j.NotBefore, currentTime)
	}

	// Validate issuer and audience
	if !slices.Contains(j.config.Issuers, j.Issuer) || !slices.Contains(j.Audience, j.config.Audience) {
		return fmt.Errorf("jwt issuer or audience is invalid. Issuer: %s, Audience: %v, ConfigIssuers: %v, ConfigAudience: %s", j.Issuer, j.Audience, j.config.Issuers, j.config.Audience)
	}

	// Validate domain
	if err := j.Flight.IsValid(); err != nil {
		return fmt.Errorf("jwt flight is invalid: %w", err)
	}

	return nil
}

type jwtClaim struct {
	jwtV5.RegisteredClaims
	Flight jwtFlight `json:"flight"`
}

func (j *jwt) ParseToken(token string) error {
	if j == nil {
		return fmt.Errorf("jwt is nil")
	}
	if j.config == nil {
		return fmt.Errorf("jwt config is nil")
	}

	claim := &jwtClaim{}

	jwtToken, err := jwtV5.ParseWithClaims(token, claim, func(token *jwtV5.Token) (any, error) {
		// Validate method (alg) is HMAC
		if _, ok := token.Method.(*jwtV5.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(j.config.Secret), nil
	})
	if err != nil {
		return fmt.Errorf("failed to parse JWT: %w", err)
	}

	if !jwtToken.Valid {
		return fmt.Errorf("token is invalid or expired")
	}

	// Populate the jwt struct with the parsed claims
	if claim.ExpiresAt != nil {
		j.Expiration = claim.ExpiresAt.Unix()
	}
	if claim.IssuedAt != nil {
		j.IssuedAt = claim.IssuedAt.Unix()
	}
	if claim.NotBefore != nil {
		j.NotBefore = claim.NotBefore.Unix()
	}

	j.Issuer = claim.Issuer
	if len(claim.Audience) > 0 {
		j.Audience = claim.Audience
	}
	j.JTI = claim.ID
	j.Flight = &claim.Flight

	// Validate the token is now valid after parsing
	if err := j.IsValid(); err != nil {
		return fmt.Errorf("parsed token is invalid: %w", err)
	}

	return nil
}
