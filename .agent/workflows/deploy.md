---
description: Loyihani serverga yangilash (Deploy to Server)
---

Ushbu workflow o'zgarishlarni serverga (213.206.62.196) yuklash ketma-ketligini tushuntiradi.

### 1-qadam: Lokal kompyuterda (Hozirgi joyda)

O'zgarishlarni Git repozitoriysiga yuboring. Terminalda quyidagi buyruqlarni birma-bir bajaring:

```bash
# Hamma o'zgarishlarni qo'shish
git add .

# Commit qilish (izoh bilan)
git commit -m "Fix: DB schema update, Bot upload folder permissions, Excel detection"

# GitHub/GitLab ga yuklash
git push origin main
```
*(Eslatma: Agar `git push` paytida login/parol so'rasa, kiriting)*

---

### 2-qadam: Serverda (VPS)

Serverga ulaning (agar hali ulanmagan bo'lsangiz):
```bash
ssh root@213.206.62.196
```

Loyiha papkasiga kiring (taxminan `convertpdfgo` yoki shunga o'xshash):
```bash
cd convertpdfgo
```

**Diqqat:** Agar serverda hali loyiha bo'lmasa, uni klonlang:
`git clone <REPO_URL> convertpdfgo`

O'zgarishlarni qabul qiling:
```bash
git pull origin main
```

---

### 3-qadam: Ma'lumotlar bazasini tozalash (Muhim!)

Serverdagi bazada ham eski jadvallar qolib ketgan bo'lishi mumkin. Ularni tozalaymiz (xuddi lokalda qilganimizdek):

```bash
docker exec pdfbot_postgres psql -U postgres -d convertpdfgo -c "DROP TABLE IF EXISTS merge_jobs CASCADE;"
```

---

### 4-qadam: Botni yangilash va ishga tushirish

Yangi kod bilan containerlarni qayta quramiz va ishga tushiramiz:

```bash
# Agar Makefile bo'lsa:
make up

# Yoki to'g'ridan-to'g'ri Docker Compose bilan:
docker compose up -d --build
```

### 5-qadam: Tekshirish

Bot ishlayotganini loglardan tekshiring:
```bash
docker logs -f pdfbot_app
```
(Loglardan chiqish uchun `Ctrl+C` bosing).
