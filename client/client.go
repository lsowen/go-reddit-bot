package client

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	ClientId                  string
	clientSecret              string
	AccessToken               string
	AccessTokenExpirationTime time.Time
	Username                  string
	password                  string

	httpClient *http.Client
	UserAgent  string

	lastRequestTime time.Time
}

func New(clientId string, clientSecret string) *Client {
	c := &Client{
		ClientId:     clientId,
		clientSecret: clientSecret,
		UserAgent:    "golang:maybemaybemaybe_bot:v0.0.1 (by /u/ninja_haiku)",
		httpClient:   &http.Client{},
	}

	return c
}

func (c *Client) doRequest(request *http.Request) (response *http.Response, err error) {
	request.Header.Set("User-Agent", c.UserAgent)

	waitLength := time.Duration(2) * time.Second
	elapsedTime := time.Now().Sub(c.lastRequestTime)
	if elapsedTime < waitLength {
		time.Sleep(waitLength - elapsedTime)
	}

	response, err = c.httpClient.Do(request)
	c.lastRequestTime = time.Now()
	return
}

func (c *Client) Signin(username string, password string) {
	c.Username = username
	c.password = password
}

func (c *Client) authorize() error {
	if c.AccessToken != "" && time.Now().Before(c.AccessTokenExpirationTime) {
		return nil
	}

	form := url.Values{
		"grant_type": {"password"},
		"username":   {c.Username},
		"password":   {c.password},
	}

	endpointUrl := "https://www.reddit.com/api/v1/access_token"
	req, err := http.NewRequest("POST", endpointUrl, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.ClientId, c.clientSecret)

	resp, err := c.doRequest(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	type TokenStruct struct {
		Scope       string
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	token_struct := TokenStruct{}
	decoder := json.NewDecoder(resp.Body)

	err = decoder.Decode(&token_struct)
	if err != nil {
		return err
	}

	c.AccessToken = token_struct.AccessToken
	c.AccessTokenExpirationTime = time.Now().Add(time.Duration(token_struct.ExpiresIn) * time.Second)
	return nil
}

func (c *Client) Get(resourceUrl string) (*http.Response, error) {
	err := c.authorize()
	if err != nil {
		return nil, err
	}

	endpointUrl := "https://oauth.reddit.com" + resourceUrl
	req, err := http.NewRequest("GET", endpointUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Authorization", "bearer "+c.AccessToken)

	return c.doRequest(req)
}

func (c *Client) Post(resourceUrl string, values url.Values) (*http.Response, error) {
	err := c.authorize()
	if err != nil {
		return nil, err
	}

	endpointUrl := "https://oauth.reddit.com" + resourceUrl
	req, err := http.NewRequest("POST", endpointUrl, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Authorization", "bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return c.doRequest(req)
}

func (c *Client) SubmitComment(parentId string, text string) error {
	values := url.Values{
		"api_type": {"json"},
		"text":     {text},
		"thing_id": {parentId},
	}

	response, err := c.Post("/api/comment.json", values)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	/* TODO: Parse response and actually make sure submission was successful */

	return nil
}
