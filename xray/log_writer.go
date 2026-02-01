package xray

import (
    "regexp"
    "strings"

    "x-ui/logger"
)

// --- ارتقا 1: کامپایل Regex در سطح پکیج (بهینه‌سازی شدید) ---
// کامپایل Regex داخل تابع Write باعث هدر رفتن شدید CPU می‌شود.
var (
    crashRegex   *regexp.Regexp
    xrayLogRegex *regexp.Regexp
)

func init() {
    // این کدها فقط یک بار در هنگام اجرای اولیه برنامه اجرا می‌شوند
    crashRegex = regexp.MustCompile(`(?i)(panic|exception|stack trace|fatal error)`)
    xrayLogRegex = regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}\.\d{6}) \[([^\]]+)\] (.+)$`)
}

func NewLogWriter() *LogWriter {
    return &LogWriter{}
}

type LogWriter struct {
    lastLine string
}

func (lw *LogWriter) Write(m []byte) (n int, err error) {
    // Check if it is a crash
    // از متغیر سراسری استفاده می‌کنیم
    if crashRegex.MatchString(m) {
        message := string(m)
        logger.Debug("Core crash detected:\n", message)
        lw.lastLine = message
        err1 := writeCrachReport(m)
        if err1 != nil {
            logger.Error("Unable to write crash report:", err1)
        }
        return len(m), nil
    }

    // --- ارتقا 2: اصلاح نام تابع Split (SplitSeq وجود ندارد) ---
    // پارس کردن پیام به خطوط جداگانه
    message := strings.TrimSpace(string(m))
    messages := strings.Split(message, "\n")

    for _, msg := range messages {
        // نادیده گرفتن خطوط خالی
        if msg == "" {
            continue
        }

        matches := xrayLogRegex.FindStringSubmatch(msg)

        if len(matches) > 3 {
            // فرمت استاندارد Xray پیدا شد: [Info] Message
            level := matches[2]
            msgBody := matches[3]
            msgBodyLower := strings.ToLower(msgBody)

            // فیلتر کردن خطاهای عادی (که نویز محسوب می‌شوند)
            if strings.Contains(msgBodyLower, "tls handshake error") ||
                strings.Contains(msgBodyLower, "connection ends") ||
                strings.Contains(msgBodyLower, "connection reset") {
                // خطاهای قطع ارتباط را در UI نشان نده و فقط Debug بنویس
                logger.Debug("XRAY: " + msgBody)
                lw.lastLine = ""
                continue
            }

            // لاگ‌ریزی بر اساس سطح خطا
            if strings.Contains(msgBodyLower, "failed") {
                logger.Error("XRAY: " + msgBody)
            } else {
                switch level {
                case "Debug":
                    logger.Debug("XRAY: " + msgBody)
                case "Info":
                    logger.Info("XRAY: " + msgBody)
                case "Warning":
                    logger.Warning("XRAY: " + msgBody)
                case "Error":
                    logger.Error("XRAY: " + msgBody)
                default:
                    logger.Debug("XRAY: " + msg)
                }
            }
            lw.lastLine = ""
        } else {
            // فرمت غیر استاندارد (مثلا لاگ‌های استارت برنامه)
            msgLower := strings.ToLower(msg)
            
            // جلوگیری از نمایش خطاهای قطع ارتباط در آخرین خط
            if strings.Contains(msgLower, "tls handshake error") ||
               strings.Contains(msgLower, "connection ends") {
                logger.Debug("XRAY: " + msg)
                lw.lastLine = msg
                continue
            }

            if strings.Contains(msgLower, "failed") {
                logger.Error("XRAY: " + msg)
            } else {
                logger.Debug("XRAY: " + msg)
            }
            lw.lastLine = msg
        }
    }

    return len(m), nil
}
