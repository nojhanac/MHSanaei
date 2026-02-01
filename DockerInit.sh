#!/bin/sh

# --- ارتقا 1: خروج سریع در صورت بروز خطا (Error Handling) ---
set -e

# --- ارتقا 2: امکان تعیین نسخه به صورت متغیر ---
# اگر متغیر XRAY_VERSION ست نشده باشد، آخرین نسخه را پیدا می‌کند
LATEST_VERSION=$(wget -qO- "https://api.github.com/repos/XTLS/Xray-core/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
XRAY_VERSION=${XRAY_VERSION:-$LATEST_VERSION}

echo "Installing Xray version: $XRAY_VERSION"

case $1 in
    amd64)
        ARCH="64"
        FNAME="amd64"
        ;;
    i386)
        ARCH="32"
        FNAME="i386"
        ;;
    armv8 | arm64 | aarch64)
        ARCH="arm64-v8a"
        FNAME="arm64"
        ;;
    armv7 | arm | arm32)
        ARCH="arm32-v7a"
        FNAME="arm32"
        ;;
    armv6)
        ARCH="arm32-v6"
        FNAME="armv6"
        ;;
    *)
        ARCH="64"
        FNAME="amd64"
        ;;
esac

mkdir -p build/bin
cd build/bin

# دانلود هسته Xray با متغیر نسخه
# استفاده از -O برای نامگذاری فایل
wget -q "https://github.com/XTLS/Xray-core/releases/download/${XRAY_VERSION}/Xray-linux-${ARCH}.zip" -O "Xray-linux-${ARCH}.zip"

# --- ارتقا 3: بررسی موفقیت‌آمیز بودن دانلود ---
if [ ! -f "Xray-linux-${ARCH}.zip" ]; then
    echo "Error: Failed to download Xray core."
    exit 1
fi

unzip -o "Xray-linux-${ARCH}.zip"
rm -f "Xray-linux-${ARCH}.zip" geoip.dat geosite.dat
mv xray "xray-linux-${FNAME}"

# --- ارتقا 4: دادن مجوز اجرایی ---
chmod +x "xray-linux-${FNAME}"

# دانلود فایل‌های Geo (استفاده از پارامتر -t برای محدودیت زمانی در صورت لگ بودن)
wget -q --timeout=30 https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat
wget -q --timeout=30 https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat
wget -q -O geoip_IR.dat --timeout=30 https://github.com/chocolate4u/Iran-v2ray-rules/releases/latest/download/geoip.dat
wget -q -O geosite_IR.dat --timeout=30 https://github.com/chocolate4u/Iran-v2ray-rules/releases/latest/download/geosite.dat
wget -q -O geoip_RU.dat --timeout=30 https://github.com/runetfreedom/russia-v2ray-rules-dat/releases/latest/download/geoip.dat
wget -q -O geosite_RU.dat --timeout=30 https://github.com/runetfreedom/russia-v2ray-rules-dat/releases/latest/download/geosite.dat

cd ../../
