package registryauth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/thin-edge/tedge-oscar/internal/config"
)

var debugHTTP bool

// SetDebugHTTP sets whether HTTP debug output is enabled for registry communication.
func SetDebugHTTP(level string) {
	debugHTTP = (level == "debug")
}

type AuthResult struct {
	Client   *http.Client
	Username string
	Password string
	Token    string
}

type roundTripperWithBasicAuth struct {
	base     http.RoundTripper
	username string
	password string
}

func (rt roundTripperWithBasicAuth) RoundTrip(req *http.Request) (*http.Response, error) {
	if rt.username != "" {
		auth := rt.username + ":" + rt.password
		header := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
		req.Header.Set("Authorization", header)
	}
	if debugHTTP {
		printHTTPRequest(req)
	}
	return rt.base.RoundTrip(req)
}

type roundTripperWithBearerToken struct {
	base  http.RoundTripper
	token string
}

func (rt roundTripperWithBearerToken) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+rt.token)
	if debugHTTP {
		printHTTPRequest(req)
	}
	return rt.base.RoundTrip(req)
}

func printHTTPRequest(req *http.Request) {
	fmt.Fprintln(os.Stderr, "--- HTTP Request ---")
	fmt.Fprintf(os.Stderr, "%s %s\n", req.Method, req.URL.String())
	for k, v := range req.Header {
		for _, vv := range v {
			fmt.Fprintf(os.Stderr, "%s: %s\n", k, vv)
		}
	}
	fmt.Fprintln(os.Stderr, "-------------------")
}

// GetAuthenticatedClient returns an http.Client with the appropriate auth for the registry, or nil if no auth is needed.
// If the registry is ghcr.io and username/password are present, it will fetch a Bearer token for the given scope.
func GetAuthenticatedClient(cfg *config.Config, repoRef, scope string) (*http.Client, string, string, string, error) {
	u, err := url.Parse("https://" + repoRef)
	if err != nil {
		return nil, "", "", "", fmt.Errorf("invalid repoRef: %w", err)
	}
	reg := u.Host
	var username, password string
	if cred := cfg.FindCredential(reg); cred != nil {
		username = cred.Username
		password = cred.Password
	}
	token := ""
	if reg == "ghcr.io" && username != "" && password != "" {
		if scope == "" {
			ownerRepo := strings.TrimPrefix(repoRef, "ghcr.io/")
			scope = "repository:" + ownerRepo + ":pull"
		}
		tokenURL := "https://ghcr.io/token?service=ghcr.io&scope=" + url.QueryEscape(scope)
		req, err := http.NewRequest("GET", tokenURL, nil)
		if err == nil {
			req.SetBasicAuth(username, password)
			resp, err := http.DefaultClient.Do(req)
			if err == nil && resp.StatusCode == 200 {
				defer resp.Body.Close()
				type tokenResp struct {
					Token string `json:"token"`
				}
				var tr tokenResp
				if err := json.NewDecoder(resp.Body).Decode(&tr); err == nil && tr.Token != "" {
					token = tr.Token
				}
			}
		}
	}
	if token != "" {
		base := http.DefaultTransport
		return &http.Client{
			Transport: roundTripperWithBearerToken{base, token},
		}, username, password, token, nil
	} else if username != "" && password != "" {
		base := http.DefaultTransport
		return &http.Client{
			Transport: roundTripperWithBasicAuth{base, username, password},
		}, username, password, "", nil
	}
	return nil, "", "", "", nil
}
