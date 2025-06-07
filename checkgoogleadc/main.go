package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"go.winto.dev/errors"
	"go.winto.dev/mainpkg"
	"golang.org/x/oauth2/google"
)

func main() { mainpkg.Exec(run) }

func run(ctx context.Context) {
	cred, err := google.FindDefaultCredentials(ctx, "email")
	errors.Check(err)

	token, err := cred.TokenSource.Token()
	errors.Check(err)

	req, err := http.NewRequestWithContext(ctx,
		"GET",
		fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?access_token=%s", url.QueryEscape(token.AccessToken)),
		nil,
	)
	errors.Check(err)

	resp, err := http.DefaultClient.Do(req)
	errors.Check(err)

	io.Copy(os.Stdout, resp.Body)
}
