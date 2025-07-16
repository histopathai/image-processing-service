package adapter

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

type GCSAdapter struct {
	client    *storage.Client
	bucket    string
	ProjectID string
	numWorker int
}

type GCSManager interface {
	UploadFile(ctx context.Context, filePath string, objectName string) error
	DownloadFile(ctx context.Context, objectName string, destinationPath string) error
	DeleteFile(ctx context.Context, objectName string) error
	ListFiles(ctx context.Context, prefix string) ([]string, error)
	CreateBucket(ctx context.Context, bucketName string) error
}

func NewGCSAdapter(projectID, bucket string, numWorker int) (*GCSAdapter, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create GCS client: %v", err)
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}
	return &GCSAdapter{
		client:    client,
		bucket:    bucket,
		ProjectID: projectID,
		numWorker: numWorker,
	}, nil
}

func (g *GCSAdapter) UploadFile(ctx context.Context, filePath string, objectName string) error {
	bucket := g.client.Bucket(g.bucket)
	obj := bucket.Object(objectName)

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer f.Close()

	w := obj.NewWriter(ctx)
	if _, err := io.Copy(w, f); err != nil {
		return fmt.Errorf("failed to upload file %s: %w", filePath, err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close writer for file %s: %w", filePath, err)
	}
	return nil
}

func (g *GCSAdapter) DownloadFile(ctx context.Context, objectName string, destinationPath string) error {
	bucket := g.client.Bucket(g.bucket)
	obj := bucket.Object(objectName)

	r, err := obj.NewReader(ctx)
	if err != nil {
		return fmt.Errorf("failed to create reader for object %s: %w", objectName, err)
	}
	defer r.Close()

	f, err := os.Create(destinationPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", destinationPath, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("failed to download object %s: %w", objectName, err)
	}
	return nil
}

func (g *GCSAdapter) DeleteFile(ctx context.Context, objectName string) error {
	bucket := g.client.Bucket(g.bucket)
	obj := bucket.Object(objectName)

	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete object %s: %w", objectName, err)
	}
	return nil
}
func (g *GCSAdapter) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	var files []string
	it := g.client.Bucket(g.bucket).Objects(ctx, &storage.Query{Prefix: prefix})

	for {
		attr, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list files with prefix %s: %w", prefix, err)
		}
		files = append(files, attr.Name)
	}
	return files, nil

}

func (g *GCSAdapter) CreateBucket(ctx context.Context, bucketName string) error {
	projectID := g.ProjectID
	if projectID == "" {
		return fmt.Errorf("project ID is not set")
	}

	err := g.client.Bucket(bucketName).Create(ctx, projectID, nil)
	if err != nil {
		return fmt.Errorf("failed to create bucket %s: %w", bucketName, err)
	}
	return nil
}

func (g *GCSAdapter) UploadDir(ctx context.Context, localDir string, gcsPrefix string) error {
	type uploadJob struct {
		localPath  string
		objectName string
	}

	jobs := make(chan uploadJob)
	errCh := make(chan error, 1)

	if g.numWorker <= 0 {
		g.numWorker = runtime.NumCPU()
	}
	workerCount := g.numWorker
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for job := range jobs {
			if err := g.UploadFile(ctx, job.localPath, job.objectName); err != nil {
				select {
				case errCh <- fmt.Errorf("failed to upload file %s: %w", job.localPath, err):
				default:
				}
				return
			}
		}
	}

	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go worker()
	}

	go func() {
		defer close(jobs)
		filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				select {
				case errCh <- err:
				default:
				}
				return err
			}
			if info.IsDir() {
				return nil
			}

			relPath, err := filepath.Rel(localDir, path)
			if err != nil {
				select {
				case errCh <- err:
				default:
				}
				return err
			}

			objectName := filepath.ToSlash(filepath.Join(gcsPrefix, relPath))
			jobs <- uploadJob{localPath: path, objectName: objectName}
			return nil
		})
	}()

	wg.Wait()

	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}
