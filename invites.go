package praxicraft

import "strings"

// InvitesResource wraps invitation Public API endpoints.
type InvitesResource struct {
	client *Client
}

func (r *InvitesResource) List(params map[string]any) (Page[Invite], error) {
	return DoAs[Page[Invite]](r.client, "GET", "/invites/", &RequestOptions{Params: params})
}

func (r *InvitesResource) Retrieve(inviteToken string) (Invite, error) {
	token, err := pathSegment(inviteToken, "invite_token")
	if err != nil {
		return Invite{}, err
	}
	return DoAs[Invite](r.client, "GET", "/invites/"+token+"/", nil)
}

// InviteCreateParams are fields for creating one invitation.
type InviteCreateParams struct {
	Email       string
	Name        string
	Role        string
	ExpiresDays *int
	SendEmail   *bool
	Extra       map[string]any
}

func (r *InvitesResource) Create(assessment string, p InviteCreateParams) (Invite, error) {
	if strings.TrimSpace(p.Email) == "" {
		return Invite{}, &APIError{Message: "email is required", ErrCode: "INVALID_ARGUMENT"}
	}
	body := map[string]any{"email": p.Email}
	for k, v := range p.Extra {
		body[k] = v
	}
	if p.Name != "" {
		body["name"] = p.Name
	}
	if p.Role != "" {
		body["role"] = p.Role
	}
	if p.ExpiresDays != nil {
		body["expires_days"] = *p.ExpiresDays
	}
	if p.SendEmail != nil {
		body["send_email"] = *p.SendEmail
	}
	key, err := pathSegment(assessment, "assessment")
	if err != nil {
		return Invite{}, err
	}
	return DoAs[Invite](r.client, "POST", "/assessments/"+key+"/invites/", &RequestOptions{JSON: body})
}

func (r *InvitesResource) BulkCreate(assessment string, candidates []map[string]any, sendEmail *bool, extra map[string]any) (map[string]any, error) {
	body := map[string]any{"candidates": candidates}
	for k, v := range extra {
		body[k] = v
	}
	if sendEmail != nil {
		body["send_email"] = *sendEmail
	}
	key, err := pathSegment(assessment, "assessment")
	if err != nil {
		return nil, err
	}
	return DoAs[map[string]any](r.client, "POST", "/assessments/"+key+"/invites/bulk/", &RequestOptions{JSON: body})
}

func (r *InvitesResource) Remind(inviteToken string) (Invite, error) {
	token, err := pathSegment(inviteToken, "invite_token")
	if err != nil {
		return Invite{}, err
	}
	return DoAs[Invite](r.client, "POST", "/invites/"+token+"/remind/", nil)
}

func (r *InvitesResource) Cancel(inviteToken string) error {
	token, err := pathSegment(inviteToken, "invite_token")
	if err != nil {
		return err
	}
	_, err = DoAs[struct{}](r.client, "DELETE", "/invites/"+token+"/", nil)
	return err
}
