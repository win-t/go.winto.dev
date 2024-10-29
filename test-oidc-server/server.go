package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"
	"go.winto.dev/errors"
	"go.winto.dev/httphandler"
	"go.winto.dev/httphandler/defresponse"
)

type server struct {
	http.Handler

	issuer    string
	jwk       jose.JSONWebKey
	signer    jose.Signer
	adminPass string
	template  *template.Template
	log       *log.Logger
}

func newServer(issuer string, key jose.JSONWebKey, adminPass string, template *template.Template, log *log.Logger) *server {
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.SignatureAlgorithm(key.Algorithm), Key: key},
		(&jose.SignerOptions{}).WithType("JWT"),
	)
	errors.Check(err)

	s := server{
		issuer:    strings.TrimSuffix(issuer, "/"),
		jwk:       key.Public(),
		signer:    signer,
		adminPass: adminPass,
		template:  template,
		log:       log,
	}
	s.setupHandler()

	return &s
}

func (s *server) setupHandler() {
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", httphandler.Of(s.handleDiscovery))
	mux.HandleFunc("/jwks", httphandler.Of(s.handleJWKS))
	mux.HandleFunc("/token", httphandler.Of(s.handleToken))

	mux.HandleFunc("/authorize", httphandler.Chain(
		func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				user, pass, ok := r.BasicAuth()
				if !ok || user != "admin" || pass != s.adminPass {
					w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				next(w, r)
			}
		},
		s.handleAuthorize,
	))

	s.Handler = httphandler.Chain(
		func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				if err := errors.Catch0(func() { next(w, r) }); err != nil {
					s.log.Print(errors.Format(err))
					panic(http.ErrAbortHandler)
				}
			}
		},
		mux,
	)
}

func (s *server) handleDiscovery(r *http.Request) http.HandlerFunc {
	return defresponse.JSON(http.StatusOK, struct {
		Issuer                           string   `json:"issuer"`
		JwksURI                          string   `json:"jwks_uri"`
		AuthorizationEndpoint            string   `json:"authorization_endpoint"`
		TokenEndpoint                    string   `json:"token_endpoint"`
		ResponseTypesSupported           []string `json:"response_types_supported"`
		SubjectTypesSupported            []string `json:"subject_types_supported"`
		IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
	}{
		Issuer:                           s.issuer,
		JwksURI:                          s.issuer + "/jwks",
		AuthorizationEndpoint:            s.issuer + "/authorize",
		TokenEndpoint:                    s.issuer + "/token",
		ResponseTypesSupported:           []string{"code", "id_token"},
		SubjectTypesSupported:            []string{"public"},
		IDTokenSigningAlgValuesSupported: []string{s.jwk.Algorithm},
	})
}

func (s *server) handleJWKS(r *http.Request) http.HandlerFunc {
	return defresponse.JSON(http.StatusOK, jose.JSONWebKeySet{Keys: []jose.JSONWebKey{s.jwk}})
}

func (s *server) handleToken(r *http.Request) http.HandlerFunc {
	token := r.FormValue("code")
	if token == "" {
		return defresponse.Text(http.StatusBadRequest, "missing code")
	}
	jws, err := jose.ParseSignedCompact(token, []jose.SignatureAlgorithm{jose.SignatureAlgorithm(s.jwk.Algorithm)})
	if err != nil {
		return defresponse.Text(http.StatusBadRequest, "cannot parse code")
	}
	payload, err := jws.Verify(s.jwk)
	if err != nil {
		return defresponse.Text(http.StatusBadRequest, "cannot verify code")
	}
	var expData struct {
		Exp int64 `json:"exp"`
	}
	err = json.Unmarshal(payload, &expData)
	errors.Check(err)

	expIn := expData.Exp - time.Now().Unix()
	if expIn <= 0 {
		return defresponse.Text(http.StatusBadRequest, "code expired")
	}

	return defresponse.JSON(http.StatusOK, struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int64  `json:"expires_in"`
		IDToken     string `json:"id_token"`
	}{
		AccessToken: "acess_token_not_available",
		TokenType:   "Bearer",
		ExpiresIn:   expIn,
		IDToken:     token,
	})
}

func (s *server) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	alert := ""

	sub := "test-user@example.com"
	ttl := "300"
	claims := ""

	switch r.Method {
	case http.MethodGet:
		now := time.Now().Unix()

		claimsBytes, err := json.MarshalIndent(struct {
			Iss      string `json:"iss"`
			Aud      string `json:"aud"`
			Iat      int64  `json:"iat"`
			AuthTime int64  `json:"auth_time"`
			Nonce    string `json:"nonce,omitempty"`
		}{
			Iss:      s.issuer,
			Aud:      r.FormValue("client_id"),
			Iat:      now,
			AuthTime: now,
			Nonce:    r.FormValue("nonce"),
		}, "", "  ")
		errors.Check(err)
		claims = string(claimsBytes)

	case http.MethodPost:
		sub = r.FormValue("sub")
		ttl = r.FormValue("ttl")
		claims = r.FormValue("claims")

		ttlInt, err := strconv.ParseInt(ttl, 10, 64)
		if err != nil {
			alert = "Invalid TTL: " + err.Error()
			break
		}

		var claimsData map[string]any
		err = json.Unmarshal([]byte(claims), &claimsData)
		if err != nil {
			alert = "Invalid Claims: " + err.Error()
			break
		}

		if _, ok := claimsData["sub"]; !ok {
			claimsData["sub"] = sub
		}
		if _, ok := claimsData["exp"]; !ok {
			iat, _ := claimsData["iat"].(float64)
			if iat == 0 {
				alert = "Invalid Claims: iat is not valid"
				break
			}
			claimsData["exp"] = int64(iat) + ttlInt
		}

		claimsByes, err := json.Marshal(claimsData)
		errors.Check(err)

		jws, err := s.signer.Sign(claimsByes)
		errors.Check(err)

		token, err := jws.CompactSerialize()
		errors.Check(err)

		redirectURIStr := r.FormValue("redirect_uri")
		if redirectURIStr == "" {
			defresponse.Text(http.StatusOK, token)(w, r)
			return
		}

		redirectURI, err := url.Parse(redirectURIStr)
		if err != nil {
			alert = "Invalid redirect_uri: " + err.Error()
			break
		}
		q := redirectURI.Query()
		if r.FormValue("response_type") == "id_token" {
			q.Set("id_token", token)
		} else { // assume "response_type" == "code"
			q.Set("code", token)
		}
		if r.FormValue("state") != "" {
			q.Set("state", r.FormValue("state"))
		}
		redirectURI.RawQuery = q.Encode()

		http.Redirect(w, r, redirectURI.String(), http.StatusFound)
		return

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	err := s.template.ExecuteTemplate(w, "authorize.html", struct {
		Alert  string
		Sub    string
		TTL    string
		Claims string
	}{
		Alert:  alert,
		Sub:    sub,
		TTL:    ttl,
		Claims: claims,
	})
	errors.Check(err)
}
