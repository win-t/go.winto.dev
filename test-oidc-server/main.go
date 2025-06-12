package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"embed"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/go-jose/go-jose/v4"
	"go.winto.dev/errors"
)

//go:embed authorize.html
var assets embed.FS

func init() { errors.SetFormatFilterPkgs("main", "go.winto.dev/test-oidc-server") }

func main() {
	if err := errors.Catch0(run); err != nil {
		fmt.Fprint(os.Stderr, errors.Format(err))
		os.Exit(1)
	}
}

func run() {
	iss := os.Getenv("ISSUER")
	errors.Expect(iss != "", "ISSUER is required")

	key := os.Getenv("KEYFILE")
	if key == "" {
		key = os.Getenv("KEY")
	}
	errors.Expect(key != "", "KEYFILE is required")

	adminPass := os.Getenv("ADMIN_PASS")
	errors.Expect(adminPass != "", "ADMIN_PASS is required")

	_, err := os.Stat(key)
	if os.IsNotExist(err) {
		var privateKey *rsa.PrivateKey
		privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
		errors.Check(err)

		err = os.WriteFile(
			key,
			pem.EncodeToMemory(&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
			}),
			0o600,
		)
	}
	errors.Check(err)

	keyBytes, err := os.ReadFile(key)
	errors.Check(err)

	keyPem, _ := pem.Decode(keyBytes)
	errors.Expect(keyPem != nil && keyPem.Type == "RSA PRIVATE KEY", "invalid key file")

	privRsa, err := x509.ParsePKCS1PrivateKey(keyPem.Bytes)
	errors.Check(err)

	sumPubRsa := sha256.Sum224(x509.MarshalPKCS1PublicKey(&privRsa.PublicKey))
	keyID := base64.RawURLEncoding.EncodeToString(sumPubRsa[:])

	privJwk := jose.JSONWebKey{
		Key:       privRsa,
		KeyID:     keyID,
		Use:       "sig",
		Algorithm: string(jose.RS256),
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Println("Running on port", port)

	template, err := template.ParseFS(assets, "*")
	errors.Check(err)

	tlsCert := os.Getenv("TLS_CERT")
	tlsKey := os.Getenv("TLS_KEY")
	server := &http.Server{
		Addr:     ":" + port,
		Handler:  newServer(iss, privJwk, adminPass, template, log.New(os.Stderr, "server: ", log.LstdFlags)),
		ErrorLog: log.New(os.Stderr, "http: ", log.LstdFlags),
	}
	if tlsCert == "" || tlsKey == "" {
		err = server.ListenAndServe()
	} else {
		err = server.ListenAndServeTLS(tlsCert, tlsKey)
	}
	errors.Check(err)
}
