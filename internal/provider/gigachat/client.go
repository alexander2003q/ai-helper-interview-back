package gigachat

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const maxResponseBodySize = 2 << 20

type Config struct {
	AuthorizationKey string
	Model            string
	Scope            string
	OAuthURL         string
	ChatURL          string
	HTTPClient       *http.Client
	MaxRetries       int
}

type Client struct {
	authorizationKey string
	model            string
	scope            string
	oAuthURL         string
	chatURL          string
	httpClient       *http.Client
	maxRetries       int
	now              func() time.Time

	tokenMutex     sync.Mutex
	accessToken    string
	tokenExpiresAt time.Time
}

func NewClient(config Config) (*Client, error) {
	if strings.TrimSpace(config.AuthorizationKey) == "" {
		return nil, errors.New("GIGACHAT_AUTHORIZATION_KEY is required when AI_PROVIDER=gigachat")
	}
	if strings.TrimSpace(config.Model) == "" {
		return nil, errors.New("GIGACHAT_MODEL must not be empty")
	}
	if strings.TrimSpace(config.Scope) == "" {
		return nil, errors.New("GIGACHAT_SCOPE must not be empty")
	}
	if err := validateURL("GIGACHAT_OAUTH_URL", config.OAuthURL); err != nil {
		return nil, err
	}
	if err := validateURL("GIGACHAT_CHAT_URL", config.ChatURL); err != nil {
		return nil, err
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}

	return &Client{
		authorizationKey: config.AuthorizationKey,
		model:            config.Model,
		scope:            config.Scope,
		oAuthURL:         config.OAuthURL,
		chatURL:          config.ChatURL,
		httpClient:       httpClient,
		maxRetries:       max(config.MaxRetries, 0),
		now:              time.Now,
	}, nil
}

func validateURL(name, value string) error {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Scheme != "https" && parsed.Scheme != "http" || parsed.Host == "" {
		return fmt.Errorf("invalid %s", name)
	}
	return nil
}

func (c *Client) getAccessToken(ctx context.Context, forceRefresh bool) (string, error) {
	c.tokenMutex.Lock()
	defer c.tokenMutex.Unlock()

	if !forceRefresh && c.accessToken != "" && c.now().Add(30*time.Second).Before(c.tokenExpiresAt) {
		return c.accessToken, nil
	}

	form := url.Values{"scope": []string{c.scope}}
	var responseBody []byte
	var err error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		request, requestErr := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			c.oAuthURL,
			strings.NewReader(form.Encode()),
		)
		if requestErr != nil {
			return "", fmt.Errorf("create GigaChat OAuth request: %w", requestErr)
		}
		request.Header.Set("Accept", "application/json")
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.Header.Set("Authorization", "Basic "+c.authorizationKey)
		request.Header.Set("RqUID", newUUID())

		responseBody, err = c.send(request)
		if err == nil {
			break
		}
		if !isRetryable(err) || attempt == c.maxRetries {
			return "", fmt.Errorf("request GigaChat access token: %w", err)
		}
		if err := waitForRetry(ctx, attempt); err != nil {
			return "", err
		}
	}

	var response struct {
		AccessToken string `json:"access_token"`
		ExpiresAt   int64  `json:"expires_at"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", fmt.Errorf("decode GigaChat OAuth response: %w", err)
	}
	if response.AccessToken == "" || response.ExpiresAt <= 0 {
		return "", errors.New("GigaChat OAuth response contains no usable access token")
	}

	expiresAt := time.Unix(response.ExpiresAt, 0)
	if response.ExpiresAt > 1_000_000_000_000 {
		expiresAt = time.UnixMilli(response.ExpiresAt)
	}
	c.accessToken = response.AccessToken
	c.tokenExpiresAt = expiresAt

	return c.accessToken, nil
}

func (c *Client) send(request *http.Request) ([]byte, error) {
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("send GigaChat request: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBodySize))
	if err != nil {
		return nil, fmt.Errorf("read GigaChat response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, &providerError{
			status:  response.StatusCode,
			message: providerErrorMessage(body),
		}
	}

	return body, nil
}

type providerError struct {
	status  int
	message string
}

func (e *providerError) Error() string {
	return fmt.Sprintf("GigaChat returned HTTP %d: %s", e.status, e.message)
}

func providerErrorMessage(body []byte) string {
	var response struct {
		Message string `json:"message"`
		Error   struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &response); err == nil {
		if response.Error.Message != "" {
			return response.Error.Message
		}
		if response.Message != "" {
			return response.Message
		}
	}
	return "provider request failed"
}

func isRetryable(err error) bool {
	var providerErr *providerError
	return errors.As(err, &providerErr) && (providerErr.status == http.StatusTooManyRequests || providerErr.status >= http.StatusInternalServerError)
}

func isUnauthorized(err error) bool {
	var providerErr *providerError
	return errors.As(err, &providerErr) && providerErr.status == http.StatusUnauthorized
}

func waitForRetry(ctx context.Context, attempt int) error {
	timer := time.NewTimer(time.Duration(250*(1<<attempt)) * time.Millisecond)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func newUUID() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	value[6] = value[6]&0x0f | 0x40
	value[8] = value[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", value[0:4], value[4:6], value[6:8], value[8:10], value[10:])
}
