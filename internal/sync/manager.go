package sync

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"

	"vault-copy/internal/config"
	"vault-copy/internal/vault"
)

type SyncStats struct {
	SecretsRead    int64
	SecretsWritten int64
	SecretsSkipped int64
	Errors         int64
}

type SyncManager struct {
	sourceClient *vault.Client
	destClient   *vault.Client
	config       *config.Config
}

func NewManager(sourceClient, destClient *vault.Client, cfg *config.Config) *SyncManager {
	return &SyncManager{
		sourceClient: sourceClient,
		destClient:   destClient,
		config:       cfg,
	}
}

func (m *SyncManager) Sync(ctx context.Context) (*SyncStats, error) {
	stats := &SyncStats{}

	log.Printf("Начинаем синхронизацию из %s в %s",
		m.config.SourcePath, m.config.DestinationPath)

	if m.config.DryRun {
		log.Println("Режим dry-run - секреты не будут записаны")
	}

	// Проверяем, является ли источник директорией
	isDir, err := m.sourceClient.IsDirectory(m.config.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("ошибка проверки пути источника: %v", err)
	}

	if isDir && !m.config.Recursive {
		return nil, fmt.Errorf("источник является директорией, используйте --recursive для копирования")
	}

	if !isDir {
		// Копирование одиночного секрета
		return m.syncSingleSecret(ctx, stats)
	}

	// Копирование директории
	return m.syncDirectory(ctx, stats)
}

func (m *SyncManager) syncSingleSecret(ctx context.Context, stats *SyncStats) (*SyncStats, error) {
	log.Printf("Чтение секрета: %s", m.config.SourcePath)

	secret, err := m.sourceClient.ReadSecret(m.config.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения секрета: %v", err)
	}

	atomic.AddInt64(&stats.SecretsRead, 1)

	// Проверяем существование в destination
	destPath := m.transformPath(m.config.SourcePath, m.config.DestinationPath)

	exists, err := m.destClient.SecretExists(destPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка проверки существования секрета: %v", err)
	}

	if exists && !m.config.Overwrite {
		log.Printf("Секрет уже существует в destination: %s (используйте --overwrite)", destPath)
		atomic.AddInt64(&stats.SecretsSkipped, 1)
		return stats, nil
	}

	if m.config.DryRun {
		log.Printf("[DRY-RUN] Будет записан секрет: %s", destPath)
		atomic.AddInt64(&stats.SecretsWritten, 1)
		return stats, nil
	}

	// Записываем секрет
	log.Printf("Запись секрета: %s", destPath)
	err = m.destClient.WriteSecret(destPath, secret.Data)
	if err != nil {
		atomic.AddInt64(&stats.Errors, 1)
		return nil, fmt.Errorf("ошибка записи секрета: %v", err)
	}

	atomic.AddInt64(&stats.SecretsWritten, 1)

	return stats, nil
}

func (m *SyncManager) syncDirectory(ctx context.Context, stats *SyncStats) (*SyncStats, error) {
	log.Printf("Чтение директории: %s", m.config.SourcePath)

	// Создаем каналы для параллельной обработки
	secretsChan := make(chan *vault.Secret, m.config.ParallelWorkers*2)
	errChan := make(chan error, m.config.ParallelWorkers)

	var wg sync.WaitGroup

	// Запускаем читателей
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(secretsChan)

		sourceSecrets, sourceErrChan := m.sourceClient.GetAllSecrets(ctx, m.config.SourcePath)

		for {
			select {
			case secret, ok := <-sourceSecrets:
				if !ok {
					return
				}
				atomic.AddInt64(&stats.SecretsRead, 1)
				secretsChan <- secret
			case err := <-sourceErrChan:
				if err != nil {
					errChan <- err
				}
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	// Запускаем писателей
	writerWg := sync.WaitGroup{}
	for i := 0; i < m.config.ParallelWorkers; i++ {
		writerWg.Add(1)
		go func(workerID int) {
			defer writerWg.Done()
			m.writeWorker(ctx, workerID, secretsChan, errChan, stats)
		}(i)
	}

	// Ждем завершения всех горутин
	go func() {
		wg.Wait()
		writerWg.Wait()
		close(errChan)
	}()

	// Обрабатываем ошибки
	for err := range errChan {
		atomic.AddInt64(&stats.Errors, 1)
		log.Printf("Ошибка: %v", err)
	}

	return stats, nil
}

func (m *SyncManager) writeWorker(ctx context.Context, workerID int,
	secretsChan <-chan *vault.Secret, errChan chan<- error, stats *SyncStats) {

	for secret := range secretsChan {
		select {
		case <-ctx.Done():
			return
		default:
		}

		destPath := m.transformPath(secret.Path, m.config.DestinationPath)

		// Проверяем существование
		exists, err := m.destClient.SecretExists(destPath)
		if err != nil {
			errChan <- fmt.Errorf("worker %d: ошибка проверки %s: %v", workerID, destPath, err)
			continue
		}

		if exists && !m.config.Overwrite {
			log.Printf("Worker %d: пропуск существующего секрета: %s", workerID, destPath)
			atomic.AddInt64(&stats.SecretsSkipped, 1)
			continue
		}

		if m.config.DryRun {
			log.Printf("[DRY-RUN] Worker %d: будет записан %s", workerID, destPath)
			atomic.AddInt64(&stats.SecretsWritten, 1)
			continue
		}

		// Записываем секрет
		err = m.destClient.WriteSecret(destPath, secret.Data)
		if err != nil {
			errChan <- fmt.Errorf("worker %d: ошибка записи %s: %v", workerID, destPath, err)
			continue
		}

		log.Printf("Worker %d: записан секрет: %s", workerID, destPath)
		atomic.AddInt64(&stats.SecretsWritten, 1)
	}
}

func (m *SyncManager) transformPath(sourcePath, baseDestPath string) string {
	// Убираем префикс source path из пути
	relativePath := strings.TrimPrefix(sourcePath, m.config.SourcePath)
	if strings.HasPrefix(relativePath, "/") {
		relativePath = relativePath[1:]
	}

	// Если destination path заканчивается на /data/, это значит что он указывает на конкретный движок
	if strings.Contains(baseDestPath, "/data/") {
		return baseDestPath + "/" + relativePath
	}

	// Иначе предполагаем что это путь внутри движка secret
	return "secret/data/" + baseDestPath + "/" + relativePath
}
