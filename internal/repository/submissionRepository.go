package repository

import (
	"IFJudger/internal/models"
	customErrors "IFJudger/internal/models/errors"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type SubmissionRepository struct {
	DB *sql.DB
}

func StartSubmissionRepository(db *sql.DB) (*SubmissionRepository, error) {
	createTableSQL := `CREATE TABLE IF NOT EXISTS submissions (
		id TEXT PRIMARY KEY,
		status TEXT NOT NULL,
		result_json TEXT,
		error_message TEXT,
		job_data TEXT, 
		updated_at DATETIME
	);`

	if _, err := db.Exec(createTableSQL); err != nil {
		return nil, fmt.Errorf("falha ao criar tabela submissions: %w", err)
	}

	_, _ = db.Exec("PRAGMA journal_mode=WAL;")
	_, _ = db.Exec("PRAGMA synchronous = NORMAL;")
	_, _ = db.Exec("PRAGMA busy_timeout = 5000;")

	return &SubmissionRepository{
		DB: db,
	}, nil
}

func (r *SubmissionRepository) CreateJob(job models.Job) error {
	jobDataJSON, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("falha ao serializar job data: %w", err)
	}

	query := `INSERT INTO submissions (id, status, result_json, error_message, job_data, updated_at) 
              VALUES (?, ?, ?, ?, ?, ?)`

	_, err = r.DB.Exec(query, job.ID, models.StatusQueued, "", "", string(jobDataJSON), time.Now())
	if err != nil {
		return fmt.Errorf("falha ao criar job inicial: %w", err)
	}

	return nil
}

func (r *SubmissionRepository) UpdateResult(result models.JobResult) error {
	resultJSON, err := json.Marshal(result.Result)
	if err != nil {
		return fmt.Errorf("falha ao serializar resultado: %w", err)
	}

	query := `UPDATE submissions 
              SET status = ?, result_json = ?, error_message = ?, updated_at = ? 
              WHERE id = ?`

	_, err = r.DB.Exec(query, result.Status, string(resultJSON), result.ErrorMessage, time.Now(), result.ID)
	if err != nil {
		return fmt.Errorf("falha ao atualizar job: %w", err)
	}

	return nil
}

func (r *SubmissionRepository) GetByID(id string) (models.JobResult, error) {
	query := `SELECT id, status, result_json, error_message FROM submissions WHERE id = ?`
	row := r.DB.QueryRow(query, id)

	var res models.JobResult
	var jsonString string

	err := row.Scan(&res.ID, &res.Status, &jsonString, &res.ErrorMessage)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.JobResult{}, customErrors.ErrNotFound
		}
		return models.JobResult{}, err
	}

	if len(jsonString) > 0 {
		if err := json.Unmarshal([]byte(jsonString), &res.Result); err != nil {
			return models.JobResult{}, fmt.Errorf("falha ao desserializar result: %w", err)
		}
	}

	return res, nil
}

func (r *SubmissionRepository) GetRecoverableJobs() ([]models.Job, error) {
	query := `SELECT job_data FROM submissions WHERE status IN (?, ?)`

	rows, err := r.DB.Query(query, models.StatusQueued, models.StatusProcessing)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []models.Job
	for rows.Next() {
		var jobDataString string
		if err := rows.Scan(&jobDataString); err != nil {
			return nil, err
		}

		var job models.Job
		if err := json.Unmarshal([]byte(jobDataString), &job); err != nil {
			fmt.Printf("Erro ao recuperar job (JSON corrompido): %v\n", err)
			continue
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}
