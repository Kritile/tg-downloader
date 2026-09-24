package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mediaharvester/tg-downloader/bot/internal/domain"
	"github.com/mediaharvester/tg-downloader/shared/models"
	"github.com/redis/go-redis/v9"
)

const (
	QueueName = "download_queue"
)

type QueueService struct {
	client  *redis.Client
	jobRepo domain.JobRepository
}

func NewQueueService(client *redis.Client, repos ...domain.JobRepository) *QueueService {
	var jobRepo domain.JobRepository
	if len(repos) > 0 {
		jobRepo = repos[0]
	}
	return &QueueService{
		client:  client,
		jobRepo: jobRepo,
	}
}

func (s *QueueService) PushJob(ctx context.Context, job *models.DownloadJob) error {
	if s.jobRepo != nil {
		record := &models.DownloadJobRecord{Platform: job.Platform, UserID: job.UserID, ChatID: job.ChatID, URL: job.URL, Source: job.Source, Format: job.Format, Status: string(models.StatusPending), StatusMessageID: job.StatusMessageID, QueuedAt: time.Now(), UpdatedAt: time.Now()}
		if err := s.jobRepo.Create(ctx, record); err != nil {
			return err
		}
		job.ID = record.ID
	}
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	return s.client.LPush(ctx, QueueName, data).Err()
}

func (s *QueueService) PopJob(ctx context.Context) (*models.DownloadJob, error) {
	result, err := s.client.BRPop(ctx, 0, QueueName).Result()
	if err != nil {
		return nil, err
	}

	if len(result) < 2 {
		return nil, fmt.Errorf("invalid response from Redis")
	}

	var job models.DownloadJob
	err = json.Unmarshal([]byte(result[1]), &job)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

func (s *QueueService) GetQueueLength(ctx context.Context) (int64, error) {
	return s.client.LLen(ctx, QueueName).Result()
}
