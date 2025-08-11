package httpclients

import (
	"resty.dev/v3"
)

func Init() {
	RestyClient = resty.New()
	RestyClient.OnSuccess(func(c *resty.Client, r *resty.Response) {
		// TODO: logging?
	})
}

var RestyClient *resty.Client
