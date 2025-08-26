// Copyright 2024 The Cloud Native Events Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package restapi

import (
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

// initMTLSCACertPool initializes the CA certificate pool for mTLS
func (s *Server) initMTLSCACertPool() error {
	if s.authConfig == nil || !s.authConfig.EnableMTLS || s.authConfig.CACertPath == "" {
		return nil
	}

	caCert, err := os.ReadFile(s.authConfig.CACertPath)
	if err != nil {
		log.Errorf("failed to read CA certificate: %v", err)
		return err
	}

	s.caCertPool = x509.NewCertPool()
	if !s.caCertPool.AppendCertsFromPEM(caCert) {
		log.Error("failed to parse CA certificate")
		return fmt.Errorf("failed to parse CA certificate")
	}

	log.Info("mTLS CA certificate pool initialized")
	return nil
}

// mTLSMiddleware validates client certificates for mTLS authentication
func (s *Server) mTLSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.authConfig == nil || !s.authConfig.EnableMTLS {
			next.ServeHTTP(w, r)
			return
		}

		if r.TLS == nil {
			log.Error("mTLS required but request is not over TLS")
			http.Error(w, "mTLS required", http.StatusUnauthorized)
			return
		}

		if len(r.TLS.PeerCertificates) == 0 {
			log.Error("no client certificate provided")
			http.Error(w, "Client certificate required", http.StatusUnauthorized)
			return
		}

		clientCert := r.TLS.PeerCertificates[0]

		// Verify the client certificate against our CA
		opts := x509.VerifyOptions{
			Roots: s.caCertPool,
		}

		if _, err := clientCert.Verify(opts); err != nil {
			log.Errorf("client certificate verification failed: %v", err)
			http.Error(w, "Invalid client certificate", http.StatusUnauthorized)
			return
		}

		log.Infof("mTLS authentication successful for client: %s", clientCert.Subject.CommonName)
		next.ServeHTTP(w, r)
	})
}

// OAuthClaims represents the claims in an OAuth JWT token
type OAuthClaims struct {
	Issuer    string   `json:"iss"`
	Subject   string   `json:"sub"`
	Audience  []string `json:"aud"`
	ExpiresAt int64    `json:"exp"`
	IssuedAt  int64    `json:"iat"`
	Scopes    []string `json:"scope"`
}

// oAuthMiddleware validates OAuth Bearer tokens
func (s *Server) oAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.authConfig == nil || !s.authConfig.EnableOAuth {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Error("missing Authorization header")
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			log.Error("invalid Authorization header format")
			http.Error(w, "Bearer token required", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			log.Error("empty Bearer token")
			http.Error(w, "Invalid Bearer token", http.StatusUnauthorized)
			return
		}

		// Validate the token (simplified validation - in production, use proper JWT library)
		if err := s.validateOAuthToken(token); err != nil {
			log.Errorf("OAuth token validation failed: %v", err)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		log.Info("OAuth authentication successful")
		next.ServeHTTP(w, r)
	})
}

// validateOAuthToken validates the OAuth token
// Note: This is a simplified implementation. In production, use a proper JWT library
// like github.com/golang-jwt/jwt or github.com/lestrrat-go/jwx
func (s *Server) validateOAuthToken(token string) error {
	// Simplified token validation
	// In production, you should:
	// 1. Parse and verify JWT signature using JWKS from the OAuth provider
	// 2. Validate issuer, audience, expiration, etc.
	// 3. Check required scopes

	if len(token) < 10 {
		return fmt.Errorf("token too short")
	}

	// For demonstration purposes, accept any token that starts with "valid_"
	// Replace this with proper JWT validation in production
	if !strings.HasPrefix(token, "valid_") {
		return fmt.Errorf("invalid token format")
	}

	// In production, decode JWT and validate claims here
	// Example validation logic:
	/*
		var claims OAuthClaims
		// Parse JWT token and extract claims

		// Validate issuer
		if s.authConfig.OAuthIssuer != "" && claims.Issuer != s.authConfig.OAuthIssuer {
			return fmt.Errorf("invalid issuer")
		}

		// Validate audience
		if s.authConfig.RequiredAudience != "" {
			found := false
			for _, aud := range claims.Audience {
				if aud == s.authConfig.RequiredAudience {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("invalid audience")
			}
		}

		// Validate scopes
		if len(s.authConfig.RequiredScopes) > 0 {
			for _, requiredScope := range s.authConfig.RequiredScopes {
				found := false
				for _, scope := range claims.Scopes {
					if scope == requiredScope {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("missing required scope: %s", requiredScope)
				}
			}
		}

		// Validate expiration
		if time.Now().Unix() > claims.ExpiresAt {
			return fmt.Errorf("token expired")
		}
	*/

	return nil
}

// combinedAuthMiddleware applies both mTLS and OAuth authentication
func (s *Server) combinedAuthMiddleware(next http.Handler) http.Handler {
	return s.mTLSMiddleware(s.oAuthMiddleware(next))
}
