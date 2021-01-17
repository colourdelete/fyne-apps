package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type App struct {
	ID, Name, Icon     string
	Developer, Summary string
	URL, Website       string
	Screenshots        []AppScreenshot

	Date    time.Time
	Version string

	Source   AppSource
	Requires string
}

type AppScreenshot struct {
	Image, Type string
}

type AppSource struct {
	Git, Package string
}

type AppList []App

func parseAppList(reader io.Reader) (AppList, error) {
	decode := json.NewDecoder(reader)

	appList := AppList{}
	err := decode.Decode(&appList)

	if err != nil {
		return nil, err
	}

	appList = appList.filterCompatible()
	sort.Slice(appList, func(a, b int) bool {
		return appList[a].Name < appList[b].Name
	})

	return appList, nil
}

func loadAppListFromWeb() (io.ReadCloser, error) {
	timeout := 1 * time.Second
	_, cancel := context.WithTimeout(context.Background(), timeout)

	defer cancel()

	req, err := http.NewRequest("GET", "https://apps.fyne.io/api/v1/list.json", nil)

	if err != nil {
		return nil, err
	}

	defer req.Body.Close()

	return req.Body, err
}

// TODO make actual cache read()
func loadAppListFromCache() (io.ReadCloser, error) {
	res, err := os.Open(filepath.Join("testdata", "list.json"))
	if err != nil {
		return nil, err
	}

	return res, nil
}
