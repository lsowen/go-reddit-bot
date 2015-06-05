package client

import (
	"fmt"
	"io/ioutil"
	"net/url"
)

func (c *Client) NeedsCaptcha() (bool, error) {

	response, err := c.Get("/api/needs_captcha")
	if err != nil {
		return false, err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return false, err
	}
	needsCaptcha := (string(body) == "true")

	return needsCaptcha, nil
}

func (c *Client) NewCaptcha() (string, error) {
	values := url.Values{
		"api_type": {"json"},
	}
	response, err := c.Post("/api/new_captcha", values)

	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	fmt.Println(string(body))
	return "", nil
}
