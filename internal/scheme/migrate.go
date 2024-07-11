package scheme

import (
	"database/sql"

	"github.com/GuiaBolso/darwin"
)

var migrations = []darwin.Migration{
	{
		Version:     1,
		Description: "Create uuid extension",
		Script:      `CREATE EXTENSION "uuid-ossp";`,
	},
	{
		Version:     2,
		Description: "Create tasks Table",
		Script: `
			CREATE TABLE tasks (
				id uuid NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4 (),
				subject_class_id uuid NOT NULL,
				type char(5) NOT NULL,
				name varchar(45) NOT NULL,
				description varchar(128) NOT NULL,
				end_date timestamptz NOT NULL,
				created_at timestamptz NOT NULL DEFAULT timezone('utc', NOW()),
				updated_at timestamptz NOT NULL DEFAULT timezone('utc', NOW()),
				updated_by uuid
			);
		`,
	},
	{
		Version:     3,
		Description: "Create task_files Table",
		Script: `
			CREATE TABLE task_files (
				id uuid NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4 (),
				task_id uuid NOT NULL,
				name varchar(45) NOT NULL,
				description varchar(128) NOT NULL,
				storage_id uuid NOT NULL,
				created_at timestamptz NOT NULL DEFAULT timezone('utc', NOW()),
				updated_at timestamptz NOT NULL DEFAULT timezone('utc', NOW()),
				updated_by uuid,
				CONSTRAINT fk_task_files_to_tasks FOREIGN KEY(task_id) REFERENCES tasks(id) ON DELETE CASCADE
			);
		`,
	},
	{
		Version:     4,
		Description: "Create student_tasks Table",
		Script: `
			CREATE TABLE student_tasks (
				id uuid NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4 (),
				task_id uuid NOT NULL,
				student_id uuid NOT NULL,
				answer text NOT NULL,
				score smallint NOT NULL,
				feedback varchar(255),
				feedback_file varchar(255),
				created_at timestamptz NOT NULL DEFAULT timezone('utc', NOW()),
				CONSTRAINT fk_student_tasks_to_tasks FOREIGN KEY(task_id) REFERENCES tasks(id) ON DELETE CASCADE
			);
		`,
	},
	{
		Version:     5,
		Description: "Create student_task_files Table",
		Script: `
			CREATE TABLE student_task_files (
				id uuid NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4 (),
				student_task_id uuid NOT NULL,
				name varchar(45) NOT NULL,
				description varchar(128) NOT NULL,
				file_type char(1) NOT NULL,
				storage_id uuid,
				source varchar(255),
				created_at timestamptz NOT NULL DEFAULT timezone('utc', NOW()),
				updated_at timestamptz NOT NULL DEFAULT timezone('utc', NOW()),
				updated_by uuid,
				CONSTRAINT fk_student_task_files_to_student_tasks FOREIGN KEY(student_task_id) REFERENCES student_tasks(id) ON DELETE CASCADE
			);
		`,
	},
	{
		Version:     6,
		Description: "Create feedback_files Table",
		Script: `
			CREATE TABLE feedback_files (
				id uuid NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4 (),
				student_task_id uuid NOT NULL,
				name varchar(45) NOT NULL,
				description varchar(128) NOT NULL,
				storage_id uuid,
				created_at timestamptz NOT NULL DEFAULT timezone('utc', NOW()),
				updated_at timestamptz NOT NULL DEFAULT timezone('utc', NOW()),
				updated_by uuid,
				CONSTRAINT fk_feedback_files_to_student_tasks FOREIGN KEY(student_task_id) REFERENCES student_tasks(id) ON DELETE CASCADE
			);
		`,
	},
}

// Migrate attempts to bring the schema for db up to date with the migrations
// defined in this package.
func Migrate(db *sql.DB) error {
	driver := darwin.NewGenericDriver(db, darwin.PostgresDialect{})

	d := darwin.New(driver, migrations, nil)

	return d.Migrate()
}
