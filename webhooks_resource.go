package praxicraft

import "strings"

// WebhooksResource wraps webhook CRUD + test endpoints.
type WebhooksResource struct {
	client *Client
}

func (r *WebhooksResource) List(params map[string]any) (Page[WebhookEndpoint], error) {
	return DoAs[Page[WebhookEndpoint]](r.client, "GET", "/webhooks/", &RequestOptions{Params: params})
}

func (r *WebhooksResource) Create(hookURL string, events []string, extra map[string]any) (WebhookEndpoint, error) {
	if strings.TrimSpace(hookURL) == "" {
		return WebhookEndpoint{}, &APIError{Message: "url is required", ErrCode: "INVALID_ARGUMENT"}
	}
	if len(events) == 0 {
		return WebhookEndpoint{}, &APIError{Message: "events must be a non-empty list", ErrCode: "INVALID_ARGUMENT"}
	}
	body := map[string]any{"url": hookURL, "events": events}
	for k, v := range extra {
		body[k] = v
	}
	return DoAs[WebhookEndpoint](r.client, "POST", "/webhooks/create/", &RequestOptions{JSON: body})
}

func (r *WebhooksResource) Retrieve(webhookID string) (WebhookEndpoint, error) {
	key, err := pathSegment(webhookID, "webhook_id")
	if err != nil {
		return WebhookEndpoint{}, err
	}
	return DoAs[WebhookEndpoint](r.client, "GET", "/webhooks/"+key+"/", nil)
}

func (r *WebhooksResource) Update(webhookID string, fields map[string]any) (WebhookEndpoint, error) {
	if len(fields) == 0 {
		return WebhookEndpoint{}, &APIError{Message: "Update requires at least one field to change", ErrCode: "INVALID_ARGUMENT"}
	}
	key, err := pathSegment(webhookID, "webhook_id")
	if err != nil {
		return WebhookEndpoint{}, err
	}
	return DoAs[WebhookEndpoint](r.client, "PATCH", "/webhooks/"+key+"/", &RequestOptions{JSON: fields})
}

func (r *WebhooksResource) Delete(webhookID string) error {
	key, err := pathSegment(webhookID, "webhook_id")
	if err != nil {
		return err
	}
	_, err = DoAs[struct{}](r.client, "DELETE", "/webhooks/"+key+"/", nil)
	return err
}

func (r *WebhooksResource) Deliveries(webhookID string) (map[string]any, error) {
	key, err := pathSegment(webhookID, "webhook_id")
	if err != nil {
		return nil, err
	}
	return DoAs[map[string]any](r.client, "GET", "/webhooks/"+key+"/deliveries/", nil)
}

func (r *WebhooksResource) Test(webhookID string) (map[string]any, error) {
	key, err := pathSegment(webhookID, "webhook_id")
	if err != nil {
		return nil, err
	}
	return DoAs[map[string]any](r.client, "POST", "/webhooks/"+key+"/test/", nil)
}
