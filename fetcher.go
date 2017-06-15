package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type fetcher struct {
	URL string
}

func (f *fetcher) fetch(path string, reply interface{}) error {
	client := &http.Client{}

	req, err := http.NewRequest("GET", f.URL+path, nil)
	if nil != err {
		return err
	}

	resp, err := client.Do(req)
	if nil != err {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		return err
	}

	if err := json.Unmarshal(body, reply); err != nil {
		return err
	}

	return nil
}
