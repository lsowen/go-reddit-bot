package client

import (
	"encoding/json"
)

type Listing struct {
	Kind string
	Data struct {
		Domain    string
		Subreddit string
		Id        string
		Author    string
		Permalink string
		Title     string
		Url       string
		Score     int
		Over18    bool `json:"over_18"`
		IsSelf    bool `json:"is_self"`
	}
}

func (l *Listing) Copy() *Listing {
	c := &Listing{
		Kind: l.Kind,
		Data: struct {
			Domain    string
			Subreddit string
			Id        string
			Author    string
			Permalink string
			Title     string
			Url       string
			Score     int
			Over18    bool `json:"over_18"`
			IsSelf    bool `json:"is_self"`
		}{
			Domain:    l.Data.Domain,
			Subreddit: l.Data.Subreddit,
			Id:        l.Data.Id,
			Author:    l.Data.Author,
			Permalink: l.Data.Permalink,
			Title:     l.Data.Title,
			Url:       l.Data.Url,
			Score:     l.Data.Score,
			Over18:    l.Data.Over18,
			IsSelf:    l.Data.IsSelf,
		},
	}

	return c
}

type Response struct {
	Type string
	Data struct {
		Children []Listing
	}
}

func (c *Client) GetSubreddit(resourceUrl string) (*Response, error) {
	response, err := c.Get(resourceUrl)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	response_struct := &Response{}

	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&response_struct)
	if err != nil {
		return nil, err
	}

	return response_struct, nil

}

func ParseResponse(response []byte) (*Response, error) {
	r := &Response{}

	err := json.Unmarshal(response, r)
	return r, err
}
