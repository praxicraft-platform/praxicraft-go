package praxicraft

import (
	"net/url"
	"strings"
)

const maxResultPages = 10_000

// ResultsResource wraps cohort / candidate result endpoints.
type ResultsResource struct {
	client *Client
}

func (r *ResultsResource) List(assessment string, cursor string, pageSize *int, params map[string]any) (Page[ResultRow], error) {
	query := map[string]any{}
	for k, v := range params {
		query[k] = v
	}
	if cursor != "" {
		query["cursor"] = cursor
	}
	if pageSize != nil {
		query["page_size"] = *pageSize
	}
	key, err := pathSegment(assessment, "assessment")
	if err != nil {
		return Page[ResultRow]{}, err
	}
	return DoAs[Page[ResultRow]](r.client, "GET", "/assessments/"+key+"/results/", &RequestOptions{Params: query})
}

func (r *ResultsResource) Retrieve(inviteToken string) (ResultRow, error) {
	token, err := pathSegment(inviteToken, "invite_token")
	if err != nil {
		return ResultRow{}, err
	}
	return DoAs[ResultRow](r.client, "GET", "/invites/"+token+"/result/", nil)
}

// IterAll yields each result row, following cursor / next pagination links.
func (r *ResultsResource) IterAll(assessment string, pageSize *int, params map[string]any, fn func(row ResultRow) error) error {
	cursor := ""
	seen := map[string]struct{}{}
	for range maxResultPages {
		page, err := r.List(assessment, cursor, pageSize, params)
		if err != nil {
			return err
		}
		for _, row := range page.Results {
			if err := fn(row); err != nil {
				return err
			}
		}
		next := nextCursorFromPage(page.NextCursor, page.Next)
		if next == "" {
			return nil
		}
		if _, exists := seen[next]; exists {
			return nil
		}
		seen[next] = struct{}{}
		cursor = next
	}
	return nil
}

func nextCursorFromPage(nextCursor, nextLink string) string {
	if nextCursor != "" {
		return nextCursor
	}
	if nextLink == "" {
		return ""
	}
	u, err := url.Parse(nextLink)
	if err != nil {
		return ""
	}
	values := u.Query()["cursor"]
	if len(values) == 0 || strings.TrimSpace(values[0]) == "" {
		return ""
	}
	return values[0]
}
