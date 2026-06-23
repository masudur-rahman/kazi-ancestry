package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/masudur-rahman/kazi-ancestry/configs"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const userinfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"

// GoogleConfig builds the OAuth2 config from the app's auth settings.
func GoogleConfig() *oauth2.Config {
	a := configs.KaziConfig.Auth
	return &oauth2.Config{
		ClientID:     a.GoogleClientID,
		ClientSecret: a.GoogleClientSecret,
		RedirectURL:  a.RedirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

// GoogleUser is the subset of the userinfo response we use.
type GoogleUser struct {
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
}

// FetchUser exchanges the auth code and returns the Google account profile.
func FetchUser(ctx context.Context, code string) (*GoogleUser, error) {
	conf := GoogleConfig()
	tok, err := conf.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}

	resp, err := conf.Client(ctx, tok).Get(userinfoURL)
	if err != nil {
		return nil, fmt.Errorf("fetch userinfo: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo status %d", resp.StatusCode)
	}

	var u GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, fmt.Errorf("decode userinfo: %w", err)
	}
	return &u, nil
}
