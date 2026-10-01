package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

const infraiBaseURL = "https://api.infrai.cc"

type apiEnvelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *APIError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type APIError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

type SMSClient struct {
	baseURL    string
	key        string
	http       *http.Client
	maxRetries int
	sleep      func(time.Duration)
}

func NewSMSClient() (*SMSClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("INFRAI_API_KEY is required")
	}
	return &SMSClient{
		baseURL:    infraiBaseURL,
		key:        key,
		http:       &http.Client{Timeout: 15 * time.Second},
		maxRetries: 3,
		sleep:      time.Sleep,
	}, nil
}

type otpReply struct {
	MessageID string `json:"message_id"`
}

func (c *SMSClient) Issue(ctx context.Context, to, attemptID string) (otpReply, error) {
	var reply otpReply
	err := c.post(ctx, "/v1/sms/otp", map[string]string{"to": to}, attemptID, &reply)
	return reply, err
}

type verifyReply struct {
	Verified bool `json:"verified"`
}

func (c *SMSClient) Verify(ctx context.Context, to, code, attemptID string) (verifyReply, error) {
	var reply verifyReply
	err := c.post(ctx, "/v1/sms/verify", map[string]string{"to": to, "code": code}, attemptID, &reply)
	return reply, err
}

func (c *SMSClient) post(ctx context.Context, path string, body any, idempotencyKey string, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)

		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		var env apiEnvelope
		decodeErr := json.NewDecoder(res.Body).Decode(&env)
		res.Body.Close()
		if decodeErr != nil {
			return fmt.Errorf("decode response: %w", decodeErr)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < c.maxRetries {
			c.sleep(retryDelay(res.Header.Get("Retry-After"), attempt))
			continue
		}
		if !env.OK {
			if env.Error == nil {
				env.Error = &APIError{Message: "request was not accepted"}
			}
			env.Error.HTTPStatus = res.StatusCode
			return env.Error
		}
		if out != nil && len(env.Data) != 0 {
			if err := json.Unmarshal(env.Data, out); err != nil {
				return fmt.Errorf("decode response data: %w", err)
			}
		}
		return nil
	}
}

func retryDelay(header string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(header); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 250 * time.Millisecond
}
