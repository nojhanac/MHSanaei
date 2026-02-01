#!/bin/sh

echo "================================================"
echo "Starting 3x-ui Container Entrypoint..."
echo "================================================"

# --- ارتقا 1: بررسی و آماده‌سازی فایل‌های لاگ ---
# Fail2ban برای کار کردن نیاز به وجود فایل لاگ دارد، حتی اگر خالی باشد
mkdir -p /var/log/x-ui
touch /var/log/x-ui/3xipl.log

# --- ارتقا 2: فعال‌سازی امن Fail2ban با کنترل خطا ---
if [ "$X_UI_ENABLE_FAIL2BAN" = "true" ]; then
    echo "Initializing Fail2ban..."
    
    # بررسی اینکه آیا fail2ban نصب است یا خیر
    if command -v fail2ban-client > /dev/null 2>&1; then
        # پرچم -x (Force) به ما اطمینان می‌دهد که اگر سرویس در حال اجرا بود، تداخلی پیش نیاید
        # پرچم -b برای Background است تا بقیه سیستم قفل نشود
        if ! fail2ban-client -x start > /dev/null 2>&1; then
            echo "Warning: Fail2ban failed to start or is already running in background."
        else
            echo "Fail2ban started successfully."
        fi
    else
        echo "Warning: Fail2ban command not found. Skipping security initialization."
    fi
else
    echo "Fail2ban is disabled via environment variables."
fi

echo "================================================"
echo "Starting X-UI Panel..."
echo "================================================"

# --- ارتقا 3: اجرای نهایی با exec ---
# exec باعث می‌شود که سرویس پنل PID=1 باشد و سیگنال‌های داکر را مستقیماً دریافت کند
exec /app/x-ui
