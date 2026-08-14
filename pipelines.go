package praxicraft

import "strings"

// PipelinesResource wraps hiring pipeline list / enroll endpoints.
type PipelinesResource struct {
	client *Client
}

func (r *PipelinesResource) List(params map[string]any) (Page[Pipeline], error) {
	return DoAs[Page[Pipeline]](r.client, "GET", "/pipelines/", &RequestOptions{Params: params})
}

func (r *PipelinesResource) Retrieve(pipeline string) (Pipeline, error) {
	key, err := pathSegment(pipeline, "pipeline")
	if err != nil {
		return Pipeline{}, err
	}
	return DoAs[Pipeline](r.client, "GET", "/pipelines/"+key+"/", nil)
}

// PipelineEnrollParams are fields for enrolling one candidate.
type PipelineEnrollParams struct {
	Email     string
	Name      string
	SendEmail *bool
	Extra     map[string]any
}

func (r *PipelinesResource) Enroll(pipeline string, p PipelineEnrollParams) (Enrollment, error) {
	if strings.TrimSpace(p.Email) == "" {
		return Enrollment{}, &APIError{Message: "email is required", ErrCode: "INVALID_ARGUMENT"}
	}
	body := map[string]any{"email": p.Email}
	for k, v := range p.Extra {
		body[k] = v
	}
	if p.Name != "" {
		body["name"] = p.Name
	}
	if p.SendEmail != nil {
		body["send_email"] = *p.SendEmail
	}
	key, err := pathSegment(pipeline, "pipeline")
	if err != nil {
		return Enrollment{}, err
	}
	return DoAs[Enrollment](r.client, "POST", "/pipelines/"+key+"/enroll/", &RequestOptions{JSON: body})
}

func (r *PipelinesResource) BulkEnroll(pipeline string, candidates []map[string]any, sendEmail *bool, extra map[string]any) (map[string]any, error) {
	body := map[string]any{"candidates": candidates}
	for k, v := range extra {
		body[k] = v
	}
	if sendEmail != nil {
		body["send_email"] = *sendEmail
	}
	key, err := pathSegment(pipeline, "pipeline")
	if err != nil {
		return nil, err
	}
	return DoAs[map[string]any](r.client, "POST", "/pipelines/"+key+"/enroll/bulk/", &RequestOptions{JSON: body})
}

func (r *PipelinesResource) ListEnrollments(pipeline string, params map[string]any) (Page[Enrollment], error) {
	key, err := pathSegment(pipeline, "pipeline")
	if err != nil {
		return Page[Enrollment]{}, err
	}
	return DoAs[Page[Enrollment]](r.client, "GET", "/pipelines/"+key+"/enrollments/", &RequestOptions{Params: params})
}

func (r *PipelinesResource) GetEnrollment(enrollmentID string) (Enrollment, error) {
	key, err := pathSegment(enrollmentID, "enrollment_id")
	if err != nil {
		return Enrollment{}, err
	}
	return DoAs[Enrollment](r.client, "GET", "/pipelines/enrollments/"+key+"/", nil)
}
