package tasks

import (
	"context"
	"database/sql"
	"log"
	"time"

	"lms-task-service/internal/pkg/app"
	"lms-task-service/internal/pkg/db/redis"
	genericPb "lms-task-service/pb/generic"
	tasksPb "lms-task-service/pb/tasks"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TaskServiceServer struct
type TaskServiceServer struct {
	Db    *sql.DB
	Cache *redis.Cache
	Log   *log.Logger
	tasksPb.UnimplementedTaskServiceServer
}

// Create task (Issue #1)
func (s *TaskServiceServer) Create(ctx context.Context, in *tasksPb.TaskInput) (*tasksPb.Task, error) {
	repo := TaskRepository{Db: s.Db}

	userID := ctx.Value(app.Ctx("user_id")).(string)

	// Validation
	if in.GetSubjectClassId() == "" {
		return nil, status.Error(codes.InvalidArgument, "subject_class_id is required")
	}
	if in.GetType() == "" {
		return nil, status.Error(codes.InvalidArgument, "type is required")
	}
	if in.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if in.GetDescription() == "" {
		return nil, status.Error(codes.InvalidArgument, "description is required")
	}
	if in.GetEndDate() == "" {
		return nil, status.Error(codes.InvalidArgument, "end_date is required")
	}

	task := &tasksPb.Task{
		SubjectClassId: in.GetSubjectClassId(),
		Type:           in.GetType(),
		Name:           in.GetName(),
		Description:    in.GetDescription(),
		EndDate:        in.GetEndDate(),
		UpdatedBy:      userID,
	}

	err := repo.Create(ctx, task)
	if err != nil {
		s.Log.Printf("error creating task: %v", err)
		return nil, status.Error(codes.Internal, "failed to create task")
	}

	// Create task files
	for _, fileInput := range in.GetFiles() {
		f := &tasksPb.TaskFile{
			TaskId:      task.Id,
			Name:        fileInput.GetName(),
			Description: fileInput.GetDescription(),
			StorageId:   fileInput.GetStorageId(),
			UpdatedBy:   userID,
		}
		err := repo.CreateTaskFile(ctx, f)
		if err != nil {
			s.Log.Printf("error creating task file: %v", err)
			continue
		}
		task.Files = append(task.Files, f)
	}

	// Call gRPC to create post in lms-post-service
	go s.createPostForTask(ctx, task, userID)

	return task, nil
}

// createPostForTask calls lms-post-service to create a post with type TASK (3)
func (s *TaskServiceServer) createPostForTask(ctx context.Context, task *tasksPb.Task, userID string) {
	client := &postServiceClient{Log: s.Log}
	client.createPost(ctx, task.SubjectClassId, task.Id, task.Name, userID)
}

// Get task by ID (Issue #2)
func (s *TaskServiceServer) Get(ctx context.Context, in *genericPb.Id) (*tasksPb.Task, error) {
	repo := TaskRepository{Db: s.Db}

	if in.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	task, err := repo.Get(ctx, in.GetId())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "task not found")
		}
		s.Log.Printf("error getting task: %v", err)
		return nil, status.Error(codes.Internal, "failed to get task")
	}

	return task, nil
}

// Update task (Issue #3)
func (s *TaskServiceServer) Update(ctx context.Context, in *tasksPb.TaskUpdate) (*tasksPb.Task, error) {
	repo := TaskRepository{Db: s.Db}

	userID := ctx.Value(app.Ctx("user_id")).(string)

	// Validation
	if in.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if in.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	task := &tasksPb.Task{
		Id:             in.GetId(),
		SubjectClassId: in.GetSubjectClassId(),
		Type:           in.GetType(),
		Name:           in.GetName(),
		Description:    in.GetDescription(),
		EndDate:        in.GetEndDate(),
		UpdatedBy:      userID,
		UpdatedAt:      time.Now().UTC().Format(time.RFC3339),
	}

	err := repo.Update(ctx, task)
	if err != nil {
		s.Log.Printf("error updating task: %v", err)
		return nil, status.Error(codes.Internal, "failed to update task")
	}

	// Fetch updated task with files
	updated, err := repo.Get(ctx, task.Id)
	if err != nil {
		s.Log.Printf("error fetching updated task: %v", err)
		return task, nil
	}

	return updated, nil
}

// Delete task (Issue #4)
func (s *TaskServiceServer) Delete(ctx context.Context, in *genericPb.Id) (*genericPb.BoolMessage, error) {
	repo := TaskRepository{Db: s.Db}

	if in.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	err := repo.Delete(ctx, in.GetId())
	if err != nil {
		s.Log.Printf("error deleting task: %v", err)
		return &genericPb.BoolMessage{IsTrue: false}, status.Error(codes.Internal, "failed to delete task")
	}

	return &genericPb.BoolMessage{IsTrue: true}, nil
}

// StudentSubmit (Issue #5)
func (s *TaskServiceServer) StudentSubmit(ctx context.Context, in *tasksPb.StudentTaskInput) (*tasksPb.StudentTask, error) {
	studentRepo := StudentTaskRepository{Db: s.Db}

	userID := ctx.Value(app.Ctx("user_id")).(string)

	// Validation
	if in.GetTaskId() == "" {
		return nil, status.Error(codes.InvalidArgument, "task_id is required")
	}
	if in.GetStudentId() == "" {
		return nil, status.Error(codes.InvalidArgument, "student_id is required")
	}
	if in.GetAnswer() == "" {
		return nil, status.Error(codes.InvalidArgument, "answer is required")
	}

	studentTask := &tasksPb.StudentTask{
		TaskId:    in.GetTaskId(),
		StudentId: in.GetStudentId(),
		Answer:    in.GetAnswer(),
	}

	err := studentRepo.Create(ctx, studentTask)
	if err != nil {
		s.Log.Printf("error creating student task: %v", err)
		return nil, status.Error(codes.Internal, "failed to submit task")
	}

	// Create student task files
	for _, fileInput := range in.GetFiles() {
		f := &tasksPb.StudentTaskFile{
			StudentTaskId: studentTask.Id,
			Name:          fileInput.GetName(),
			Description:   fileInput.GetDescription(),
			FileType:      fileInput.GetFileType(),
			StorageId:     fileInput.GetStorageId(),
			Source:        fileInput.GetSource(),
			UpdatedBy:     userID,
		}
		err := studentRepo.CreateStudentTaskFile(ctx, f)
		if err != nil {
			s.Log.Printf("error creating student task file: %v", err)
			continue
		}
		studentTask.Files = append(studentTask.Files, f)
	}

	return studentTask, nil
}

// TeacherGrade (Issue #6)
func (s *TaskServiceServer) TeacherGrade(ctx context.Context, in *tasksPb.GradeInput) (*tasksPb.StudentTask, error) {
	studentRepo := StudentTaskRepository{Db: s.Db}

	userID := ctx.Value(app.Ctx("user_id")).(string)

	// Validation
	if in.GetStudentTaskId() == "" {
		return nil, status.Error(codes.InvalidArgument, "student_task_id is required")
	}

	err := studentRepo.Grade(ctx, in.GetStudentTaskId(), in.GetScore(), in.GetFeedback())
	if err != nil {
		s.Log.Printf("error grading student task: %v", err)
		return nil, status.Error(codes.Internal, "failed to grade task")
	}

	// Create feedback files
	for _, fileInput := range in.GetFeedbackFiles() {
		f := &tasksPb.FeedbackFile{
			StudentTaskId: in.GetStudentTaskId(),
			Name:          fileInput.GetName(),
			Description:   fileInput.GetDescription(),
			StorageId:     fileInput.GetStorageId(),
			UpdatedBy:     userID,
		}
		err := studentRepo.CreateFeedbackFile(ctx, f)
		if err != nil {
			s.Log.Printf("error creating feedback file: %v", err)
			continue
		}
	}

	// Return updated student task
	studentTask, err := studentRepo.Get(ctx, in.GetStudentTaskId())
	if err != nil {
		s.Log.Printf("error fetching graded student task: %v", err)
		return nil, status.Error(codes.Internal, "failed to fetch graded task")
	}

	return studentTask, nil
}

// ListStudentTasks (Issue #7)
func (s *TaskServiceServer) ListStudentTasks(ctx context.Context, in *tasksPb.StudentTaskListInput) (*tasksPb.StudentTaskList, error) {
	studentRepo := StudentTaskRepository{Db: s.Db}

	if in.GetTaskId() == "" {
		return nil, status.Error(codes.InvalidArgument, "task_id is required")
	}

	pagination := in.GetPagination()
	limit := uint32(10)
	offset := uint32(0)
	orderBy := "created_at"
	sort := "DESC"

	if pagination != nil {
		if pagination.Limit > 0 {
			limit = pagination.Limit
		}
		offset = pagination.Offset
		if pagination.Order != "" {
			orderBy = pagination.Order
		}
		if pagination.Sort != "" {
			sort = pagination.Sort
		}
	}

	studentTasks, count, err := studentRepo.List(ctx, in.GetTaskId(), limit, offset, orderBy, sort)
	if err != nil {
		s.Log.Printf("error listing student tasks: %v", err)
		return nil, status.Error(codes.Internal, "failed to list student tasks")
	}

	return &tasksPb.StudentTaskList{
		StudentTasks: studentTasks,
		Count:        count,
	}, nil
}
