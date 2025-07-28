package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/golang/snappy"
)

const (
	defaultNumHosts        = 10
	defaultIntervalSeconds = 300
	defaultPrometheusURL   = "http://localhost:9090/api/v1/write"
	defaultPrometheusUser  = "admin"
	defaultPrometheusPass  = "password"
)

var (
	// エンジニアの名前（ホスト名生成用）
	firstNames = []string{
		"takeshi", "yuki", "hiroshi", "akiko", "kenji", "mai", "satoshi", "emi",
		"daisuke", "nana", "ryota", "miki", "kazuki", "aya", "tomoya", "rina",
		"shota", "yui", "kenta", "saki",
	}
	lastNames = []string{
		"tanaka", "suzuki", "watanabe", "ito", "yamamoto", "nakamura", "kobayashi",
		"kato", "yoshida", "yamada", "sasaki", "yamaguchi", "matsumoto", "inoue",
		"kimura", "hayashi", "shimizu", "yamazaki", "mori", "abe",
	}
	hostPatterns = []string{"%s-macbook", "%s-mac", "%s-%s", "%ss-mac"}
	
	// ローカルランダムジェネレータ
	rng *rand.Rand
)

// Config はデモの設定を保持
type Config struct {
	NumHosts      int
	Interval      time.Duration
	PrometheusURL string
	Username      string
	Password      string
}

// loadConfig は環境変数から設定を読み込む
func loadConfig() *Config {
	cfg := &Config{
		NumHosts:      defaultNumHosts,
		Interval:      time.Duration(defaultIntervalSeconds) * time.Second,
		PrometheusURL: defaultPrometheusURL,
		Username:      defaultPrometheusUser,
		Password:      defaultPrometheusPass,
	}

	if url := os.Getenv("TOSAGE_PROMETHEUS_REMOTE_WRITE_URL"); url != "" {
		cfg.PrometheusURL = url
	}
	if user := os.Getenv("TOSAGE_PROMETHEUS_USERNAME"); user != "" {
		cfg.Username = user
	}
	if pass := os.Getenv("TOSAGE_PROMETHEUS_PASSWORD"); pass != "" {
		cfg.Password = pass
	}

	return cfg
}

func generateHostname(rng *rand.Rand) string {
	firstName := firstNames[rng.Intn(len(firstNames))]
	lastName := lastNames[rng.Intn(len(lastNames))]
	pattern := hostPatterns[rng.Intn(len(hostPatterns))]

	switch pattern {
	case "%s-%s":
		return fmt.Sprintf(pattern, firstName, lastName)
	case "%ss-mac":
		return fmt.Sprintf(pattern, firstName)
	default:
		return fmt.Sprintf(pattern, firstName)
	}
}

// generateTokenCount はランダムなトークン数を生成（累積的に増加）
func generateTokenCount(hostID, iteration int, rng *rand.Rand) int {
	baseTokens := 1000 + (hostID * 100) + rng.Intn(500)
	timeIncrease := iteration * (10000 + rng.Intn(40001))
	randomVariation := rng.Intn(200)

	total := baseTokens + timeIncrease + randomVariation

	return total
}

// writeRawVarint writes a raw varint value
func writeRawVarint(buf *bytes.Buffer, v uint64) {
	for v >= 0x80 {
		buf.WriteByte(byte(v) | 0x80)
		v >>= 7
	}
	buf.WriteByte(byte(v))
}

// writeFieldWithData writes a field number and wire type followed by length-delimited data
func writeFieldWithData(buf *bytes.Buffer, fieldNum int, wireType int, data []byte) {
	key := (fieldNum << 3) | wireType
	writeRawVarint(buf, uint64(key))
	writeRawVarint(buf, uint64(len(data)))
	buf.Write(data)
}

// writeString writes a string field
func writeString(buf *bytes.Buffer, fieldNum int, s string) {
	key := (fieldNum << 3) | 2 // wire type 2 for string
	writeRawVarint(buf, uint64(key))
	writeRawVarint(buf, uint64(len(s)))
	buf.WriteString(s)
}

// writeFixed64 writes a fixed64 field
func writeFixed64(buf *bytes.Buffer, fieldNum int, v uint64) {
	key := (fieldNum << 3) | 1 // wire type 1 for fixed64
	writeRawVarint(buf, uint64(key))
	_ = binary.Write(buf, binary.LittleEndian, v)
}

// writeVarint writes a varint field
func writeVarint(buf *bytes.Buffer, fieldNum int, v int64) {
	key := fieldNum << 3 // wire type 0 for varint
	writeRawVarint(buf, uint64(key))
	writeRawVarint(buf, uint64(v))
}

// encodeLabel encodes a single Label
func encodeLabel(name, value string) []byte {
	var buf bytes.Buffer
	writeString(&buf, 1, name)
	writeString(&buf, 2, value)
	return buf.Bytes()
}

// encodeSample encodes a single Sample
func encodeSample(value float64, timestamp int64) []byte {
	var buf bytes.Buffer
	writeFixed64(&buf, 1, math.Float64bits(value))
	writeVarint(&buf, 2, timestamp)
	return buf.Bytes()
}

// encodeTimeSeries encodes a single TimeSeries
func encodeTimeSeries(labels map[string]string, value float64, timestamp int64) []byte {
	var buf bytes.Buffer

	// Field 1: labels (repeated)
	for name, val := range labels {
		labelData := encodeLabel(name, val)
		writeFieldWithData(&buf, 1, 2, labelData)
	}

	// Field 2: samples (repeated)
	sampleData := encodeSample(value, timestamp)
	writeFieldWithData(&buf, 2, 2, sampleData)

	return buf.Bytes()
}

// encodeWriteRequest manually encodes a WriteRequest into protobuf format
func encodeWriteRequest(metricName string, value float64, labels map[string]string, timestamp int64) ([]byte, error) {
	var buf bytes.Buffer

	// Create labels including __name__
	allLabels := make(map[string]string)
	allLabels["__name__"] = metricName
	for k, v := range labels {
		allLabels[k] = v
	}

	// Field 1: timeseries (repeated)
	timeseriesData := encodeTimeSeries(allLabels, value, timestamp)
	writeFieldWithData(&buf, 1, 2, timeseriesData)

	return buf.Bytes(), nil
}

// sendMetric はメトリクスをPrometheus Remote Write APIに送信
func sendMetric(ctx context.Context, cfg *Config, hostname, metricName string, value float64, timestamp int64) error {
	// ラベルを作成
	labels := map[string]string{
		"host": hostname,
		"demo": "true",
	}

	// Protobufメッセージをエンコード
	data, err := encodeWriteRequest(metricName, value, labels, timestamp)
	if err != nil {
		return fmt.Errorf("failed to encode write request: %w", err)
	}

	// Snappy圧縮
	compressed := snappy.Encode(nil, data)

	// HTTPリクエストを作成
	req, err := http.NewRequestWithContext(ctx, "POST", cfg.PrometheusURL, bytes.NewReader(compressed))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// ヘッダーを設定
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

	// Basic認証
	auth := base64.StdEncoding.EncodeToString([]byte(cfg.Username + ":" + cfg.Password))
	req.Header.Set("Authorization", "Basic "+auth)

	// リクエストを送信
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("[WARN] failed to close response body: %v", err)
		}
	}()

	// レスポンスを確認
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("remote write failed with status %d", resp.StatusCode)
	}

	return nil
}

// sendMetricsBatch は1バッチ分のメトリクスを送信
func sendMetricsBatch(ctx context.Context, cfg *Config, hostnames []string, iteration int, rng *rand.Rand) {
	timestamp := time.Now().UnixMilli()
	log.Printf("[INFO] バッチ %d: メトリクス送信開始 (%s)", iteration, time.Now().Format(time.RFC3339))

	var wg sync.WaitGroup
	successCount := 0
	failCount := 0
	var mu sync.Mutex

	// 並列度を制限
	sem := make(chan struct{}, 10)

	for i := 1; i <= cfg.NumHosts; i++ {
		wg.Add(1)
		go func(hostID int) {
			defer wg.Done()
			sem <- struct{}{} // 並列度制限
			defer func() { <-sem }()

			hostname := hostnames[hostID-1]
			ccTokens := float64(generateTokenCount(hostID, iteration, rng))
			cursorTokens := float64(generateTokenCount(hostID+1000, iteration, rng))

			// Claude Codeトークンを送信
			if err := sendMetric(ctx, cfg, hostname, "tosage_cc_token", ccTokens, timestamp); err != nil {
				log.Printf("[WARN] ホスト %s のClaude Codeメトリクス送信失敗: %v", hostname, err)
				mu.Lock()
				failCount++
				mu.Unlock()
			} else {
				log.Printf("[DEBUG] ホスト %s のメトリクス送信成功: tosage_cc_token=%.0f", hostname, ccTokens)
				mu.Lock()
				successCount++
				mu.Unlock()
			}

			// Cursorトークンを送信
			if err := sendMetric(ctx, cfg, hostname, "tosage_cursor_token", cursorTokens, timestamp); err != nil {
				log.Printf("[WARN] ホスト %s のCursorメトリクス送信失敗: %v", hostname, err)
				mu.Lock()
				failCount++
				mu.Unlock()
			} else {
				log.Printf("[DEBUG] ホスト %s のメトリクス送信成功: tosage_cursor_token=%.0f", hostname, cursorTokens)
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	log.Printf("[INFO] バッチ %d: 送信完了 - 成功: %d, 失敗: %d", iteration, successCount, failCount)
}

func main() {
	log.SetFlags(log.Ltime)
	log.Println("[INFO] Tosage デモメトリクス送信プログラム開始 (Go版)")

	// 設定を読み込み
	cfg := loadConfig()
	log.Printf("[INFO] 設定確認完了")
	log.Printf("[INFO] Prometheus URL: %s", cfg.PrometheusURL)
	log.Printf("[INFO] ホスト数: %d", cfg.NumHosts)
	log.Printf("[INFO] 送信間隔: %v", cfg.Interval)

	// コンテキストとシグナルハンドリング
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("[INFO] デモプログラムを停止中...")
		cancel()
	}()

	// 乱数生成器を初期化
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// ホスト名を事前に生成
	hostnames := make([]string, cfg.NumHosts)
	for i := 0; i < cfg.NumHosts; i++ {
		hostnames[i] = generateHostname(rng)
	}
	log.Printf("[INFO] %d台分のホスト名を生成しました", cfg.NumHosts)

	log.Println("[INFO] デモメトリクス送信を開始します...")
	log.Println("[INFO] Ctrl+C で停止できます")

	iteration := 1
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	// 最初のバッチをすぐに送信
	sendMetricsBatch(ctx, cfg, hostnames, iteration, rng)
	iteration++

	// 定期的にメトリクスを送信
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sendMetricsBatch(ctx, cfg, hostnames, iteration, rng)
			iteration++
			log.Printf("[INFO] 次の送信まで %v 待機中...", cfg.Interval)
		}
	}
}
