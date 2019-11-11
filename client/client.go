package client

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

// ErrInvalidDecodeResponseParameter invalid parameter for decodeResponse
var ErrInvalidDecodeResponseParameter = errors.New("nil interface provided to decodeResponse")

// Config for Client
type Config struct {
	Address          string
	TerraformVersion string
	Auth             Auth
}

// Auth for login on spinnaker
type Auth struct {
	Enabled     bool
	CertPath    string
	CertContent string
	KeyPath     string
	KeyContent  string
	UserEmail   string
}

// Client to talk to Spinnaker
type Client struct {
	Config Config
	client *http.Client
}

// NewClient Return a new client with loaded configuration
func NewClient(config Config) (*Client, error) {
	httpClient := http.DefaultClient
	if config.Auth.Enabled {
		var cert tls.Certificate
		var err error
		if config.Auth.CertContent != "" {
			cert, err = decodeBase64KeyPair(config.Auth.CertContent, config.Auth.KeyContent)
		} else {
			cert, err = tls.LoadX509KeyPair(config.Auth.CertPath, config.Auth.KeyPath)
		}
		if err != nil {
			return nil, err
		}

		tlsConfig := &tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: true,
		}
		tlsConfig.BuildNameToCertificate()
		transport := &http.Transport{TLSClientConfig: tlsConfig}
		httpClient = &http.Client{Transport: transport}
	}

	return &Client{
		Config: config,
		client: httpClient,
	}, nil
}

func decodeBase64KeyPair(cert64, key64 string) (tls.Certificate, error) {
	certBytes, err := base64.StdEncoding.DecodeString(cert64)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyBytes, err := base64.StdEncoding.DecodeString(key64)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.X509KeyPair(certBytes, keyBytes)
}

// NewRequest create http request
func (client *Client) NewRequest(method string, path string) (*http.Request, error) {
	return client.NewRequestWithBody(method, path, nil)
}

// NewRequestWithBody create http request with data as body
func (client *Client) NewRequestWithBody(method string, path string, data interface{}) (*http.Request, error) {
	reqURL, urlErr := url.Parse(client.Config.Address + path)
	if urlErr != nil {
		return nil, urlErr
	}

	jsonValue, jsonErr := json.Marshal(data)
	if jsonErr != nil {
		return nil, jsonErr
	}

	log.Printf("[INFO] Sending %s %s with body %s\n", method, reqURL, jsonValue)
	req, err := http.NewRequest(method, reqURL.String(), bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json;charset=UTF-8")
	return req, nil
}

// Do send http request
func (client *Client) Do(req *http.Request) (*http.Response, error) {
	resp, err := client.do(req)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()
	return resp, nil
}

// DoWithRetry send http request with retry
func (client *Client) DoWithRetry(createReq func() (*http.Request, error)) (*http.Response, error) {
	attempts := 0
	req, err := createReq()
	if err != nil {
		return nil, err
	}
	resp, respErr := client.Do(req)
	for respErr != nil && attempts < 5 {
		spinnakerError, ok := respErr.(*SpinnakerError)
		if !ok {
			return nil, respErr
		}
		if spinnakerError.Status != 400 {
			return nil, spinnakerError
		}
		time.Sleep(time.Duration(attempts*attempts) * time.Second)

		req, err := createReq()
		if err != nil {
			return nil, err
		}
		log.Printf("[INFO] retry attempt %v for request %v\n", attempts+2, req)
		resp, respErr = client.Do(req)
		attempts++
	}
	return resp, respErr
}

// DoWithResponse send http request and parse response body
func (client *Client) DoWithResponse(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := client.do(req)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()

	err = decodeResponse(resp, v)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// do internal function used by Do and DoWithResponse to validate response
func (client *Client) do(req *http.Request) (*http.Response, error) {
	resp, err := client.client.Do(req)
	if err != nil {
		return resp, err
	}

	err = validateResponse(resp)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

func decodeResponse(r *http.Response, v interface{}) error {
	if v == nil {
		return ErrInvalidDecodeResponseParameter
	}

	bodyBytes, _ := ioutil.ReadAll(r.Body)
	bodyString := string(bodyBytes)
	log.Println("[DEBUG] Got response body", bodyString)

	return json.Unmarshal([]byte(bodyString), &v)
}

func validateResponse(r *http.Response) error {
	if c := r.StatusCode; 200 <= c && c <= 299 {
		return nil
	}

	bodyBytes, _ := ioutil.ReadAll(r.Body)
	bodyString := string(bodyBytes)
	log.Println("[INFO] Error response body", bodyString)

	spinnakerError := SpinnakerError{}
	err := json.Unmarshal([]byte(bodyString), &spinnakerError)
	if err != nil {
		return err
	}

	return &spinnakerError
}
