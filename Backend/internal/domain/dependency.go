package domain

import (
	"project-management-system/pkg/errcode"
)

type TaskDependency struct {
	ID               string `json:"id"`
	ProjectID        string `json:"project_id"`
	TaskID           string `json:"task_id"`
	DependsOnTaskID  string `json:"depends_on_task_id"`
	DependencyType   string `json:"dependency_type"` // FS = Finish-to-Start
}

func (d *TaskDependency) Validate() error {
	if d.TaskID == d.DependsOnTaskID {
		return errcode.New(errcode.ErrValidationFailed, "Task cannot depend on itself (self-dependency)", 422, nil)
	}
	if d.DependencyType == "" {
		d.DependencyType = "FS"
	}
	return nil
}
