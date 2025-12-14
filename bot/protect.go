package bot

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
)
// ==================== PROTECT PDF ====================

func (b *Bot) startProtectFlow(ctx context.Context, chatID, userID int64) {
	lang := b.getUserLang(ctx, userID)
	
	b.userStates.Store(userID, StateProtectUpload)

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("↩️ " + T(lang, "back_to_menu")),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "🔒 *Protect PDF*\n\nParol qo'yish uchun PDF faylni yuboring:")
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

func (b *Bot) processProtect(ctx context.Context, chatID, userID int64, fileID, password, lang string) {
	// Coin tekshirish
	canUse, coins, isPremium, _ := b.services.BotUser().CanUseService(ctx, userID)
	if !canUse {
		refLink := fmt.Sprintf("https://t.me/%s?start=ref%d", b.botUsername, userID)
		b.sendNoCoinsMessage(chatID, lang, refLink)
		return
	}
	if !isPremium && coins <= 5 && coins > 0 {
		b.sendMessage(chatID, fmt.Sprintf(T(lang, "coins_left"), coins))
	}

	b.sendMessage(chatID, "⏳ PDF himoyalanmoqda...")
	b.api.Send(tgbotapi.NewChatAction(chatID, tgbotapi.ChatUploadDocument))

	// Parol minimal uzunligi tekshirish
	if len(password) < 4 {
		b.sendMessage(chatID, "❌ Parol kamida 4 ta belgidan iborat bo'lishi kerak!")
		b.sendMainMenu(chatID, lang)
		return
	}

	req := models.ProtectPDFRequest{
		InputFileID: fileID,
		Password:    password,
	}

	jobID, err := b.services.Protect().Create(ctx, req, nil)
	if err != nil {
		b.log.Error("Protect failed", logger.Error(err))
		b.sendMessage(chatID, "❌ Xatolik: "+err.Error())
		b.sendMainMenu(chatID, lang)
		return
	}

	job, err := b.services.Protect().GetByID(ctx, jobID)
	if err != nil || job == nil || job.OutputFileID == nil {
		b.sendMessage(chatID, "❌ Natija topilmadi")
		b.sendMainMenu(chatID, lang)
		return
	}

	b.services.BotUser().UseService(ctx, userID)
	b.sendResult(chatID, userID, job.OutputFileID, lang)
	b.sendMessage(chatID, fmt.Sprintf("✅ PDF himoyalandi! Parol: %s", password))
	b.sendMainMenu(chatID, lang)
}
