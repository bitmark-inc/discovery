// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed http request: status code = %d; url = %s", resp.StatusCode, f.URL+path)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		return err
	}

	if err := json.Unmarshal(body, reply); err != nil {
		return err
	}

	return nil
}
