package repository

import (
	"encoding/json"
	"os"
	"sort" // Для сортировки метрик перед сравнением
	"testing"
	"time"

	"github.com/makgig/factory/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Тест для функции New
func TestNew(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "create new storage",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Приводим результат New к *MemStorage, чтобы получить доступ к внутренним полям и методам
			m := New("").(*MemStorage)

			require.NotNil(t, m, "New() should not return nil")
			require.NotNil(t, m.GetAllGauges(), "gauges map should not be nil")
			require.NotNil(t, m.GetAllCounters(), "counters map should not be nil")
			assert.Empty(t, m.GetAllGauges(), "gauges map should be empty")
			assert.Empty(t, m.GetAllCounters(), "counters map should be empty")
			// Теперь можно проверить stopChan, так как m имеет тип *MemStorage
			require.NotNil(t, m.stopChan, "stopChan should be initialized")
		})
	}
}

// Тест для UpdateGauge, включая проверку синхронного сохранения
func TestMemStorage_UpdateGauge(t *testing.T) {
	type fields struct {
		gauges   map[string]float64
		counters map[string]int64
		filePath string
		syncSave bool
	}
	type args struct {
		name  string
		value float64
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		expectedGauges map[string]float64
		expectFileSave bool // Ожидаем ли сохранение в файл
	}{
		{
			name: "add new gauge metric (no file save)",
			fields: fields{
				gauges:   map[string]float64{},
				counters: map[string]int64{},
				filePath: "", // Не настроен файл, нет сохранения
				syncSave: false,
			},
			args: args{
				name:  "temperature",
				value: 23.5,
			},
			expectedGauges: map[string]float64{"temperature": 23.5},
			expectFileSave: false,
		},
		{
			name: "update existing gauge metric (no file save)",
			fields: fields{
				gauges:   map[string]float64{"temperature": 20.0},
				counters: map[string]int64{},
				filePath: "",
				syncSave: false,
			},
			args: args{
				name:  "temperature",
				value: 25.0,
			},
			expectedGauges: map[string]float64{"temperature": 25.0},
			expectFileSave: false,
		},
		{
			name: "add gauge with synchronous file save",
			fields: fields{
				gauges:   map[string]float64{},
				counters: map[string]int64{},
				filePath: "test_gauge_sync_save.json", // Будет временный файл
				syncSave: true,
			},
			args: args{
				name:  "humidity",
				value: 60.1,
			},
			expectedGauges: map[string]float64{"humidity": 60.1},
			expectFileSave: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем временный файл, если ожидается сохранение
			var tmpFilePath string
			if tt.expectFileSave {
				tmpFile, err := os.CreateTemp("", tt.fields.filePath)
				require.NoError(t, err)
				tmpFilePath = tmpFile.Name()
				tmpFile.Close()                              // Закрываем файл, SaveToFile откроет его снова
				t.Cleanup(func() { os.Remove(tmpFilePath) }) // Удаляем после теста
			} else {
				// Если не ожидаем сохранения в файл, убедимся, что filePath пуст
				tmpFilePath = ""
			}

			// Приводим результат New к *MemStorage
			m := New(tmpFilePath).(*MemStorage)
			m.gauges = tt.fields.gauges
			m.counters = tt.fields.counters
			m.SetStoreConfig(0, tt.fields.syncSave) // storeInterval = 0 для синхронного сохранения

			m.UpdateGauge(tt.args.name, tt.args.value)

			assert.Equal(t, tt.expectedGauges, m.gauges, "gauges should match expected values")

			if tt.expectFileSave {
				// Проверяем, что файл был создан и содержит корректные данные
				data, err := os.ReadFile(tmpFilePath)
				require.NoError(t, err, "should read file without error")

				var savedMetrics []models.Metrics
				err = json.Unmarshal(data, &savedMetrics)
				require.NoError(t, err, "should unmarshal saved metrics")

				// Проверяем наличие нашей метрики
				found := false
				for _, metric := range savedMetrics {
					if metric.ID == tt.args.name && metric.MType == "gauge" && metric.Value != nil {
						assert.InDelta(t, tt.args.value, *metric.Value, 0.001, "saved gauge value mismatch")
						found = true
						break
					}
				}
				assert.True(t, found, "saved file should contain the updated gauge metric")
			}
		})
	}
}

// Тест для UpdateCounter, включая проверку синхронного сохранения
func TestMemStorage_UpdateCounter(t *testing.T) {
	type fields struct {
		gauges   map[string]float64
		counters map[string]int64
		filePath string
		syncSave bool
	}
	type args struct {
		name  string
		value int64
	}
	tests := []struct {
		name             string
		fields           fields
		args             args
		expectedCounters map[string]int64
		expectFileSave   bool
	}{
		{
			name: "add new counter metric (no file save)",
			fields: fields{
				gauges:   map[string]float64{},
				counters: map[string]int64{},
				filePath: "",
				syncSave: false,
			},
			args: args{
				name:  "requests",
				value: 10,
			},
			expectedCounters: map[string]int64{"requests": 10},
			expectFileSave:   false,
		},
		{
			name: "update existing counter metric (no file save)",
			fields: fields{
				gauges:   map[string]float64{},
				counters: map[string]int64{"requests": 100},
				filePath: "",
				syncSave: false,
			},
			args: args{
				name:  "requests",
				value: 5,
			},
			expectedCounters: map[string]int64{"requests": 105},
			expectFileSave:   false,
		},
		{
			name: "add counter with synchronous file save",
			fields: fields{
				gauges:   map[string]float64{},
				counters: map[string]int64{},
				filePath: "test_counter_sync_save.json",
				syncSave: true,
			},
			args: args{
				name:  "errors",
				value: 1,
			},
			expectedCounters: map[string]int64{"errors": 1},
			expectFileSave:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tmpFilePath string
			if tt.expectFileSave {
				tmpFile, err := os.CreateTemp("", tt.fields.filePath)
				require.NoError(t, err)
				tmpFilePath = tmpFile.Name()
				tmpFile.Close()
				t.Cleanup(func() { os.Remove(tmpFilePath) })
			} else {
				tmpFilePath = ""
			}

			// Приводим результат New к *MemStorage
			m := New(tmpFilePath).(*MemStorage)
			m.gauges = tt.fields.gauges
			m.counters = tt.fields.counters
			m.SetStoreConfig(0, tt.fields.syncSave)

			m.UpdateCounter(tt.args.name, tt.args.value)

			assert.Equal(t, tt.expectedCounters, m.counters, "counters should match expected values")

			if tt.expectFileSave {
				data, err := os.ReadFile(tmpFilePath)
				require.NoError(t, err, "should read file without error")

				var savedMetrics []models.Metrics
				err = json.Unmarshal(data, &savedMetrics)
				require.NoError(t, err, "should unmarshal saved metrics")

				found := false
				for _, metric := range savedMetrics {
					if metric.ID == tt.args.name && metric.MType == "counter" && metric.Delta != nil {
						assert.Equal(t, tt.args.value, *metric.Delta, "saved counter delta mismatch")
						found = true
						break
					}
				}
				assert.True(t, found, "saved file should contain the updated counter metric")
			}
		})
	}
}

// Тест для GetGauge
func TestMemStorage_GetGauge(t *testing.T) {
	type fields struct {
		gauges   map[string]float64
		counters map[string]int64
	}
	type args struct {
		name string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   float64
		want1  bool
	}{
		{
			name: "get existing gauge",
			fields: fields{
				gauges:   map[string]float64{"temperature": 23.5},
				counters: map[string]int64{},
			},
			args: args{
				name: "temperature",
			},
			want:  23.5,
			want1: true,
		},
		{
			name: "get non-existing gauge",
			fields: fields{
				gauges:   map[string]float64{"temperature": 23.5},
				counters: map[string]int64{},
			},
			args: args{
				name: "pressure",
			},
			want:  0.0,
			want1: false,
		},
		{
			name: "get from empty storage",
			fields: fields{
				gauges:   map[string]float64{},
				counters: map[string]int64{},
			},
			args: args{
				name: "temperature",
			},
			want:  0.0,
			want1: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Приводим результат New к *MemStorage
			m := New("").(*MemStorage)
			m.gauges = tt.fields.gauges
			m.counters = tt.fields.counters

			got, got1 := m.GetGauge(tt.args.name)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.want1, got1) // Исправлено: Сравниваем с got1
		})
	}
}

// Тест для GetCounter
func TestMemStorage_GetCounter(t *testing.T) {
	type fields struct {
		gauges   map[string]float64
		counters map[string]int64
	}
	type args struct {
		name string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int64
		want1  bool
	}{
		{
			name: "get existing counter",
			fields: fields{
				gauges:   map[string]float64{},
				counters: map[string]int64{"requests": 100},
			},
			args: args{
				name: "requests",
			},
			want:  100,
			want1: true,
		},
		{
			name: "get non-existing counter",
			fields: fields{
				gauges:   map[string]float64{},
				counters: map[string]int64{"requests": 100},
			},
			args: args{
				name: "errors",
			},
			want:  0,
			want1: false,
		},
		{
			name: "get from empty storage",
			fields: fields{
				gauges:   map[string]float64{},
				counters: map[string]int64{},
			},
			args: args{
				name: "requests",
			},
			want:  0,
			want1: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Приводим результат New к *MemStorage
			m := New("").(*MemStorage)
			m.gauges = tt.fields.gauges
			m.counters = tt.fields.counters
			got, got1 := m.GetCounter(tt.args.name)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.want1, got1)
		})
	}
}

// Тест для GetAllGauges
func TestMemStorage_GetAllGauges(t *testing.T) {
	type fields struct {
		gauges   map[string]float64
		counters map[string]int64
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]float64
	}{
		{
			name: "get all from populated storage",
			fields: fields{
				gauges:   map[string]float64{"temperature": 23.5, "cpu_usage": 75.5},
				counters: map[string]int64{},
			},
			want: map[string]float64{"temperature": 23.5, "cpu_usage": 75.5},
		},
		{
			name: "get all from empty storage",
			fields: fields{
				gauges:   map[string]float64{},
				counters: map[string]int64{},
			},
			want: map[string]float64{},
		},
		{
			name: "get all with single gauge",
			fields: fields{
				gauges:   map[string]float64{"temperature": 23.5},
				counters: map[string]int64{},
			},
			want: map[string]float64{"temperature": 23.5},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Приводим результат New к *MemStorage
			m := New("").(*MemStorage)
			m.gauges = tt.fields.gauges
			m.counters = tt.fields.counters
			got := m.GetAllGauges()
			assert.Equal(t, tt.want, got)
		})
	}
}

// Тест для GetAllCounters
func TestMemStorage_GetAllCounters(t *testing.T) {
	type fields struct {
		gauges   map[string]float64
		counters map[string]int64
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]int64
	}{
		{
			name: "get all from populated storage",
			fields: fields{
				gauges:   map[string]float64{},
				counters: map[string]int64{"requests": 100, "errors": 5},
			},
			want: map[string]int64{"requests": 100, "errors": 5},
		},
		{
			name: "get all from empty storage",
			fields: fields{
				gauges:   map[string]float64{},
				counters: map[string]int64{},
			},
			want: map[string]int64{},
		},
		{
			name: "get all with single counter",
			fields: fields{
				gauges:   map[string]float64{},
				counters: map[string]int64{"requests": 100},
			},
			want: map[string]int64{"requests": 100},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Приводим результат New к *MemStorage
			m := New("").(*MemStorage)
			m.gauges = tt.fields.gauges
			m.counters = tt.fields.counters
			got := m.GetAllCounters()
			assert.Equal(t, tt.want, got)
		})
	}
}

// Новый тест для SaveToFile
func TestMemStorage_SaveToFile(t *testing.T) {
	tests := []struct {
		name       string
		gauges     map[string]float64
		counters   map[string]int64
		filePath   string
		expectErr  bool
		expectedFn func(t *testing.T, filePath string) // Функция для проверки содержимого файла
	}{
		{
			name:      "save empty storage to file",
			gauges:    map[string]float64{},
			counters:  map[string]int64{},
			filePath:  "empty_storage.json",
			expectErr: false,
			expectedFn: func(t *testing.T, filePath string) {
				data, err := os.ReadFile(filePath)
				require.NoError(t, err)
				var metrics []models.Metrics
				require.NoError(t, json.Unmarshal(data, &metrics))
				assert.Empty(t, metrics, "file should contain no metrics")
			},
		},
		{
			name:      "save populated storage to file",
			gauges:    map[string]float64{"g1": 1.1},
			counters:  map[string]int64{"c1": 100},
			filePath:  "populated_storage.json",
			expectErr: false,
			expectedFn: func(t *testing.T, filePath string) {
				data, err := os.ReadFile(filePath)
				require.NoError(t, err)
				var metrics []models.Metrics
				require.NoError(t, json.Unmarshal(data, &metrics))
				assert.Len(t, metrics, 2, "file should contain 2 metrics")

				// Проверяем метрики
				gaugeFound := false
				counterFound := false
				for _, m := range metrics {
					if m.ID == "g1" && m.MType == "gauge" && m.Value != nil && *m.Value == 1.1 {
						gaugeFound = true
					}
					if m.ID == "c1" && m.MType == "counter" && m.Delta != nil && *m.Delta == 100 {
						counterFound = true
					}
				}
				assert.True(t, gaugeFound, "gauge g1 should be saved")
				assert.True(t, counterFound, "counter c1 should be saved")
			},
		},
		{
			name:      "save with no file path set",
			gauges:    map[string]float64{"g1": 1.1},
			counters:  map[string]int64{},
			filePath:  "",    // Нет пути к файлу
			expectErr: false, // SaveToFile должен просто вернуть nil
			expectedFn: func(t *testing.T, filePath string) {
				// Файл не должен быть создан или изменен
				_, err := os.Stat(filePath)
				assert.True(t, os.IsNotExist(err) || filePath == "", "file should not exist if path is empty")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tmpFilePath string
			if tt.filePath != "" {
				tmpFile, err := os.CreateTemp("", tt.filePath)
				require.NoError(t, err)
				tmpFilePath = tmpFile.Name()
				tmpFile.Close()
				t.Cleanup(func() { os.Remove(tmpFilePath) })
			} else {
				// Если filePath пуст, то tmpFilePath тоже будет пустым,
				// и SaveToFile не будет пытаться что-то сохранить.
				tmpFilePath = ""
			}

			// Приводим результат New к *MemStorage
			m := New(tmpFilePath).(*MemStorage)
			m.gauges = tt.gauges
			m.counters = tt.counters

			err := m.SaveToFile()

			if tt.expectErr {
				assert.Error(t, err, "expected an error")
			} else {
				assert.NoError(t, err, "did not expect an error")
			}

			if tt.expectedFn != nil {
				tt.expectedFn(t, tmpFilePath)
			}
		})
	}
}

// Новый тест для LoadFromFile
func TestMemStorage_LoadFromFile(t *testing.T) {
	type setup struct {
		filePath string
		fileData []byte
	}
	tests := []struct {
		name             string
		setup            setup
		expectErr        bool
		expectedGauges   map[string]float64
		expectedCounters map[string]int64
	}{
		{
			name: "load from existing file with data",
			setup: setup{
				filePath: "data_file.json",
				fileData: []byte(`[
					{"id":"g1","type":"gauge","value":12.3},
					{"id":"c1","type":"counter","delta":42}
				]`),
			},
			expectErr:        false,
			expectedGauges:   map[string]float64{"g1": 12.3},
			expectedCounters: map[string]int64{"c1": 42},
		},
		{
			name: "load from empty file",
			setup: setup{
				filePath: "empty_file.json",
				fileData: []byte(`[]`),
			},
			expectErr:        false,
			expectedGauges:   map[string]float64{},
			expectedCounters: map[string]int64{},
		},
		{
			name: "load from non-existent file",
			setup: setup{
				filePath: "non_existent.json", // Файл не будет создан
				fileData: nil,
			},
			expectErr:        false, // Ожидаем nil, т.к. файл не существует
			expectedGauges:   map[string]float64{},
			expectedCounters: map[string]int64{},
		},
		{
			name: "load from file with invalid json",
			setup: setup{
				filePath: "invalid_json.json",
				fileData: []byte(`{"invalid": "json"`),
			},
			expectErr:        true,                 // Ожидаем ошибку парсинга
			expectedGauges:   map[string]float64{}, // Метрики не должны быть загружены
			expectedCounters: map[string]int64{},
		},
		{
			name: "load with no file path set",
			setup: setup{
				filePath: "", // Нет пути к файлу
				fileData: nil,
			},
			expectErr:        false, // LoadFromFile должен просто вернуть nil
			expectedGauges:   map[string]float64{},
			expectedCounters: map[string]int64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tmpFilePath string
			if tt.setup.filePath != "" {
				tmpFile, err := os.CreateTemp("", tt.setup.filePath)
				require.NoError(t, err)
				tmpFilePath = tmpFile.Name()
				t.Cleanup(func() { os.Remove(tmpFilePath) })

				if tt.setup.fileData != nil {
					_, err := tmpFile.Write(tt.setup.fileData)
					require.NoError(t, err)
				}
				tmpFile.Close()
			} else {
				tmpFilePath = ""
			}

			// Инициализируем хранилище с некоторыми данными,
			// чтобы убедиться, что LoadFromFile их очистит
			// Приводим результат New к *MemStorage
			m := New(tmpFilePath).(*MemStorage)
			m.UpdateGauge("old_gauge", 99.9)
			m.UpdateCounter("old_counter", 10)

			err := m.LoadFromFile()

			if tt.expectErr {
				assert.Error(t, err, "expected an error")
			} else {
				assert.NoError(t, err, "did not expect an error")
				assert.Equal(t, tt.expectedGauges, m.gauges, "gauges should match loaded values")
				assert.Equal(t, tt.expectedCounters, m.counters, "counters should match loaded values")
			}
		})
	}
}

// Новый тест для SetStoreConfig
func TestMemStorage_SetStoreConfig(t *testing.T) {
	// Приводим результат New к *MemStorage
	m := New("").(*MemStorage)
	assert.Equal(t, time.Duration(0), m.storeInterval)
	assert.False(t, m.syncSave)

	m.SetStoreConfig(5*time.Second, true)
	assert.Equal(t, 5*time.Second, m.storeInterval)
	assert.True(t, m.syncSave)

	m.SetStoreConfig(10*time.Minute, false)
	assert.Equal(t, 10*time.Minute, m.storeInterval)
	assert.False(t, m.syncSave)
}

// Helper function to convert map to sorted slice of models.Metrics for comparison
func convertAndSortMetrics(gauges map[string]float64, counters map[string]int64) []models.Metrics {
	var metrics []models.Metrics

	for name, value := range gauges {
		val := value // копия для указателя
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: "gauge",
			Value: &val,
		})
	}
	for name, value := range counters {
		delta := value // копия для указателя
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: "counter",
			Delta: &delta,
		})
	}

	sort.Slice(metrics, func(i, j int) bool {
		if metrics[i].ID != metrics[j].ID {
			return metrics[i].ID < metrics[j].ID
		}
		return metrics[i].MType < metrics[j].MType
	})
	return metrics
}

// Новый тест для периодического сохранения с использованием StartSavingLoop
func TestMemStorage_PeriodicSaving(t *testing.T) {
	// Создаем временный файл
	tmpFile, err := os.CreateTemp("", "periodic_save_test.json")
	require.NoError(t, err)
	tmpFilePath := tmpFile.Name()
	tmpFile.Close()                              // Закрываем, чтобы MemStorage мог его открыть
	t.Cleanup(func() { os.Remove(tmpFilePath) }) // Удаляем после теста

	saveInterval := 200 * time.Millisecond // Было 100ms
	waitGrace := 100 * time.Millisecond    // Было 50ms

	// Приводим результат New к *MemStorage
	m := New(tmpFilePath).(*MemStorage)
	m.SetStoreConfig(saveInterval, false) // Периодическое сохранение
	m.StartSavingLoop()                   // Теперь это корректно вызывается на *MemStorage
	t.Cleanup(func() {
		// Принудительное сохранение перед остановкой цикла, если оно не произошло
		if err := m.SaveToFile(); err != nil {
			t.Logf("Error saving on cleanup: %v", err)
		}
		m.StopSavingLoop() // Теперь это корректно вызывается на *MemStorage
	})

	// 1. Добавляем метрики
	m.UpdateGauge("test_gauge", 1.23)
	m.UpdateCounter("test_counter", 45)

	// 2. Ждем чуть больше интервала сохранения, чтобы гарантировать запись
	time.Sleep(saveInterval + waitGrace)

	// 3. Проверяем содержимое файла
	data, err := os.ReadFile(tmpFilePath)
	require.NoError(t, err, "should read file without error after first save")

	var savedMetrics []models.Metrics
	require.NoError(t, json.Unmarshal(data, &savedMetrics), "should unmarshal saved metrics after first save")

	expectedMetrics1 := convertAndSortMetrics(
		map[string]float64{"test_gauge": 1.23},
		map[string]int64{"test_counter": 45},
	)
	assert.ElementsMatch(t, expectedMetrics1, savedMetrics, "file content after first save mismatch")

	// 4. Обновляем метрики
	m.UpdateGauge("test_gauge", 4.56)  // Обновляем существующую
	m.UpdateCounter("test_counter", 5) // Обновляем существующую
	m.UpdateGauge("new_gauge", 7.89)   // Добавляем новую

	// 5. Ждем еще один интервал сохранения
	time.Sleep(saveInterval + waitGrace)

	// 6. Проверяем содержимое файла снова
	data, err = os.ReadFile(tmpFilePath)
	require.NoError(t, err, "should read file without error after second save")

	savedMetrics = []models.Metrics{} // Очищаем для нового unmarshal
	require.NoError(t, json.Unmarshal(data, &savedMetrics), "should unmarshal saved metrics after second save")

	expectedMetrics2 := convertAndSortMetrics(
		map[string]float64{"test_gauge": 4.56, "new_gauge": 7.89},
		map[string]int64{"test_counter": 50}, // 45 (initial) + 5 (update) = 50
	)
	assert.ElementsMatch(t, expectedMetrics2, savedMetrics, "file content after second save mismatch")

	// Проверка, что периодическое сохранение не запускается при syncSave = true
	t.Run("no periodic saving for syncSave", func(t *testing.T) {
		syncTmpFile, err := os.CreateTemp("", "sync_save_only.json")
		require.NoError(t, err)
		syncTmpFilePath := syncTmpFile.Name()
		syncTmpFile.Close()
		t.Cleanup(func() { os.Remove(syncTmpFilePath) })

		// Приводим результат New к *MemStorage
		mSync := New(syncTmpFilePath).(*MemStorage)
		mSync.SetStoreConfig(0, true)                // storeInterval = 0, syncSave = true
		mSync.StartSavingLoop()                      // Этот вызов не должен запускать горутину
		t.Cleanup(func() { mSync.StopSavingLoop() }) // на всякий случай

		// Обновляем метрику, это должно вызвать синхронное сохранение
		mSync.UpdateGauge("sync_test", 10.0)

		// Читаем файл сразу после синхронного обновления
		data, err = os.ReadFile(syncTmpFilePath)
		require.NoError(t, err)
		var syncMetrics []models.Metrics
		require.NoError(t, json.Unmarshal(data, &syncMetrics))
		assert.Len(t, syncMetrics, 1, "sync file should have 1 metric immediately")
		assert.InDelta(t, 10.0, *syncMetrics[0].Value, 0.001, "sync gauge value mismatch")
		assert.Equal(t, "sync_test", syncMetrics[0].ID)
		assert.Equal(t, "gauge", syncMetrics[0].MType)

		// Ждем, чтобы убедиться, что *дополнительного* периодического сохранения не было
		initialModTime := getFileModTime(t, syncTmpFilePath)
		time.Sleep(saveInterval + waitGrace)
		finalModTime := getFileModTime(t, syncTmpFilePath)

		// Время модификации файла не должно измениться, если не было новых обновлений.
		assert.Equal(t, initialModTime, finalModTime, "file modification time should not change for syncSave only")
	})
}

// Вспомогательная функция для получения времени модификации файла
func getFileModTime(t *testing.T, filePath string) time.Time {
	info, err := os.Stat(filePath)
	require.NoError(t, err)
	return info.ModTime()
}
