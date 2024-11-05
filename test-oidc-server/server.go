package main

import (
	"context"
	"encoding/json"
	"fmt"
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
	"go.winto.dev/httphandler/defmiddleware"
	"go.winto.dev/httphandler/defresponse"
	"go.winto.dev/typedcontext"
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

type reqContext struct{ claims map[string]any }

func newServer(issuer string, key jose.JSONWebKey, adminPass string, template *template.Template, log *log.Logger) *server {
	var err error

	s := server{
		issuer:    strings.TrimSuffix(issuer, "/"),
		jwk:       key.Public(),
		adminPass: adminPass,
		template:  template,
		log:       log,
	}

	s.signer, err = jose.NewSigner(
		jose.SigningKey{Algorithm: jose.SignatureAlgorithm(key.Algorithm), Key: key},
		(&jose.SignerOptions{}).WithType("JWT"),
	)
	errors.Check(err)

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", s.handleDiscovery())
	mux.HandleFunc("/jwks", s.handleJWKS())
	mux.HandleFunc("/userinfo", httphandler.Chain(
		defmiddleware.BearerHeaderAuth(s.verifyTokenAuth),
		s.handleUserInfo,
	))
	mux.HandleFunc("/token", httphandler.Of(s.handleToken))
	mux.HandleFunc("/authorize", httphandler.Chain(
		defmiddleware.BasicAuth(s.verifyAdminAuth),
		s.handleAuthorize,
	))

	s.Handler = httphandler.Chain(
		s.commonMiddleware,
		mux,
	)

	return &s
}

func (s *server) commonMiddleware(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	// Initialize reqContext
	r = r.WithContext(typedcontext.New(r.Context(), &reqContext{}))

	// Recover from panics and log the error
	if err := errors.Catch0(func() { next(w, r) }); err != nil {
		s.log.Print(errors.Format(err))
		panic(http.ErrAbortHandler)
	}
}

func (s *server) verifyAdminAuth(ctx context.Context, user, pass string) (bool, string) {
	return user == "admin" && pass == s.adminPass, ""
}

func (s *server) verifyTokenAuth(ctx context.Context, token string) (bool, string) {
	payload, _, err := s.verifyJWT(token)
	if err != nil {
		return false, err.Error()
	}

	var claims map[string]any
	err = json.Unmarshal(payload, &claims)
	errors.Check(err)

	delete(claims, "iss")
	delete(claims, "aud")
	delete(claims, "iat")
	delete(claims, "auth_time")
	delete(claims, "nonce")
	delete(claims, "exp")

	typedcontext.MustGet[*reqContext](ctx).claims = claims

	return true, ""
}

func (s *server) verifyJWT(jwt string) (payload []byte, exp int64, err error) {
	jws, err := jose.ParseSignedCompact(jwt, []jose.SignatureAlgorithm{jose.SignatureAlgorithm(s.jwk.Algorithm)})
	if err != nil {
		return nil, 0, fmt.Errorf("cannot parse jwt")
	}
	payload, err = jws.Verify(s.jwk)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot verify jwt")
	}
	var expData struct {
		Exp int64 `json:"exp"`
	}
	err = json.Unmarshal(payload, &expData)
	errors.Check(err)

	if expData.Exp <= time.Now().Unix() {
		return nil, expData.Exp, fmt.Errorf("jwt expired")
	}
	return payload, expData.Exp, nil
}

func (s *server) handleDiscovery() http.HandlerFunc {
	resp, err := json.Marshal(struct {
		Issuer                           string   `json:"issuer"`
		JwksURI                          string   `json:"jwks_uri"`
		UserInfoEndpoint                 string   `json:"userinfo_endpoint"`
		TokenEndpoint                    string   `json:"token_endpoint"`
		AuthorizationEndpoint            string   `json:"authorization_endpoint"`
		ResponseTypesSupported           []string `json:"response_types_supported"`
		SubjectTypesSupported            []string `json:"subject_types_supported"`
		IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
	}{
		Issuer:                           s.issuer,
		JwksURI:                          s.issuer + "/jwks",
		AuthorizationEndpoint:            s.issuer + "/authorize",
		TokenEndpoint:                    s.issuer + "/token",
		UserInfoEndpoint:                 s.issuer + "/userinfo",
		ResponseTypesSupported:           []string{"code", "id_token"},
		SubjectTypesSupported:            []string{"public"},
		IDTokenSigningAlgValuesSupported: []string{s.jwk.Algorithm},
	})
	errors.Check(err)

	return defresponse.Data(http.StatusOK, "application/json", resp)
}

func (s *server) handleJWKS() http.HandlerFunc {
	resp, err := json.Marshal(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{s.jwk}})
	errors.Check(err)

	return defresponse.Data(http.StatusOK, "application/json", resp)
}

func (s *server) handleUserInfo(r *http.Request) http.HandlerFunc {
	claims := typedcontext.MustGet[*reqContext](r.Context()).claims
	return defresponse.JSON(http.StatusOK, claims)
}

func (s *server) handleToken(r *http.Request) http.HandlerFunc {
	token := r.FormValue("code")
	_, exp, err := s.verifyJWT(token)
	if err != nil {
		return defresponse.Text(http.StatusBadRequest, err.Error())
	}

	return defresponse.JSON(http.StatusOK, struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int64  `json:"expires_in"`
		IDToken     string `json:"id_token"`
	}{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   exp - time.Now().Unix(),
		IDToken:     token,
	})
}

func (s *server) handleAuthorize(r *http.Request) http.HandlerFunc {
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

		claimsBytes, err := json.Marshal(claimsData)
		errors.Check(err)

		jws, err := s.signer.Sign(claimsBytes)
		errors.Check(err)

		token, err := jws.CompactSerialize()
		errors.Check(err)

		redirectURIStr := r.FormValue("redirect_uri")
		if redirectURIStr == "" {
			return defresponse.Text(http.StatusOK, token)
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

		return defresponse.Redirect(http.StatusFound, redirectURI.String())

	default:
		return defresponse.Error(http.StatusMethodNotAllowed, "Method Not Allowed")
	}

	return defresponse.HTMLTemplate(http.StatusOK, s.template, "authorize.html", struct {
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
}
