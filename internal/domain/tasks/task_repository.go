package tasks

import (
	"context"
	"database/sql"
	"fmt"

	tasksPb "lms-task-service/pb/tasks"
)

// TaskRepository struct
type TaskRepository struct {
	Db *sql.DB
}

// Create task
func (r *TaskRepository) Create(ctx context.Context, t *tasksPb.Task) error {
	query := `
		INSERT INTO tasks (subject_class_id, type, name, description, end_date, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	return r.Db.QueryRowContext(ctx, query,
		t.SubjectClassId, t.Type, t.Name, t.Description, t.EndDate, t.UpdatedBy,
	).Scan(&t.Id, &t.CreatedAt, &t.UpdatedAt)
}

// CreateTaskFile creates a file associated with a task
func (r *TaskRepository) CreateTaskFile(ctx context.Context, f *tasksPb.TaskFile) error {
	query := `
		INSERT INTO task_files (task_id, name, description, storage_id, updated_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	return r.Db.QueryRowContext(ctx, query,
		f.TaskId, f.Name, f.Description, f.StorageId, f.UpdatedBy,
	).Scan(&f.Id, &f.CreatedAt, &f.UpdatedAt)
}

// Get task by ID with its files
func (r *TaskRepository) Get(ctx context.Context, id string) (*tasksPb.Task, error) {
	query := `
		SELECT id, subject_class_id, type, name, description,
			end_date, created_at, updated_at, COALESCE(updated_by::text, '')
		FROM tasks WHERE id = $1
	`

	var t tasksPb.Task
	err := r.Db.QueryRowContext(ctx, query, id).Scan(
		&t.Id, &t.SubjectClassId, &t.Type, &t.Name, &t.Description,
		&t.EndDate, &t.CreatedAt, &t.UpdatedAt, &t.UpdatedBy,
	)
	if err != nil {
		return nil, err
	}

	// Get task files
	files, err := r.GetTaskFiles(ctx, id)
	if err != nil {
		return nil, err
	}
	t.Files = files

	return &t, nil
}

// GetTaskFiles returns files for a task
func (r *TaskRepository) GetTaskFiles(ctx context.Context, taskID string) ([]*tasksPb.TaskFile, error) {
	query := `
		SELECT id, task_id, name, description, storage_id,
			created_at, updated_at, COALESCE(updated_by::text, '')
		FROM task_files WHERE task_id = $1
	`

	rows, err := r.Db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*tasksPb.TaskFile
	for rows.Next() {
		var f tasksPb.TaskFile
		err := rows.Scan(
			&f.Id, &f.TaskId, &f.Name, &f.Description, &f.StorageId,
			&f.CreatedAt, &f.UpdatedAt, &f.UpdatedBy,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, &f)
	}

	return files, nil
}

// Update task
func (r *TaskRepository) Update(ctx context.Context, t *tasksPb.Task) error {
	query := `
		UPDATE tasks SET
			subject_class_id = $1, type = $2, name = $3,
			description = $4, end_date = $5, updated_by = $6, updated_at = NOW()
		WHERE id = $7
	`

	result, err := r.Db.ExecContext(ctx, query,
		t.SubjectClassId, t.Type, t.Name, t.Description,
		t.EndDate, t.UpdatedBy, t.Id,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("task not found")
	}

	return nil
}

// Delete task (cascade deletes files and student tasks)
func (r *TaskRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM tasks WHERE id = $1`

	result, err := r.Db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("task not found")
	}

	return nil
}

// StudentTaskRepository struct
type StudentTaskRepository struct {
	Db *sql.DB
}

// Create student task (submit)
func (r *StudentTaskRepository) Create(ctx context.Context, st *tasksPb.StudentTask) error {
	query := `
		INSERT INTO student_tasks (task_id, student_id, answer, score)
		VALUES ($1, $2, $3, 0)
		RETURNING id, score, created_at
	`

	return r.Db.QueryRowContext(ctx, query,
		st.TaskId, st.StudentId, st.Answer,
	).Scan(&st.Id, &st.Score, &st.CreatedAt)
}

// CreateStudentTaskFile creates a file for a student task submission
func (r *StudentTaskRepository) CreateStudentTaskFile(ctx context.Context, f *tasksPb.StudentTaskFile) error {
	query := `
		INSERT INTO student_task_files (student_task_id, name, description, file_type, storage_id, source, updated_by)
		VALUES ($1, $2, $3, $4, NULLIF($5, '')::uuid, NULLIF($6, ''), $7)
		RETURNING id, created_at, updated_at
	`

	return r.Db.QueryRowContext(ctx, query,
		f.StudentTaskId, f.Name, f.Description, f.FileType, f.StorageId, f.Source, f.UpdatedBy,
	).Scan(&f.Id, &f.CreatedAt, &f.UpdatedAt)
}

// Grade updates score and feedback for a student task
func (r *StudentTaskRepository) Grade(ctx context.Context, studentTaskID string, score int32, feedback string) error {
	query := `
		UPDATE student_tasks SET score = $1, feedback = $2
		WHERE id = $3
	`

	result, err := r.Db.ExecContext(ctx, query, score, feedback, studentTaskID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("student task not found")
	}

	return nil
}

// CreateFeedbackFile creates a feedback file for a student task
func (r *StudentTaskRepository) CreateFeedbackFile(ctx context.Context, f *tasksPb.FeedbackFile) error {
	query := `
		INSERT INTO feedback_files (student_task_id, name, description, storage_id, updated_by)
		VALUES ($1, $2, $3, NULLIF($4, '')::uuid, $5)
		RETURNING id, created_at, updated_at
	`

	return r.Db.QueryRowContext(ctx, query,
		f.StudentTaskId, f.Name, f.Description, f.StorageId, f.UpdatedBy,
	).Scan(&f.Id, &f.CreatedAt, &f.UpdatedAt)
}

// Get student task by ID with files and feedback files
func (r *StudentTaskRepository) Get(ctx context.Context, id string) (*tasksPb.StudentTask, error) {
	query := `
		SELECT id, task_id, student_id, answer, score, COALESCE(feedback, ''), created_at
		FROM student_tasks WHERE id = $1
	`

	var st tasksPb.StudentTask
	err := r.Db.QueryRowContext(ctx, query, id).Scan(
		&st.Id, &st.TaskId, &st.StudentId, &st.Answer, &st.Score, &st.Feedback, &st.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Get student task files
	files, err := r.GetStudentTaskFiles(ctx, id)
	if err != nil {
		return nil, err
	}
	st.Files = files

	// Get feedback files
	feedbackFiles, err := r.GetFeedbackFiles(ctx, id)
	if err != nil {
		return nil, err
	}
	st.FeedbackFiles = feedbackFiles

	return &st, nil
}

// GetStudentTaskFiles returns files for a student task
func (r *StudentTaskRepository) GetStudentTaskFiles(ctx context.Context, studentTaskID string) ([]*tasksPb.StudentTaskFile, error) {
	query := `
		SELECT id, student_task_id, name, description, file_type,
			COALESCE(storage_id::text, ''), COALESCE(source, ''),
			created_at, updated_at, COALESCE(updated_by::text, '')
		FROM student_task_files WHERE student_task_id = $1
	`

	rows, err := r.Db.QueryContext(ctx, query, studentTaskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*tasksPb.StudentTaskFile
	for rows.Next() {
		var f tasksPb.StudentTaskFile
		err := rows.Scan(
			&f.Id, &f.StudentTaskId, &f.Name, &f.Description, &f.FileType,
			&f.StorageId, &f.Source, &f.CreatedAt, &f.UpdatedAt, &f.UpdatedBy,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, &f)
	}

	return files, nil
}

// GetFeedbackFiles returns feedback files for a student task
func (r *StudentTaskRepository) GetFeedbackFiles(ctx context.Context, studentTaskID string) ([]*tasksPb.FeedbackFile, error) {
	query := `
		SELECT id, student_task_id, name, description,
			COALESCE(storage_id::text, ''), created_at, updated_at, COALESCE(updated_by::text, '')
		FROM feedback_files WHERE student_task_id = $1
	`

	rows, err := r.Db.QueryContext(ctx, query, studentTaskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*tasksPb.FeedbackFile
	for rows.Next() {
		var f tasksPb.FeedbackFile
		err := rows.Scan(
			&f.Id, &f.StudentTaskId, &f.Name, &f.Description,
			&f.StorageId, &f.CreatedAt, &f.UpdatedAt, &f.UpdatedBy,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, &f)
	}

	return files, nil
}

// List student tasks by task ID
func (r *StudentTaskRepository) List(ctx context.Context, taskID string, limit, offset uint32, orderBy, sort string) ([]*tasksPb.StudentTask, uint32, error) {
	query := `
		SELECT id, task_id, student_id, answer, score, COALESCE(feedback, ''), created_at
		FROM student_tasks
		WHERE task_id = $1
		ORDER BY ` + orderBy + ` ` + sort + `
		LIMIT $2 OFFSET $3
	`

	rows, err := r.Db.QueryContext(ctx, query, taskID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var studentTasks []*tasksPb.StudentTask
	for rows.Next() {
		var st tasksPb.StudentTask
		err := rows.Scan(
			&st.Id, &st.TaskId, &st.StudentId, &st.Answer, &st.Score, &st.Feedback, &st.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		// Get files for each student task
		files, err := r.GetStudentTaskFiles(ctx, st.Id)
		if err != nil {
			return nil, 0, err
		}
		st.Files = files

		// Get feedback files
		feedbackFiles, err := r.GetFeedbackFiles(ctx, st.Id)
		if err != nil {
			return nil, 0, err
		}
		st.FeedbackFiles = feedbackFiles

		studentTasks = append(studentTasks, &st)
	}

	countQuery := `SELECT COUNT(*) FROM student_tasks WHERE task_id = $1`
	var count uint32
	err = r.Db.QueryRowContext(ctx, countQuery, taskID).Scan(&count)
	if err != nil {
		return nil, 0, err
	}

	return studentTasks, count, nil
}
