package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mediaharvester/tg-downloader/shared/models"
	"github.com/redis/go-redis/v9"
)

const (
	QueueName = "download_queue"
)

type QueueService struct {
	client *redis.Client
}

func NewQueueService(client *redis.Client) *QueueService {
	return &QueueService{
		client: client,
	}
}

func (s *QueueService) PushJob(ctx context.Context, job *models.DownloadJob) error {
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
