package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"errors"

	"github.com/kyma-project/control-plane/tools/cli/pkg/ers"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"golang.org/x/oauth2/clientcredentials"
)

const timeoutInMilli = 3000

type HTTPClient struct {
	logger logger.Logger
	Client *http.Client
}

func NewHTTPClient(logger logger.Logger) (*HTTPClient, error) {

	// create a shared ERS HTTP client which does the oauth flow
	client, err := createConfigClient()
	if err != nil {
		return nil, fmt.Errorf("while create http client: %w", err)
	}

	return &HTTPClient{
		logger,
		client,
	}, nil
}

func (c *HTTPClient) put(url string) error {
	c.logger.Debugf("Executing PUT request: %s", url)
	return c.do(nil, func(ctx context.Context) (resp *http.Response, err error) {
		req, err := http.NewRequestWithContext(ctx, "PUT", url, nil)
		if err != nil {
			return nil, fmt.Errorf("Error while sending a PUT request: %w", err)
		}
		return c.Client.Do(req)
	})
}

func (c *HTTPClient) get(url string) ([]ers.Instance, error) {
	kymaEnv := make([]ers.Instance, 0)

	err := c.do(&kymaEnv, func(ctx context.Context) (resp *http.Response, err error) {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		return c.Client.Do(req)
	})

	return kymaEnv, err
}

func (c *HTTPClient) do(v interface{}, request func(ctx context.Context) (resp *http.Response, err error)) error {

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutInMilli)*time.Millisecond)
	defer cancel()

	resp, err := request(ctx)

	if err != nil {
		return fmt.Errorf("Error while sending request: %w", err)
	}

	// TODO: return error codes

	defer resp.Body.Close()

	d, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Error while reading from response: %w", err)
	}
	c.logger.Debugf("Received status code: %d", resp.StatusCode)
	c.logger.Debugf("Received raw response: %s", string(d))

	if v == nil {
		return nil
	}

	err = json.Unmarshal(d, v)
	if err != nil {
		return fmt.Errorf("Error while unmarshaling: %w", err)
	}

	return nil
}

func (c *HTTPClient) Close() {
	c.Client.CloseIdleConnections()
}

func createConfigClient() (*http.Client, error) {
	if ers.GlobalOpts.ClientID() == "" ||
		ers.GlobalOpts.ClientSecret() == "" ||
		ers.GlobalOpts.OauthUrl() == "" {
		return nil, errors.New("no auth data provided")
	}

	config := clientcredentials.Config{
		ClientID:     ers.GlobalOpts.ClientID(),
		ClientSecret: ers.GlobalOpts.ClientSecret(),
		TokenURL:     ers.GlobalOpts.OauthUrl(),
	}
	return config.Client(context.Background()), nil
}
