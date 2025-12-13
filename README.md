# ConvertPDFGo - Telegram PDF Bot

PDF fayllar bilan ishlash uchun Telegram bot.

## 🚀 Funksiyalar

- 📝 Word → PDF
- 📊 Excel → PDF
- 📑 PowerPoint → PDF
- 🖼 JPG → PDF
- 🗜 PDF Compress
- 🖇 PDF Merge
- ✂️ PDF Split
- 🔄 PDF Rotate
- 💧 Watermark
- 🔓 Unlock PDF

## 📦 O'rnatish

### 1. .env faylini yaratish

```bash
cp .env.example .env
# .env faylini o'zgartiring
```

### 2. Docker bilan ishga tushirish

```bash
# Build va start
docker-compose up -d --build

# Loglarni ko'rish
docker-compose logs -f app

# To'xtatish
docker-compose down
```

### 3. Local ishga tushirish

```bash
# Dependencies
go mod download

# PostgreSQL va Gotenberg ishga tushirish
docker-compose up -d postgres gotenberg

# Botni ishga tushirish
go run cmd/main.go
```

## ⚙️ Muhit o'zgaruvchilari

| O'zgaruvchi | Tavsif |
|-------------|--------|
| `BOT_TOKEN` | Telegram Bot Token (@BotFather dan) |
| `ADMIN_USER_ID` | Admin Telegram ID |
| `POSTGRES_*` | PostgreSQL ulanish ma'lumotlari |
| `GOTENBERG_URL` | Gotenberg serveri manzili |

## 📊 Admin Komandalar

```
/stats    - Statistika
/cleanup  - Eski fayllarni o'chirish
/addcoins - Coin qo'shish
/top      - Top foydalanuvchilar
```

## 💳 Payment

1 Telegram Star = 1 Coin = 1 PDF operatsiya

## 📁 Papka tuzilishi

```
storage/
├── bot_uploads/   # Yuklangan fayllar
├── compress/      # Siqilgan fayllar
├── jpg_to_pdf/    # Konvertatsiya
├── merge/         # Birlashtirilgan
├── split/         # Ajratilgan
├── rotate/        # Aylantirilgan
├── watermark/     # Suv belgili
└── unlock/        # Qulfdan chiqarilgan
```

## 📄 License

MIT# convertpdfgobot
