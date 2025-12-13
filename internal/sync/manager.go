package sync

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"vault-copy/internal/config"
	"vault-copy/internal/logger"
	"vault-copy/internal/vault"
)

type SyncStats struct {
	SecretsRead    int64
	SecretsWritten int64
	SecretsSkipped int64
	Errors         int64
}

type SyncManager struct {
	sourceClient vault.ClientInterface
	destClient   vault.ClientInterface
	config       *config.Config
	logger       *logger.Logger
}

func NewManager(sourceClient, destClient vault.ClientInterface, cfg *config.Config) *SyncManager {
	return &SyncManager{
		sourceClient: sourceClient,
		destClient:   destClient,
		config:       cfg,
		logger:       logger.NewLogger(cfg),
	}
}

func (m *SyncManager) Sync(ctx context.Context) (*SyncStats, error) {
	stats := &SyncStats{}

	m.logger.Info("Начинаем синхронизацию из %s в %s",
		m.config.SourcePath, m.config.DestinationPath)

	if m.config.DryRun {
		m.logger.Info("Режим dry-run - секреты не будут записаны")
	}

	// Подробный вывод при включенном verbose режиме
	m.logger.Verbose("Конфигурация синхронизации:")
	m.logger.Verbose("  Источник: %s", m.config.SourcePath)
	m.logger.Verbose("  Назначение: %s", m.config.DestinationPath)
	m.logger.Verbose("  Рекурсивно: %t", m.config.Recursive)
	m.logger.Verbose("  Dry-run: %t", m.config.DryRun)
	m.logger.Verbose("  Перезапись: %t", m.config.Overwrite)
	m.logger.Verbose("  Параллельные операции: %d", m.config.ParallelWorkers)
	m.logger.Verbose("  Vault-источник: %s", m.config.SourceAddr)
	m.logger.Verbose("  Vault-приемник: %s", m.config.DestAddr)

	// Проверяем, является ли источник директорией
	m.logger.Verbose("Проверка, является ли источник директорией: %s", m.config.SourcePath)
	isDir, err := m.sourceClient.IsDirectory(m.config.SourcePath, m.logger)
	if err != nil {
		return nil, fmt.Errorf("ошибка проверки пути источника: %v", err)
	}
	m.logger.Verbose("Источник %s директория: %t", m.config.SourcePath, isDir)

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
	m.logger.Info("Чтение секрета: %s", m.config.SourcePath)
	m.logger.Verbose("Подключение к Vault-источнику: %s", m.config.SourceAddr)

	secret, err := m.sourceClient.ReadSecret(m.config.SourcePath, m.logger)
	if err != nil {
		m.logger.Error("Ошибка чтения секрета %s: %v", m.config.SourcePath, err)
		return nil, fmt.Errorf("ошибка чтения секрета: %v", err)
	}

	atomic.AddInt64(&stats.SecretsRead, 1)
	m.logger.Verbose("Успешно прочитан секрет: %s", m.config.SourcePath)

	// Проверяем существование в destination
	destPath := m.transformPath(m.config.SourcePath, m.config.DestinationPath)
	m.logger.Verbose("Проверка существования секрета в destination: %s", destPath)

	exists, err := m.destClient.SecretExists(destPath, m.logger)
	if err != nil {
		m.logger.Error("Ошибка проверки существования секрета %s: %v", destPath, err)
		return nil, fmt.Errorf("ошибка проверки существования секрета: %v", err)
	}

	if exists && !m.config.Overwrite {
		m.logger.Info("Секрет уже существует в destination: %s (используйте --overwrite)", destPath)
		atomic.AddInt64(&stats.SecretsSkipped, 1)
		return stats, nil
	}

	if m.config.DryRun {
		m.logger.Info("[DRY-RUN] Будет записан секрет: %s", destPath)
		atomic.AddInt64(&stats.SecretsWritten, 1)
		return stats, nil
	}

	// Записываем секрет
	m.logger.Info("Запись секрета: %s", destPath)
	m.logger.Verbose("Подключение к Vault-приемнику: %s", m.config.DestAddr)
	err = m.destClient.WriteSecret(destPath, secret.Data, m.logger)
	if err != nil {
		m.logger.Error("Ошибка записи секрета %s: %v", destPath, err)
		atomic.AddInt64(&stats.Errors, 1)
		return nil, fmt.Errorf("ошибка записи секрета: %v", err)
	}

	m.logger.Verbose("Успешно записан секрет: %s", destPath)
	atomic.AddInt64(&stats.SecretsWritten, 1)

	return stats, nil
}

func (m *SyncManager) syncDirectory(ctx context.Context, stats *SyncStats) (*SyncStats, error) {
	m.logger.Info("Чтение директории: %s", m.config.SourcePath)
	m.logger.Verbose("Подключение к Vault-источнику: %s", m.config.SourceAddr)

	// Создаем каналы для параллельной обработки
	secretsChan := make(chan *vault.Secret, m.config.ParallelWorkers*2)
	errChan := make(chan error, m.config.ParallelWorkers)

	var wg sync.WaitGroup

	// Запускаем читателей
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(secretsChan)

		m.logger.Verbose("Получение списка всех секретов из: %s", m.config.SourcePath)
		sourceSecrets, sourceErrChan := m.sourceClient.GetAllSecrets(ctx, m.config.SourcePath, m.logger)

		for {
			select {
			case secret, ok := <-sourceSecrets:
				if !ok {
					m.logger.Verbose("Завершено чтение секретов из: %s", m.config.SourcePath)
					return
				}
				atomic.AddInt64(&stats.SecretsRead, 1)
				m.logger.Verbose("Прочитан секрет: %s", secret.Path)
				secretsChan <- secret
			case err := <-sourceErrChan:
				if err != nil {
					m.logger.Error("Ошибка при получении списка секретов: %v", err)
					errChan <- err
				}
				return
			case <-ctx.Done():
				m.logger.Verbose("Контекст отменен при чтении секретов")
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
		m.logger.Verbose("Завершено чтение всех секретов")
		writerWg.Wait()
		m.logger.Verbose("Завершена запись всех секретов")
		close(errChan)
	}()

	// Обрабатываем ошибки
	for err := range errChan {
		atomic.AddInt64(&stats.Errors, 1)
		m.logger.Error("Ошибка: %v", err)
	}

	return stats, nil
}

func (m *SyncManager) writeWorker(ctx context.Context, workerID int,
	secretsChan <-chan *vault.Secret, errChan chan<- error, stats *SyncStats) {

	m.logger.Verbose("Worker %d: запущен", workerID)

	for secret := range secretsChan {
		select {
		case <-ctx.Done():
			m.logger.Verbose("Worker %d: контекст отменен", workerID)
			return
		default:
		}

		destPath := m.transformPath(secret.Path, m.config.DestinationPath)
		m.logger.Verbose("Worker %d: обработка секрета %s -> %s", workerID, secret.Path, destPath)

		// Проверяем существование
		m.logger.Verbose("Worker %d: проверка существования секрета: %s", workerID, destPath)
		exists, err := m.destClient.SecretExists(destPath, m.logger)
		if err != nil {
			m.logger.Error("Worker %d: ошибка проверки %s: %v", workerID, destPath, err)
			errChan <- fmt.Errorf("worker %d: ошибка проверки %s: %v", workerID, destPath, err)
			continue
		}

		if exists && !m.config.Overwrite {
			m.logger.Info("Worker %d: пропуск существующего секрета: %s", workerID, destPath)
			atomic.AddInt64(&stats.SecretsSkipped, 1)
			continue
		}

		if m.config.DryRun {
			m.logger.Info("[DRY-RUN] Worker %d: будет записан %s", workerID, destPath)
			atomic.AddInt64(&stats.SecretsWritten, 1)
			continue
		}

		// Записываем секрет
		m.logger.Verbose("Worker %d: запись секрета: %s", workerID, destPath)
		m.logger.Verbose("Worker %d: подключение к Vault-приемнику: %s", workerID, m.config.DestAddr)
		err = m.destClient.WriteSecret(destPath, secret.Data, m.logger)
		if err != nil {
			m.logger.Error("Worker %d: ошибка записи %s: %v", workerID, destPath, err)
			errChan <- fmt.Errorf("worker %d: ошибка записи %s: %v", workerID, destPath, err)
			continue
		}

		m.logger.Info("Worker %d: записан секрет: %s", workerID, destPath)
		m.logger.Verbose("Worker %d: успешно записан секрет: %s", workerID, destPath)
		atomic.AddInt64(&stats.SecretsWritten, 1)
	}

	m.logger.Verbose("Worker %d: завершен", workerID)
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
