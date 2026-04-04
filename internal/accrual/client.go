package accrual

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/apperrors"
	"github.com/vyacheslavskl/go-musthave-diploma-tpl/internal/models"
)

type Client struct {
	baseURL string
	client  *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

type Result struct {
	Data       *models.AccrualResponse
	RetryAfter time.Duration
}

// GetAccrualInfo делает запрос в accrual систему
func (c *Client) GetAccrualInfo(ctx context.Context, number string) (*Result, error) {
	url := fmt.Sprintf("%s/api/orders/%s", c.baseURL, number)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {

	case http.StatusOK:
		var data models.AccrualResponse
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, err
		}
		return &Result{Data: &data}, nil

	case http.StatusNoContent:
		// заказ не зарегистрирован в системе начислений
		return nil, apperrors.ErrOrderNotFound

	case http.StatusTooManyRequests:
		retryAfter := 60 * time.Second // дефолт

		if h := resp.Header.Get("Retry-After"); h != "" {
			if sec, err := strconv.Atoi(h); err == nil {
				retryAfter = time.Duration(sec) * time.Second
			}
		}

		return &Result{RetryAfter: retryAfter}, apperrors.ErrTooManyRequests

	default:
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}
