package praxicraft

// OrgResource wraps organisation summary endpoints.
type OrgResource struct {
	client *Client
}

func (r *OrgResource) Retrieve() (Org, error) {
	return DoAs[Org](r.client, "GET", "/org/", nil)
}

func (r *OrgResource) Stats(params map[string]any) (map[string]any, error) {
	return DoAs[map[string]any](r.client, "GET", "/org/stats/", &RequestOptions{Params: params})
}
