package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
)

type linkResponseError struct {
	Type string
}

func (l *linkResponseError) UnmarshalJSON(data []byte) error {
	var elements []string
	err := json.Unmarshal(data, &elements)
	if err != nil {
		return errors.New("Could not unmarshal error")
	}

	l.Type = elements[0]

	return nil
}

type linkResponseData struct {
	Url  string
	Id   string
	Name string
}

type linkResponse struct {
	Captcha string
	Errors  []linkResponseError
	Data    *linkResponseData
}

type BadCaptchaError struct {
	CaptchaId string
}

func (e BadCaptchaError) Error() string {
	return fmt.Sprintf("Captcha Required (id %s)", e.CaptchaId)
}

type SubmitLinkParameters struct {
	Subreddit       string
	Title           string
	Url             string
	CaptchaId       string
	CaptchaResponse string
	Nsfw            bool
}

func (c *Client) SubmitLink(parameters SubmitLinkParameters) (*linkResponseData, error) {
	values := url.Values{
		"api_type":  {"json"},
		"kind":      {"link"},
		"extension": {"json"},
		"sr":        {parameters.Subreddit},
		"title":     {parameters.Title},
		"url":       {parameters.Url},
	}

	if parameters.CaptchaId != "" && parameters.CaptchaResponse != "" {
		values.Add("iden", parameters.CaptchaId)
		values.Add("captcha", parameters.CaptchaResponse)
	}

	response, err := c.Post("/api/submit.json", values)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	type ResponseMsg struct {
		Json linkResponse
	}

	response_struct := ResponseMsg{}

	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&response_struct)
	if err != nil {
		return nil, err
	}

	for _, e := range response_struct.Json.Errors {
		if e.Type == "BAD_CAPTCHA" {
			return nil, BadCaptchaError{
				CaptchaId: response_struct.Json.Captcha,
			}
		}
	}

	return response_struct.Json.Data, nil
}
