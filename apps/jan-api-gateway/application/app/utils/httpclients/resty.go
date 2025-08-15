package httpclients

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"menlo.ai/jan-api-gateway/app/utils/contextkeys"
	"menlo.ai/jan-api-gateway/app/utils/logger"
	"resty.dev/v3"
)

func NewClient(clientName string) *resty.Client {
	client := resty.New()
	client.AddRequestMiddleware(func(c *resty.Client, r *resty.Request) error {
		start := time.Now()
		ctx := context.WithValue(r.Context(), contextkeys.HttpClientStartsAt{}, start)
		ctx = context.WithValue(ctx, contextkeys.HttpClientRequestBody{}, r.Body)
		r.SetContext(ctx)
		return nil
	})
	client.AddResponseMiddleware(func(c *resty.Client, r *resty.Response) error {
		logger := logger.GetLogger()
		requestID := r.Request.Context().Value(contextkeys.RequestId{})
		startTime, _ := r.Request.Context().Value(contextkeys.HttpClientStartsAt{}).(time.Time)
		requestBody := r.Request.Context().Value(contextkeys.HttpClientRequestBody{})
		latency := time.Since(startTime)
		var responseBody any
		if !r.Request.DoNotParseResponse {
			responseBody = r.Result()
		}
		logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"client":     clientName,
			"status":     r.StatusCode(),
			"method":     r.Request.RawRequest.Method,
			"path":       r.Request.RawRequest.URL.Path,
			"query":      r.Request.RawRequest.URL.RawQuery,
			"headers":    r.Request.RawRequest.Header,
			"req_body":   requestBody,
			"resp_body":  responseBody,
			"latency":    latency.String(),
			"client_ip":  nil,
		}).Info("")
		return nil
	})
	return client
}
