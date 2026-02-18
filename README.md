# 🎵 MusicProc

**ماژول پردازش صدا با کیفیت Production-Grade برای Go**

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

---

## 📋 فهرست مطالب

- [معرفی](#معرفی)
- [ویژگی‌ها](#ویژگی‌ها)
- [معماری](#معماری)
- [پیش‌نیازها](#پیش‌نیازها)
- [نصب](#نصب)
- [شروع سریع](#شروع-سریع)
- [راهنمای کامل API](#راهنمای-کامل-api)
- [مثال‌های کاربردی](#مثال‌های-کاربردی)
- [مدیریت خطا](#مدیریت-خطا)
- [پیکربندی پیشرفته](#پیکربندی-پیشرفته)
- [تست‌ها](#تست‌ها)
- [ملاحظات Production](#ملاحظات-production)

---

## معرفی

`musicproc` یک ماژول Go است برای پردازش فایل‌های صوتی در محیط‌های Production. این ماژول بر پایه FFmpeg ساخته شده و امکانات زیر را با معماری لایه‌ای ارائه می‌دهد:

- تبدیل codec به Opus، AAC و MP3
- Loudness normalization طبق استاندارد EBU R128
- Highpass / Lowpass filtering
- پردازش دسته‌ای (batch) با worker pool
- مدیریت کامل context cancellation و timeout
- خطاهای ساختاریافته و گزارش progress

---

## ویژگی‌ها

| ویژگی | توضیح |
|-------|-------|
| **EBU R128 Normalization** | استاندارد loudness برای پخش آنلاین (Spotify، YouTube) |
| **Multi-codec** | Opus، AAC، MP3 |
| **Batch Processing** | worker pool با concurrency قابل تنظیم |
| **Context-aware** | پشتیبانی از timeout و cancellation |
| **Functional Options** | API تمیز و قابل توسعه |
| **Structured Errors** | دسته‌بندی خطا: ProcessingError، FFmpegError، ValidationError |
| **Progress Reporting** | channel-based real-time updates |
| **Retry Mechanism** | exponential backoff با قابلیت تنظیم |
| **Thread-safe** | بدون global state |
| **Testable** | تمام وابستگی‌ها از طریق interface |

---

## پیش‌نیازها

| ابزار | نسخه |
|-------|-------|
| Go | 1.22+ |
| FFmpeg | 4.0+ |
| FFprobe | همراه FFmpeg |

### نصب FFmpeg

```bash
# Ubuntu/Debian
sudo apt-get install ffmpeg

# macOS
brew install ffmpeg

# بررسی نصب
ffmpeg -version
ffprobe -version
```

---

## نصب

```bash
go get github.com/Skryldev/audio-lab
```

---

## شروع سریع

```go
package main

import (
    "context"
    "fmt"
    "log"

    musicproc "github.com/musicproc"
)

func main() {
    // ساخت processor با تنظیمات پیش‌فرض
    processor, err := musicproc.New(musicproc.Config{})
    if err != nil {
        log.Fatal(err)
    }
    defer processor.Close()

    // پردازش یک فایل
    result, err := processor.ProcessAudio(
        context.Background(),
        "input.wav",
        "output.opus",
        musicproc.WithCodec(musicproc.CodecOpus),
        musicproc.WithBitrate(128_000),
        musicproc.WithNormalization(true),
    )
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("✅ Done! Duration: %s\n", result.Duration)
}
```

---

## راهنمای کامل API

### ساخت Processor

```go
processor, err := musicproc.New(musicproc.Config{
    // مسیر باینری‌های FFmpeg (اختیاری - auto-detect از PATH)
    FFmpegPath:  "/usr/local/bin/ffmpeg",
    FFprobePath: "/usr/local/bin/ffprobe",

    // تعداد worker برای batch processing
    Workers: 8,

    // کانال دریافت progress (اختیاری)
    ProgressCh: progressChan,

    // logger سفارشی (اختیاری)
    ZapLogger: myZapLogger,

    // تنظیمات retry سفارشی (اختیاری)
    RetryConfig: &retry.Config{
        MaxAttempts: 5,
        Delay:       2 * time.Second,
        Multiplier:  2.0,
        MaxDelay:    60 * time.Second,
    },
})
```

### ProcessAudio — پردازش تک فایل

```go
result, err := processor.ProcessAudio(ctx, inputPath, outputPath,
    // تنظیمات codec
    musicproc.WithCodec(musicproc.CodecOpus),      // CodecOpus | CodecAAC | CodecMP3
    musicproc.WithBitrate(128_000),                 // بیت‌ریت به bps
    musicproc.WithBitrateMode(musicproc.BitrateModeVBR), // VBR یا CBR
    musicproc.WithSampleRate(48_000),               // نرخ نمونه‌برداری

    // Normalization (EBU R128)
    musicproc.WithNormalization(true),
    musicproc.WithLoudnessTarget(-16.0),            // LUFS (پیش‌فرض: -23)

    // فیلترها
    musicproc.WithHighpass(80),                     // حذف فرکانس‌های زیر 80Hz
    musicproc.WithLowpass(18000),                   // حذف فرکانس‌های بالای 18kHz

    // تعداد تلاش مجدد
    musicproc.WithWorkers(4),
)
```

**خروجی ProcessingResult:**

```go
type ProcessingResult struct {
    InputPath    string           // مسیر فایل ورودی
    OutputPath   string           // مسیر فایل خروجی
    InputMeta    *AudioMetadata   // متادیتای ورودی
    OutputMeta   *AudioMetadata   // متادیتای خروجی
    Duration     time.Duration    // مدت زمان پردازش
    ProcessedAt  time.Time        // زمان پردازش
}
```

### ProcessBatch — پردازش دسته‌ای

```go
jobs := []musicproc.BatchJob{
    {
        ID:         "job-001",
        InputPath:  "track1.wav",
        OutputPath: "track1.opus",
        Options:    nil,  // استفاده از تنظیمات پیش‌فرض
    },
    {
        ID:         "job-002",
        InputPath:  "track2.wav",
        OutputPath: "track2.aac",
        Options: func() *model.ProcessingOptions {
            opts := model.DefaultProcessingOptions()
            opts.Codec = musicproc.CodecAAC
            opts.Bitrate = 256_000
            return opts
        }(),
    },
}

// بازگشت channel—نتایج به محض آماده شدن ارسال می‌شوند
resultsCh, err := processor.ProcessBatch(ctx, jobs)
if err != nil {
    log.Fatal(err)
}

for res := range resultsCh {
    if res.Err != nil {
        fmt.Printf("❌ [%s] failed: %v\n", res.JobID, res.Err)
        continue
    }
    fmt.Printf("✅ [%s] completed in %s\n", res.JobID, res.Result.Duration)
}
```

### ProbeAudio — خواندن متادیتا

```go
meta, err := processor.ProbeAudio(ctx, "audio.wav")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Duration  : %s\n", meta.Duration)
fmt.Printf("Codec     : %s\n", meta.Codec)
fmt.Printf("SampleRate: %d Hz\n", meta.SampleRate)
fmt.Printf("Channels  : %d\n", meta.Channels)
fmt.Printf("Bitrate   : %d bps\n", meta.Bitrate)
fmt.Printf("Format    : %s\n", meta.Format)
fmt.Printf("Size      : %d bytes\n", meta.Size)
```

---

## مثال‌های کاربردی

### مثال ۱: آپلود و پردازش برای پلتفرم استریم

```go
// استاندارد Spotify: -14 LUFS، Opus 160kbps
result, err := processor.ProcessAudio(ctx,
    "uploaded_raw.wav",
    "stream_ready.opus",
    musicproc.WithCodec(musicproc.CodecOpus),
    musicproc.WithBitrate(160_000),
    musicproc.WithSampleRate(48_000),
    musicproc.WithNormalization(true),
    musicproc.WithLoudnessTarget(-14.0),
    musicproc.WithHighpass(80),
)
```

### مثال ۲: پردازش با timeout و graceful shutdown

```go
func processWithGracefulShutdown(inputPath, outputPath string) error {
    // context با timeout ۵ دقیقه
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    // پشتیبانی از Ctrl+C
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        <-sigCh
        cancel() // لغو پردازش در حال انجام
    }()

    processor, _ := musicproc.New(musicproc.Config{Workers: 4})
    defer processor.Close()

    _, err := processor.ProcessAudio(ctx, inputPath, outputPath,
        musicproc.WithCodec(musicproc.CodecAAC),
        musicproc.WithBitrate(192_000),
    )
    return err
}
```

### مثال ۳: Progress tracking در real-time

```go
progressCh := make(chan musicproc.ProgressUpdate, 64)

processor, _ := musicproc.New(musicproc.Config{
    ProgressCh: progressCh,
})
defer processor.Close()

// listener در یک goroutine جداگانه
go func() {
    for upd := range progressCh {
        fmt.Printf("\r[%s] %-12s %.0f%%", upd.JobID, upd.Stage, upd.Percent)
        if upd.Stage == musicproc.StageDone {
            fmt.Println(" ✅")
        }
    }
}()

result, err := processor.ProcessAudio(ctx, "input.wav", "output.opus",
    musicproc.WithCodec(musicproc.CodecOpus),
    musicproc.WithNormalization(true),
)
```

### مثال ۴: Batch processing با custom options

```go
// پردازش یک آلبوم کامل
tracks := []string{"01.wav", "02.wav", "03.wav", "04.wav", "05.wav"}

jobs := make([]musicproc.BatchJob, len(tracks))
for i, track := range tracks {
    outputName := strings.TrimSuffix(track, ".wav") + ".opus"
    jobs[i] = musicproc.BatchJob{
        ID:         fmt.Sprintf("album-track-%02d", i+1),
        InputPath:  filepath.Join("raw", track),
        OutputPath: filepath.Join("processed", outputName),
        // Options = nil → تنظیمات پیش‌فرض اعمال می‌شود
    }
}

processor, _ := musicproc.New(musicproc.Config{Workers: 3})
defer processor.Close()

resultsCh, _ := processor.ProcessBatch(ctx, jobs)

var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    for res := range resultsCh {
        if res.Err != nil {
            log.Printf("track %s failed: %v", res.JobID, res.Err)
        } else {
            log.Printf("track %s done in %s", res.JobID, res.Result.Duration)
        }
    }
}()

wg.Wait()
```

### مثال ۵: استفاده با logger سفارشی

```go
import "go.uber.org/zap"

zapLogger, _ := zap.NewProduction()
defer zapLogger.Sync()

processor, _ := musicproc.New(musicproc.Config{
    ZapLogger: zapLogger,
    Workers:   8,
})
defer processor.Close()
```

---

## مدیریت خطا

```go
import (
    musicproc "github.com/musicproc"
    pkgerrors "github.com/musicproc/pkg/errors"
    "errors"
)

result, err := processor.ProcessAudio(ctx, input, output)
if err != nil {
    // بررسی نوع خطا
    var processingErr *pkgerrors.ProcessingError
    var ffmpegErr *pkgerrors.FFmpegError
    var validationErr *pkgerrors.ValidationError

    switch {
    case errors.As(err, &validationErr):
        // خطای اعتبارسنجی — ورودی نامعتبر
        fmt.Printf("validation error: field=%s value=%v msg=%s\n",
            validationErr.Field, validationErr.Value, validationErr.Message)

    case errors.As(err, &ffmpegErr):
        // خطای FFmpeg — مشکل در اجرا
        fmt.Printf("ffmpeg error: exit=%d stderr=%s\n",
            ffmpegErr.ExitCode, ffmpegErr.Stderr)

    case errors.As(err, &processingErr):
        // خطای پردازش — مشکل در مرحله‌ای از pipeline
        fmt.Printf("processing error at stage=%s: %v\n",
            processingErr.Stage, processingErr.Cause)

    case errors.Is(err, context.DeadlineExceeded):
        fmt.Println("processing timed out")

    case errors.Is(err, context.Canceled):
        fmt.Println("processing was canceled")

    default:
        fmt.Printf("unexpected error: %v\n", err)
    }
}
```

---

## پیکربندی پیشرفته

### تنظیمات پیش‌فرض ProcessingOptions

| پارامتر | پیش‌فرض | توضیح |
|---------|---------|-------|
| `Codec` | `opus` | کدک خروجی |
| `Bitrate` | `128000` | بیت‌ریت به bps |
| `BitrateMode` | `cbr` | حالت CBR |
| `SampleRate` | `48000` | نرخ نمونه‌برداری |
| `NormalizationEnabled` | `true` | فعال بودن EBU R128 |
| `LoudnessTarget` | `-23.0` | هدف loudness به LUFS |
| `TruePeakLimit` | `-1.0` | محدودیت true peak به dBTP |
| `LoudnessRange` | `7.0` | محدوده LRA به LU |
| `Timeout` | `5m` | timeout پردازش |
| `Workers` | `4` | تعداد worker |
| `MaxRetries` | `3` | تعداد تلاش مجدد |
| `RetryDelay` | `1s` | تأخیر اولیه retry |

### استانداردهای Loudness پلتفرم‌های محبوب

| پلتفرم | Target LUFS | True Peak |
|--------|-------------|-----------|
| Spotify | -14.0 | -1.0 dBTP |
| YouTube | -14.0 | -1.0 dBTP |
| Apple Music | -16.0 | -1.0 dBTP |
| EBU R128 (Broadcast) | -23.0 | -1.0 dBTP |

---

## ملاحظات Production

### Graceful Shutdown

```go
processor, _ := musicproc.New(musicproc.Config{Workers: 4})

// signal listener
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

ctx, cancel := context.WithCancel(context.Background())

go func() {
    <-sigCh
    log.Println("shutdown signal received, canceling jobs...")
    cancel()
}()

// اجرای batch
resultsCh, _ := processor.ProcessBatch(ctx, jobs)
for res := range resultsCh {
    // پردازش نتایج...
}

processor.Close()
```

### Resource Management

- تمام فایل‌های موقت در صورت خطا پاک‌سازی می‌شوند
- goroutine leak جلوگیری شده از طریق worker pool با semaphore
- context cancellation در تمام لایه‌ها منتشر می‌شود

### Observability

```go
// اضافه کردن logger سفارشی با fields محیطی
zapLogger, _ := zap.NewProduction()
zapLogger = zapLogger.With(
    zap.String("service", "audio-processor"),
    zap.String("environment", "production"),
)

processor, _ := musicproc.New(musicproc.Config{
    ZapLogger: zapLogger,
})
```

### امنیت

- اجتناب از تزریق command line: تمام آرگومان‌های FFmpeg به صورت ساختاریافته ساخته می‌شوند
- بدون global state: ایمن برای استفاده همزمان
- مسیرهای فایل اعتبارسنجی می‌شوند قبل از اجرا