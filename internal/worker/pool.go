package worker

import (
    "context"
    "sync"
	"github.com/pmkrish02/payflow-ai/internal/repository"
	"fmt"
)

type WorkerPool struct {
    jobs      chan Job
    workers   int
    wg        sync.WaitGroup
    auditRepo *repository.AuditRepository
}
type Job struct {
    Type    string
    Payload []byte
    UserID  string
}

func (wp *WorkerPool) Start(ctx context.Context) {
    for i := 0; i < wp.workers; i++ {
        wp.wg.Add(1)
        go func() {
			fmt.Println("Worker started")
            defer wp.wg.Done()
            for {
                select {
                case job := <-wp.jobs:
                    process(ctx, job, wp.auditRepo)
                case <-ctx.Done():
                    return
                }
            }
        }()
    }
}

func process(ctx context.Context, job Job, auditRepo *repository.AuditRepository) {
    switch job.Type {
    case "audit_log":
		err := auditRepo.CreateAuditLog(ctx, job.UserID, "transfer", "transaction", nil, nil)
		if err != nil {
			fmt.Println("Audit log error:", err)
		}
}
}

func NewWorkerPool(workers int, auditRepo *repository.AuditRepository) *WorkerPool {
    return &WorkerPool{
        jobs:      make(chan Job, 100),
        workers:   workers,
        auditRepo: auditRepo,
    }
}

func (wp *WorkerPool) Submit(job Job) {
    fmt.Println("Job submitted:", job.Type, job.UserID)
    wp.jobs <- job
}
