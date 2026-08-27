package praxicraft

import "strings"

// AssessmentsResource wraps /assessments/ Public API endpoints.
type AssessmentsResource struct {
	client *Client
}

func (r *AssessmentsResource) List(params map[string]any) (Page[Assessment], error) {
	return DoAs[Page[Assessment]](r.client, "GET", "/assessments/", &RequestOptions{Params: params})
}

func (r *AssessmentsResource) Retrieve(assessment string) (Assessment, error) {
	key, err := pathSegment(assessment, "assessment")
	if err != nil {
		return Assessment{}, err
	}
	return DoAs[Assessment](r.client, "GET", "/assessments/"+key+"/", nil)
}

func (r *AssessmentsResource) Create(fields map[string]any) (Assessment, error) {
	return DoAs[Assessment](r.client, "POST", "/assessments/create/", &RequestOptions{JSON: fields})
}

func (r *AssessmentsResource) Update(assessment string, fields map[string]any) (Assessment, error) {
	if len(fields) == 0 {
		return Assessment{}, &APIError{Message: "Update requires at least one field to change", ErrCode: "INVALID_ARGUMENT"}
	}
	key, err := pathSegment(assessment, "assessment")
	if err != nil {
		return Assessment{}, err
	}
	return DoAs[Assessment](r.client, "PATCH", "/assessments/"+key+"/update/", &RequestOptions{JSON: fields})
}

func (r *AssessmentsResource) Activate(assessment string) (Assessment, error) {
	return r.Update(assessment, map[string]any{"status": "active"})
}

func (r *AssessmentsResource) ListTasks(assessment string, params map[string]any) (Page[map[string]any], error) {
	key, err := pathSegment(assessment, "assessment")
	if err != nil {
		return Page[map[string]any]{}, err
	}
	return DoAs[Page[map[string]any]](r.client, "GET", "/assessments/"+key+"/tasks/", &RequestOptions{Params: params})
}

// AttachTasks attaches tasks. Pass "tasks" ([]map) and/or "task_id"/"source" in body.
func (r *AssessmentsResource) AttachTasks(assessment string, body map[string]any) (map[string]any, error) {
	if len(body) == 0 {
		return nil, &APIError{Message: "AttachTasks requires tasks or task_id", ErrCode: "INVALID_ARGUMENT"}
	}
	key, err := pathSegment(assessment, "assessment")
	if err != nil {
		return nil, err
	}
	return DoAs[map[string]any](r.client, "POST", "/assessments/"+key+"/tasks/attach/", &RequestOptions{JSON: body})
}

func (r *AssessmentsResource) ReplaceTasks(assessment string, tasks []map[string]any, extra map[string]any) (map[string]any, error) {
	body := map[string]any{"tasks": tasks}
	for k, v := range extra {
		body[k] = v
	}
	key, err := pathSegment(assessment, "assessment")
	if err != nil {
		return nil, err
	}
	return DoAs[map[string]any](r.client, "PUT", "/assessments/"+key+"/tasks/replace/", &RequestOptions{JSON: body})
}

func (r *AssessmentsResource) RemoveTask(assessment, assessmentTaskID string) error {
	key, err := pathSegment(assessment, "assessment")
	if err != nil {
		return err
	}
	taskID := strings.TrimSpace(assessmentTaskID)
	if taskID == "" {
		return &APIError{Message: "assessmentTaskID must be a non-empty string", ErrCode: "INVALID_ARGUMENT"}
	}
	_, err = DoAs[struct{}](r.client, "DELETE", "/assessments/"+key+"/tasks/remove/", &RequestOptions{
		JSON: map[string]any{"assessment_task_id": taskID},
	})
	return err
}
