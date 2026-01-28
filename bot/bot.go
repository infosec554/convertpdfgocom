package bot

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/service"
)

const (
	StateNone                  = ""
	StateAddCoinsUser          = "admin_add_coins_user"
	StateAddCoinsAmount        = "admin_add_coins_amount"
	StateMergeUpload           = "merge_upload_files"
	StateSplitUpload           = "split_upload_file"
	StateSplitPages            = "split_enter_pages"
	StateRotateUpload          = "rotate_upload_file"
	StateWatermarkUpload       = "watermark_upload"
	StateWatermarkText         = "watermark_enter_text"
	StateUnlockUpload          = "unlock_upload_file"
	StatePDFToJPGUpload        = "pdf_to_jpg_upload"
	StateWordToPDFUpload       = "word_to_pdf_upload"
	StateExcelToPDFUpload      = "excel_to_pdf_upload"
	StatePowerPointToPDFUpload = "ppt_to_pdf_upload"
	StateJPGToPDFUpload        = "jpg_to_pdf_upload"
	StateProtectUpload         = "protect_upload_file"
	StateProtectPassword       = "protect_enter_password"
)

type Bot struct {
	api              *tgbotapi.BotAPI
	services         service.IServiceManager
	log              logger.ILogger
	adminUserID      int64
	botUsername      string
	requiredChannel  string
	processingUsers  sync.Map
	userStates       sync.Map
	adminContext     sync.Map
	mergeContext     sync.Map
	splitContext     sync.Map
	jpgContext       sync.Map
	mediaGroupWarned sync.Map
	workerPool       chan struct{}
}

func New(token string, services service.IServiceManager, log logger.ILogger, adminUserID string, requiredChannel string) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	var adminID int64
	if adminUserID != "" {
		adminID, _ = strconv.ParseInt(adminUserID, 10, 64)
	}

	return &Bot{
		api:             api,
		services:        services,
		log:             log,
		adminUserID:     adminID,
		botUsername:     api.Self.UserName,
		requiredChannel: requiredChannel,
		workerPool:      make(chan struct{}, 10),
	}, nil
}

func (b *Bot) GetAPI() *tgbotapi.BotAPI {
	return b.api
}

func (b *Bot) Start(ctx context.Context) {
	b.log.Info("Bot started", logger.String("username", b.api.Self.UserName))
	b.log.Info("Configured Admin ID", logger.Int64("admin_id", b.adminUserID))

	commands := []tgbotapi.BotCommand{
		{Command: "start", Description: "Menyuni ko'rsatish"},
		{Command: "help", Description: "Yordam va rejimlar"},
		{Command: "pdf_to_word", Description: "PDF → Word"},
		{Command: "word_to_pdf", Description: "Word → PDF"},
		{Command: "pdf_to_jpg", Description: "PDF → JPG (ZIP)"},
		{Command: "jpg_to_pdf", Description: "JPG → PDF"},
		{Command: "compress", Description: "PDF siqish"},
		{Command: "merge", Description: "PDF birlashtirish"},
		{Command: "split", Description: "PDF ajratish"},
		{Command: "rotate", Description: "PDF aylantirish"},
		{Command: "watermark", Description: "Suv belgisi qo'shish"},
		{Command: "unlock", Description: "PDF qulfini ochish"},
		{Command: "mycoins", Description: "Tangalarim"},
		{Command: "ref", Description: "Referal havola"},
		{Command: "lang", Description: "Tilni o'zgartirish"},
	}

	cmdConfig := tgbotapi.NewSetMyCommands(commands...)
	if _, err := b.api.Request(cmdConfig); err != nil {
		b.log.Error("Failed to set bot commands", logger.Error(err))
	} else {
		b.log.Info("Bot commands set successfully")
	}

	b.resetMenuButton()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			b.log.Info("Bot stopped")
			return
		case update := <-updates:
			go b.handleUpdate(ctx, update)
		}
	}
}

func (b *Bot) isAdmin(userID int64) bool {
	return b.adminUserID != 0 && userID == b.adminUserID
}

func (b *Bot) getUserLang(ctx context.Context, userID int64) string {
	user, _ := b.services.BotUser().GetUser(ctx, userID)
	if user != nil && user.Language != "" {
		return user.Language
	}
	return "uz"
}

func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	if update.PreCheckoutQuery != nil {
		b.handlePreCheckout(update.PreCheckoutQuery)
		return
	}

	if update.CallbackQuery != nil {
		b.handleCallback(ctx, update.CallbackQuery)
		return
	}

	if update.Message == nil {
		return
	}

	if update.Message.SuccessfulPayment != nil {
		b.handleSuccessfulPayment(ctx, update.Message)
		return
	}

	if update.Message.MediaGroupID != "" {
		lang := b.getUserLang(ctx, update.Message.From.ID)
		if _, exists := b.mediaGroupWarned.LoadOrStore(update.Message.MediaGroupID, true); !exists {
			b.sendMessage(update.Message.Chat.ID, T(lang, "only_one_file"))
			go func(groupID string) {
				time.Sleep(30 * time.Second)
				b.mediaGroupWarned.Delete(groupID)
			}(update.Message.MediaGroupID)
		}
		return
	}

	var refID *int64
	if update.Message.IsCommand() && update.Message.Command() == "start" {
		args := update.Message.CommandArguments()
		if strings.HasPrefix(args, "ref") {
			if id, err := strconv.ParseInt(strings.TrimPrefix(args, "ref"), 10, 64); err == nil && id != update.Message.From.ID {
				refID = &id
			}
		}
	}
	b.ensureUser(ctx, update.Message.From, refID)

	isSubscribed, _ := b.checkSubscription(ctx, update.Message.From.ID)
	if !isSubscribed {
		lang := b.getUserLang(ctx, update.Message.From.ID)
		b.sendSubscriptionRequest(update.Message.Chat.ID, lang)
		return
	}

	user, _ := b.services.BotUser().GetUser(ctx, update.Message.From.ID)
	if user != nil && user.Language == "" && update.Message.Command() == "start" {
		b.sendLanguageSelection(update.Message.Chat.ID)
		return
	}

	if refID != nil {
		b.notifyReferral(ctx, update.Message.Chat.ID, update.Message.From, *refID)
	}

	if update.Message.IsCommand() {
		b.handleCommand(ctx, update.Message)
		return
	}

	if update.Message.Document != nil {
		b.handleDocument(ctx, update.Message)
		return
	}

	if update.Message.Photo != nil && len(update.Message.Photo) > 0 {
		b.handlePhoto(ctx, update.Message)
		return
	}

	if update.Message.Text != "" {
		lang := b.getUserLang(ctx, update.Message.From.ID)
		if update.Message.From.ID == b.adminUserID {
			switch update.Message.Text {
			case "👨‍⚖️ Admin Panel":
				b.userStates.Delete(update.Message.From.ID)
				b.sendAdminPanel(update.Message.Chat.ID)
				return
			case "📊 Global Stats":
				b.handleAdminStats(ctx, update.Message, lang)
				return
			case "➕ Add Coins":
				b.userStates.Store(update.Message.From.ID, StateAddCoinsUser)
				b.sendMessage(update.Message.Chat.ID, "👤 User ID yoki @username kiriting:")
				return
			case "👥 Users List":
				b.sendUserList(ctx, update.Message.Chat.ID)
				return
			case "🗄 Backup":
				b.sendBackupExcel(ctx, update.Message.Chat.ID)
				return
			case "↩️ User Mode":
				b.userStates.Delete(update.Message.From.ID)
				b.mergeContext.Delete(update.Message.From.ID)
				b.splitContext.Delete(update.Message.From.ID)
				lang := b.getUserLang(ctx, update.Message.From.ID)
				b.sendUserMenu(update.Message.Chat.ID, lang)
				return
			case "🏠 Asosiy menu", "🏠 Menu", "🏠 Main menu", "🏠 Главное меню":
				b.userStates.Delete(update.Message.From.ID)
				b.sendAdminPanel(update.Message.Chat.ID)
				return
			}
		}

		if state, ok := b.userStates.Load(update.Message.From.ID); ok && state != StateNone {
			if update.Message.From.ID == b.adminUserID {
				b.handleAdminStates(ctx, &update, state.(string))
				return
			}
		}

		if key := GetKey(lang, update.Message.Text); key == "merge_pdf" {
			b.startMergeFlow(ctx, update.Message.Chat.ID, update.Message.From.ID)
			return
		}

		if key := GetKey(lang, update.Message.Text); key == "split_pdf" {
			b.startSplitFlow(ctx, update.Message.Chat.ID, update.Message.From.ID)
			return
		}

		if key := GetKey(lang, update.Message.Text); key == "rotate_pdf" {
			b.startRotateFlow(ctx, update.Message.Chat.ID, update.Message.From.ID)
			return
		}

		if key := GetKey(lang, update.Message.Text); key == "watermark" {
			b.startWatermarkFlow(ctx, update.Message.Chat.ID, update.Message.From.ID)
			return
		}

		if key := GetKey(lang, update.Message.Text); key == "unlock_pdf" {
			b.startUnlockFlow(ctx, update.Message.Chat.ID, update.Message.From.ID)
			return
		}

		if key := GetKey(lang, update.Message.Text); key == "pdf_to_jpg" {
			b.startPDFToJPGFlow(ctx, update.Message.Chat.ID, update.Message.From.ID)
			return
		}

		if update.Message.Text == "🔒 Protect PDF" {
			b.startProtectFlow(ctx, update.Message.Chat.ID, update.Message.From.ID)
			return
		}

		if update.Message.Text == "Done" || update.Message.Text == "/done" {
			if state, ok := b.userStates.Load(update.Message.From.ID); ok && state == StateMergeUpload {
				b.finishMergeFlow(ctx, update.Message.Chat.ID, update.Message.From.ID)
				return
			}
			b.sendMessage(update.Message.Chat.ID, "ℹ️ Hozirda birlashtirish jarayoni yo'q. Avval \"Merge PDF\" ni tanlang.")
			return
		}

		if key := GetKey(lang, update.Message.Text); key == "cancel" {
			b.userStates.Delete(update.Message.From.ID)
			b.mergeContext.Delete(update.Message.From.ID)
			b.splitContext.Delete(update.Message.From.ID)
			b.sendMessage(update.Message.Chat.ID, "✅ Bekor qilindi")
			b.sendMainMenu(update.Message.Chat.ID, lang)
			return
		}

		if state, ok := b.userStates.Load(update.Message.From.ID); ok && state == StateSplitPages {
			b.finishSplitFlow(ctx, update.Message.Chat.ID, update.Message.From.ID, update.Message.Text)
			return
		}

		if state, ok := b.userStates.Load(update.Message.From.ID); ok && state == StateWatermarkText {
			if fid, ok := b.splitContext.Load(update.Message.From.ID); ok {
				go b.processWatermark(ctx, update.Message.Chat.ID, update.Message.From.ID, fid.(string), update.Message.Text, lang)
				b.userStates.Delete(update.Message.From.ID)
				b.splitContext.Delete(update.Message.From.ID)
				return
			}
		}

		if state, ok := b.userStates.Load(update.Message.From.ID); ok && state == StateProtectPassword {
			if fid, ok := b.splitContext.Load(update.Message.From.ID); ok {
				go b.processProtect(ctx, update.Message.Chat.ID, update.Message.From.ID, fid.(string), update.Message.Text, lang)
				b.userStates.Delete(update.Message.From.ID)
				b.splitContext.Delete(update.Message.From.ID)
				return
			}
		}

		if key := GetKey(lang, update.Message.Text); key != "" {
			switch key {
			case "compress_pdf":
				b.services.BotUser().SetCurrentAction(ctx, update.Message.From.ID, "compress")
				b.sendMessage(update.Message.Chat.ID, T(lang, "select_file"))
			case "word_to_pdf":
				b.services.BotUser().SetCurrentAction(ctx, update.Message.From.ID, "word")
				b.userStates.Store(update.Message.From.ID, StateWordToPDFUpload)
				b.sendMessage(update.Message.Chat.ID, T(lang, "select_file"))
			case "excel_to_pdf":
				b.services.BotUser().SetCurrentAction(ctx, update.Message.From.ID, "excel")
				b.userStates.Store(update.Message.From.ID, StateExcelToPDFUpload)
				b.sendMessage(update.Message.Chat.ID, T(lang, "select_file"))
			case "ppt_to_pdf":
				b.services.BotUser().SetCurrentAction(ctx, update.Message.From.ID, "ppt")
				b.userStates.Store(update.Message.From.ID, StatePowerPointToPDFUpload)
				b.sendMessage(update.Message.Chat.ID, T(lang, "select_file"))
			case "image_to_pdf":
				b.startJPGToPDFFlow(ctx, update.Message.Chat.ID, update.Message.From.ID)
			case "my_coins":
				b.showUserCoins(ctx, update.Message.Chat.ID, update.Message.From.ID, lang)
			case "referral":
				b.showReferralLink(update.Message.Chat.ID, update.Message.From.ID, lang)
			case "settings":
				b.sendLanguageSelection(update.Message.Chat.ID)
			case "support":
				b.sendMessage(update.Message.Chat.ID, T(lang, "support_msg"))
			case "about":
				b.sendMessage(update.Message.Chat.ID, T(lang, "about_msg"))
			case "partnership":
				b.sendPartnershipInfo(update.Message.Chat.ID, lang)
			case "back_to_menu":
				b.sendMainMenu(update.Message.Chat.ID, lang)
			}
			return
		}

		if update.Message.Text == "🏠 Menu" || update.Message.Text == "🏠 Asosiy menu" {
			b.sendMainMenu(update.Message.Chat.ID, lang)
			return
		}
	}

	lang := b.getUserLang(ctx, update.Message.From.ID)
	b.sendMainMenu(update.Message.Chat.ID, lang)
}

func (b *Bot) handleCallback(ctx context.Context, callback *tgbotapi.CallbackQuery) {
	data := callback.Data
	chatID := callback.Message.Chat.ID
	userID := callback.From.ID

	if data == "check_sub" {
		lang := b.getUserLang(ctx, userID)
		isSubscribed, status := b.checkSubscription(ctx, userID)
		if isSubscribed {
			b.api.Request(tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID))
			b.sendMessage(chatID, T(lang, "sub_success"))
			b.sendMainMenu(chatID, lang)
		} else {
			msg := fmt.Sprintf("%s (%s)", T(lang, "sub_not_found"), status)
			b.api.Request(tgbotapi.NewCallback(callback.ID, msg))
		}
		return
	}

	if !strings.HasPrefix(data, "lang_") {
		isSubscribed, _ := b.checkSubscription(ctx, userID)
		if !isSubscribed {
			lang := b.getUserLang(ctx, userID)
			b.api.Request(tgbotapi.NewCallback(callback.ID, T(lang, "sub_required")))
			b.sendSubscriptionRequest(chatID, lang)
			return
		}
	}

	b.api.Request(tgbotapi.NewCallback(callback.ID, ""))

	if strings.HasPrefix(data, "lang_") {
		lang := strings.TrimPrefix(data, "lang_")
		b.services.BotUser().SetLanguage(ctx, userID, lang)

		b.api.Request(tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID))
		b.sendMessage(chatID, T(lang, "lang_selected"))
		b.sendMainMenu(chatID, lang)
		return
	}

	if strings.HasPrefix(data, "rotate:") {
		parts := strings.Split(data, ":")
		if len(parts) == 3 {
			fileID := parts[1]
			angle, _ := strconv.Atoi(parts[2])
			lang := b.getUserLang(ctx, userID)
			b.processRotate(ctx, chatID, userID, fileID, angle, lang)
		}
		return
	}

	lang := b.getUserLang(ctx, userID)

	switch data {
	case "menu_compress":
		b.services.BotUser().SetCurrentAction(ctx, userID, "compress")
		b.sendMessage(chatID, T(lang, "select_file"))
	case "menu_word":
		b.services.BotUser().SetCurrentAction(ctx, userID, "word")
		b.userStates.Store(userID, StateWordToPDFUpload)
		b.sendMessage(chatID, T(lang, "select_file"))
	case "menu_excel":
		b.services.BotUser().SetCurrentAction(ctx, userID, "excel")
		b.userStates.Store(userID, StateExcelToPDFUpload)
		b.sendMessage(chatID, T(lang, "select_file"))
	case "menu_ppt":
		b.services.BotUser().SetCurrentAction(ctx, userID, "ppt")
		b.userStates.Store(userID, StatePowerPointToPDFUpload)
		b.sendMessage(chatID, T(lang, "select_file"))
	case "menu_image":
		b.startJPGToPDFFlow(ctx, chatID, userID)
	case "menu_coins":
		b.showUserCoins(ctx, chatID, userID, lang)
	case "menu_ref", "referral":
		b.showReferralLink(chatID, userID, lang)
	case "menu_settings":
		b.sendLanguageSelection(chatID)
	case "menu_support":
		b.sendMessage(chatID, T(lang, "support_msg"))
	case "menu_about":
		b.sendMessage(chatID, T(lang, "about_msg"))
	case "menu_back":
		b.sendMainMenu(chatID, lang)
	case "menu_partnership":
		b.sendPartnershipInfo(chatID, lang)
	case "buy_coins":
		b.sendBuyCoinsMenu(chatID, lang)
	case "buy_1":
		b.sendStarsInvoice(chatID, userID, lang, 1, 1, "coins")
	case "menu_merge":
		b.startMergeFlow(ctx, chatID, userID)
	case "menu_split":
		b.startSplitFlow(ctx, chatID, userID)
	case "menu_rotate":
		b.startRotateFlow(ctx, chatID, userID)
	case "menu_watermark":
		b.startWatermarkFlow(ctx, chatID, userID)
	case "menu_unlock":
		b.startUnlockFlow(ctx, chatID, userID)
	case "menu_pdf_to_jpg":
		b.startPDFToJPGFlow(ctx, chatID, userID)
	case "menu_protect":
		b.startProtectFlow(ctx, chatID, userID)
	case "jpg_done":
		b.finishJPGToPDFFlow(ctx, chatID, userID)
	case "jpg_cancel":
		b.jpgContext.Delete(userID)
		b.userStates.Delete(userID)
		b.api.Request(tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID))
		b.sendMessage(chatID, "❌ Bekor qilindi")
		b.sendMainMenu(chatID, lang)
	}
}

func (b *Bot) sendLanguageSelection(chatID int64) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇺🇿 O'zbekcha", "lang_uz"),
			tgbotapi.NewInlineKeyboardButtonData("🇷🇺 Русский", "lang_ru"),
			tgbotapi.NewInlineKeyboardButtonData("🇬🇧 English", "lang_en"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇸🇦 العربية", "lang_ar"),
			tgbotapi.NewInlineKeyboardButtonData("🇫🇷 Français", "lang_fr"),
			tgbotapi.NewInlineKeyboardButtonData("🇩🇪 Deutsch", "lang_de"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇪🇸 Español", "lang_es"),
			tgbotapi.NewInlineKeyboardButtonData("🇮🇹 Italiano", "lang_it"),
			tgbotapi.NewInlineKeyboardButtonData("🇵🇹 Português", "lang_pt"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇹🇷 Türkçe", "lang_tr"),
			tgbotapi.NewInlineKeyboardButtonData("🇨🇳 中文", "lang_zh"),
			tgbotapi.NewInlineKeyboardButtonData("🇯🇵 日本語", "lang_ja"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇰🇷 한국어", "lang_ko"),
			tgbotapi.NewInlineKeyboardButtonData("🇮🇳 हिंदी", "lang_hi"),
			tgbotapi.NewInlineKeyboardButtonData("🇮🇩 Indonesia", "lang_id"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇻🇳 Tiếng Việt", "lang_vi"),
			tgbotapi.NewInlineKeyboardButtonData("🇹🇭 ไทย", "lang_th"),
			tgbotapi.NewInlineKeyboardButtonData("🇵🇱 Polski", "lang_pl"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇳🇱 Nederlands", "lang_nl"),
			tgbotapi.NewInlineKeyboardButtonData("🇸🇪 Svenska", "lang_sv"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "🌐 *Select your language / Tilni tanlang:*")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

func (b *Bot) sendMainMenu(chatID int64, lang string) {
	if chatID == b.adminUserID {
		b.sendAdminPanel(chatID)
		return
	}
	b.sendUserMenu(chatID, lang)
}

func (b *Bot) sendUserMenu(chatID int64, lang string) {
	rows := [][]tgbotapi.KeyboardButton{
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(T(lang, "word_to_pdf")),
			tgbotapi.NewKeyboardButton(T(lang, "image_to_pdf")),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(T(lang, "excel_to_pdf")),
			tgbotapi.NewKeyboardButton(T(lang, "ppt_to_pdf")),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(T(lang, "compress_pdf")),
			tgbotapi.NewKeyboardButton(T(lang, "split_pdf")),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(T(lang, "merge_pdf")),
			tgbotapi.NewKeyboardButton(T(lang, "rotate_pdf")),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(T(lang, "watermark")),

			tgbotapi.NewKeyboardButton(T(lang, "unlock_pdf")),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(T(lang, "pdf_to_jpg")),
			tgbotapi.NewKeyboardButton("🔒 Protect PDF"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(T(lang, "my_coins")),
			tgbotapi.NewKeyboardButton(T(lang, "referral")),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(T(lang, "settings")),
			tgbotapi.NewKeyboardButton(T(lang, "partnership")),
		),
	}

	keyboard := tgbotapi.NewReplyKeyboard(rows...)
	keyboard.ResizeKeyboard = true

	msg := tgbotapi.NewMessage(chatID, T(lang, "main_menu"))
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

func (b *Bot) handleCommand(ctx context.Context, message *tgbotapi.Message) {
	lang := b.getUserLang(ctx, message.From.ID)
	cmd := message.Command()

	switch cmd {
	case "start":
		b.sendMainMenu(message.Chat.ID, lang)
	case "help":
		b.sendHelp(message.Chat.ID, lang, b.isAdmin(message.From.ID))
	case "menu":
		b.sendMainMenu(message.Chat.ID, lang)
	case "mycoins":
		b.showUserCoins(ctx, message.Chat.ID, message.From.ID, lang)
	case "ref":
		b.showReferralLink(message.Chat.ID, message.From.ID, lang)
	case "lang":
		b.sendLanguageSelection(message.Chat.ID)
	case "partnership":
		b.sendPartnershipInfo(message.Chat.ID, lang)

	case "compress":
		b.services.BotUser().SetCurrentAction(ctx, message.From.ID, "compress")
		b.sendMessage(message.Chat.ID, T(lang, "select_file"))
	case "merge":
		b.startMergeFlow(ctx, message.Chat.ID, message.From.ID)
	case "split":
		b.startSplitFlow(ctx, message.Chat.ID, message.From.ID)
	case "rotate":
		b.startRotateFlow(ctx, message.Chat.ID, message.From.ID)
	case "watermark":
		b.startWatermarkFlow(ctx, message.Chat.ID, message.From.ID)
	case "unlock":
		b.startUnlockFlow(ctx, message.Chat.ID, message.From.ID)
	case "word_to_pdf":
		b.services.BotUser().SetCurrentAction(ctx, message.From.ID, "word")
		b.sendMessage(message.Chat.ID, T(lang, "select_file"))
	case "pdf_to_word":
		b.sendMessage(message.Chat.ID, "⚠️ Bu funksiya tez orada qo'shiladi!")
	case "pdf_to_jpg":
		b.sendMessage(message.Chat.ID, "⚠️ Bu funksiya tez orada qo'shiladi!")
	case "jpg_to_pdf":
		b.startJPGToPDFFlow(ctx, message.Chat.ID, message.From.ID)

	case "stats", "cleanup", "addcoins", "top":
		if b.isAdmin(message.From.ID) {
			switch cmd {
			case "stats":
				b.handleAdminStats(ctx, message, lang)
			case "cleanup":
				b.handleAdminCleanup(ctx, message, lang)
			case "addcoins":
				b.handleAdminAddCoins(ctx, message, lang)
			case "top":
				b.handleAdminTop(ctx, message, lang)
			}
		} else {
			b.sendMainMenu(message.Chat.ID, lang)
		}

	case "id":
		b.sendMessage(message.Chat.ID, fmt.Sprintf("🆔 Sizning ID: `%d`", message.From.ID))

	default:
		b.sendMainMenu(message.Chat.ID, lang)
	}
}

func (b *Bot) sendHelp(chatID int64, lang string, isAdmin bool) {
	text := fmt.Sprintf(`📚 *%s*

/menu - %s
/mycoins - %s
/ref - %s
/lang - %s`,
		"Help", T(lang, "main_menu"), T(lang, "my_coins"), T(lang, "referral"), T(lang, "change_lang"))

	if isAdmin {
		text += `

🔐 *Admin:*
/stats - Statistics
/cleanup - Cleanup files
/addcoins @user 100
/top - Top 100 users`
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

func (b *Bot) showUserCoins(ctx context.Context, chatID, userID int64, lang string) {
	user, refCount, _ := b.services.BotUser().GetUserStats(ctx, userID)
	if user == nil {
		return
	}

	text := fmt.Sprintf(T(lang, "your_coins"), user.Coins, user.TotalUsed, refCount)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

func (b *Bot) showReferralLink(chatID, userID int64, lang string) {
	refLink := fmt.Sprintf("https://t.me/%s?start=ref%d", b.botUsername, userID)
	text := fmt.Sprintf(T(lang, "ref_link"), refLink)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

func (b *Bot) notifyReferral(ctx context.Context, chatID int64, newUser *tgbotapi.User, referrerID int64) {
	referrer, _, _ := b.services.BotUser().GetUserStats(ctx, referrerID)
	if referrer == nil {
		return
	}

	lang := b.getUserLang(ctx, newUser.ID)
	refLang := b.getUserLang(ctx, referrerID)

	name := referrer.FirstName
	if referrer.Username != "" {
		name = "@" + referrer.Username
	}

	b.sendMessage(chatID, fmt.Sprintf(T(lang, "ref_welcome"), name))
	b.sendMessage(referrerID, fmt.Sprintf(T(refLang, "ref_notify"), newUser.FirstName))
}

func (b *Bot) handleDocument(ctx context.Context, message *tgbotapi.Message) {
	lang := b.getUserLang(ctx, message.From.ID)
	userID := message.From.ID

	if _, loaded := b.processingUsers.LoadOrStore(userID, true); loaded {
		b.sendMessage(message.Chat.ID, T(lang, "processing_wait"))
		return
	}
	defer b.processingUsers.Delete(userID)

	if message.Document != nil && message.Document.FileSize > 50*1024*1024 {
		b.sendMessage(message.Chat.ID, "❌ Fayl hajmi juda katta! Maksimal 50MB.")
		return
	}

	canUse, coins, isPremium, _ := b.services.BotUser().CanUseService(ctx, userID)
	if !canUse {
		refLink := fmt.Sprintf("https://t.me/%s?start=ref%d", b.botUsername, userID)
		b.sendNoCoinsMessage(message.Chat.ID, lang, refLink)
		return
	}

	if !isPremium && coins <= 5 && coins > 0 {
		b.sendMessage(message.Chat.ID, fmt.Sprintf(T(lang, "coins_left"), coins))
	}

	user, _ := b.services.BotUser().GetUser(ctx, userID)
	action := ""
	if user != nil {
		action = user.CurrentAction
	}

	if state, ok := b.userStates.Load(userID); ok && state == StateMergeUpload {
		fileID, err := b.saveUploadedFile(ctx, message)
		if err != nil {
			b.sendMessage(message.Chat.ID, "❌ Faylni saqlashda xatolik: "+err.Error())
			return
		}

		var fileIDs []string
		if existing, ok := b.mergeContext.Load(userID); ok {
			fileIDs = existing.([]string)
		}
		fileIDs = append(fileIDs, fileID)
		b.mergeContext.Store(userID, fileIDs)

		b.sendMessage(message.Chat.ID, fmt.Sprintf("✅ Fayl qabul qilindi (%d-fayl).\nYana yuboring yoki *Done* tugmasini bosing.", len(fileIDs)))
		return
	}

	if state, ok := b.userStates.Load(userID); ok && state == StateSplitUpload {
		doc := message.Document
		ext := strings.ToLower(filepath.Ext(doc.FileName))
		if ext != ".pdf" {
			b.sendMessage(message.Chat.ID, "❌ Faqat PDF fayl yuboring!")
			return
		}

		fileID, err := b.saveUploadedFile(ctx, message)
		if err != nil {
			if strings.Contains(err.Error(), "file is too big") || strings.Contains(err.Error(), "limit 20MB") {
				b.sendMessage(message.Chat.ID, "❌ Fayl hajmi juda katta (Max: 20MB). Telegram Bot API cheklovi.")
			} else {
				b.sendMessage(message.Chat.ID, "❌ Faylni saqlashda xatolik: "+err.Error())
			}
			return
		}

		b.splitContext.Store(userID, fileID)
		b.userStates.Store(userID, StateSplitPages)
		b.sendMessage(message.Chat.ID, T(lang, "split_enter_pages"))
		return
	}

	if state, ok := b.userStates.Load(userID); ok && state == StateRotateUpload {
		doc := message.Document
		ext := strings.ToLower(filepath.Ext(doc.FileName))
		if ext != ".pdf" {
			b.sendMessage(message.Chat.ID, "❌ Faqat PDF fayl yuboring!")
			return
		}

		fileID, err := b.saveUploadedFile(ctx, message)
		if err != nil {
			b.sendMessage(message.Chat.ID, "❌ Faylni saqlashda xatolik: "+err.Error())
			return
		}

		b.sendRotateAngleSelection(message.Chat.ID, fileID, lang)
		b.userStates.Delete(userID)
		return
	}

	if state, ok := b.userStates.Load(userID); ok && state == StateWatermarkUpload {
		doc := message.Document
		ext := strings.ToLower(filepath.Ext(doc.FileName))
		if ext != ".pdf" {
			b.sendMessage(message.Chat.ID, "❌ Faqat PDF fayl yuboring!")
			return
		}

		fileID, err := b.saveUploadedFile(ctx, message)
		if err != nil {
			b.sendMessage(message.Chat.ID, "❌ Faylni saqlashda xatolik: "+err.Error())
			return
		}

		b.splitContext.Store(userID, fileID)
		b.userStates.Store(userID, StateWatermarkText)
		b.sendMessage(message.Chat.ID, "💧 Suv belgisi uchun matn kiriting (masalan: CONFIDENTIAL):")
		return
	}

	if state, ok := b.userStates.Load(userID); ok && state == StateUnlockUpload {
		doc := message.Document
		ext := strings.ToLower(filepath.Ext(doc.FileName))
		if ext != ".pdf" {
			b.sendMessage(message.Chat.ID, "❌ Faqat PDF fayl yuboring!")
			return
		}

		fileID, err := b.saveUploadedFile(ctx, message)
		if err != nil {
			b.sendMessage(message.Chat.ID, "❌ Faylni saqlashda xatolik: "+err.Error())
			return
		}

		b.userStates.Delete(userID)
		go b.processUnlock(ctx, message.Chat.ID, userID, fileID, lang)
		return
	}

	if state, ok := b.userStates.Load(userID); ok && state == StatePDFToJPGUpload {
		doc := message.Document
		ext := strings.ToLower(filepath.Ext(doc.FileName))
		if ext != ".pdf" {
			b.sendMessage(message.Chat.ID, "❌ Faqat PDF fayl yuboring!")
			return
		}

		fileID, err := b.saveUploadedFile(ctx, message)
		if err != nil {
			b.sendMessage(message.Chat.ID, "❌ Faylni saqlashda xatolik: "+err.Error())
			return
		}

		b.userStates.Delete(userID)
		go b.processPDFToJPG(ctx, message.Chat.ID, userID, fileID, lang)
		return
	}

	if state, ok := b.userStates.Load(userID); ok && state == StateProtectUpload {
		doc := message.Document
		ext := strings.ToLower(filepath.Ext(doc.FileName))
		if ext != ".pdf" {
			b.sendMessage(message.Chat.ID, "❌ Faqat PDF fayl yuboring!")
			return
		}

		fileID, err := b.saveUploadedFile(ctx, message)
		if err != nil {
			b.sendMessage(message.Chat.ID, "❌ Faylni saqlashda xatolik: "+err.Error())
			return
		}

		b.splitContext.Store(userID, fileID)
		b.userStates.Store(userID, StateProtectPassword)
		b.sendMessage(message.Chat.ID, "🔒 PDF ga qo'yiladigan parolni kiriting:")
		return
	}

	if state, ok := b.userStates.Load(userID); ok && state == StateWordToPDFUpload {
		doc := message.Document
		ext := strings.ToLower(filepath.Ext(doc.FileName))
		if ext == ".doc" || ext == ".docx" || ext == ".odt" || ext == ".rtf" {
			b.userStates.Delete(userID)
			b.processFile(ctx, message, "word", lang)
			return
		}
		b.userStates.Delete(userID)
		action = ""
	}

	if state, ok := b.userStates.Load(userID); ok && state == StateExcelToPDFUpload {
		doc := message.Document
		ext := strings.ToLower(filepath.Ext(doc.FileName))
		if ext == ".xls" || ext == ".xlsx" {
			b.userStates.Delete(userID)
			b.processFile(ctx, message, "excel", lang)
			return
		}
		b.userStates.Delete(userID)
		action = ""
	}

	if state, ok := b.userStates.Load(userID); ok && state == StatePowerPointToPDFUpload {
		doc := message.Document
		ext := strings.ToLower(filepath.Ext(doc.FileName))
		if ext == ".ppt" || ext == ".pptx" {
			b.userStates.Delete(userID)
			b.processFile(ctx, message, "ppt", lang)
			return
		}
		b.userStates.Delete(userID)
		action = ""
	}

	if state, ok := b.userStates.Load(userID); ok && state == StateJPGToPDFUpload {
		doc := message.Document
		ext := strings.ToLower(filepath.Ext(doc.FileName))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
			b.userStates.Delete(userID)
			b.processFile(ctx, message, "image", lang)
			return
		}
		b.userStates.Delete(userID)
		action = ""
	}

	doc := message.Document
	ext := strings.ToLower(filepath.Ext(doc.FileName))

	if action == "" {
		switch ext {
		case ".pdf":
			action = "compress"
		case ".doc", ".docx", ".odt", ".rtf":
			action = "word"
		case ".xls", ".xlsx":
			action = "excel"
		case ".ppt", ".pptx":
			action = "ppt"
		case ".jpg", ".jpeg", ".png":
			action = "image"
		default:
			b.sendMessage(message.Chat.ID, T(lang, "error"))
			return
		}
	}

	b.sendMessage(message.Chat.ID, T(lang, "file_received"))

	b.processFile(ctx, message, action, lang)

	b.services.BotUser().SetCurrentAction(ctx, userID, "")
}

func (b *Bot) processFile(ctx context.Context, message *tgbotapi.Message, action, lang string) {
	fileID, err := b.saveUploadedFile(ctx, message)
	if err != nil {
		b.log.Error("saveUploadedFile failed", logger.Error(err))
		b.sendMessage(message.Chat.ID, T(lang, "error")+" (upload)")
		return
	}

	b.log.Info("File saved", logger.String("fileID", fileID), logger.String("action", action))

	b.api.Send(tgbotapi.NewChatAction(message.Chat.ID, tgbotapi.ChatUploadDocument))

	var outputFileID *string

	switch action {
	case "compress":
		req := models.CompressRequest{InputFileID: fileID, Compression: "medium"}
		jobID, err := b.services.Compress().Create(ctx, req, nil)
		if err != nil {
			b.log.Error("Compress.Create failed", logger.Error(err))
			b.sendMessage(message.Chat.ID, T(lang, "error")+" (compress)")
			return
		}
		job, err := b.services.Compress().GetByID(ctx, jobID)
		if err != nil {
			b.log.Error("Compress.GetByID failed", logger.Error(err))
		}
		if job != nil {
			outputFileID = job.OutputFileID
		}
	case "word":
		req := models.WordToPDFRequest{InputFileID: fileID}
		jobID, err := b.services.WordToPDF().Create(ctx, req, nil)
		if err != nil {
			b.log.Error("WordToPDF.Create failed", logger.Error(err))
			b.sendMessage(message.Chat.ID, T(lang, "error")+" (word)")
			return
		}
		job, _ := b.services.WordToPDF().GetByID(ctx, jobID)
		if job != nil {
			outputFileID = job.OutputFileID
		}
	case "excel":
		req := models.ExcelToPDFRequest{InputFileID: fileID}
		jobID, err := b.services.ExcelToPDF().Create(ctx, req, nil)
		if err != nil {
			b.log.Error("ExcelToPDF.Create failed", logger.Error(err))
			b.sendMessage(message.Chat.ID, T(lang, "error")+" (excel)")
			return
		}
		job, _ := b.services.ExcelToPDF().GetByID(ctx, jobID)
		if job != nil {
			outputFileID = job.OutputFileID
		}
	case "ppt":
		req := models.PowerPointToPDFRequest{InputFileID: fileID}
		jobID, err := b.services.PowerPointToPDF().Create(ctx, req, nil)
		if err != nil {
			b.log.Error("PowerPointToPDF.Create failed", logger.Error(err))
			b.sendMessage(message.Chat.ID, T(lang, "error")+" (ppt)")
			return
		}
		job, _ := b.services.PowerPointToPDF().GetByID(ctx, jobID)
		if job != nil {
			outputFileID = job.OutputFileID
		}
	case "image":
		jobID, err := b.services.JPGToPDF().CreateJob(ctx, nil, []string{fileID})
		if err != nil {
			b.log.Error("JPGToPDF.CreateJob failed", logger.Error(err))
			b.sendMessage(message.Chat.ID, T(lang, "error")+" (image)")
			return
		}
		job, _ := b.services.JPGToPDF().GetJobByID(ctx, jobID)
		if job != nil {
			outputFileID = job.OutputFileID
		}
	}

	b.sendResult(message.Chat.ID, message.From.ID, outputFileID, lang)
}

func (b *Bot) sendResult(chatID, userID int64, outputFileID *string, lang string) {
	if outputFileID == nil {
		b.sendMessage(chatID, T(lang, "error"))
		return
	}

	isAdmin := userID == b.adminUserID
	if isAdmin {
		b.services.BotUser().IncrementUsage(context.Background(), userID)
	} else {
		err := b.services.BotUser().UseService(context.Background(), userID)
		if err != nil {
			if err.Error() == "insufficient funds" {
				b.sendNoCoinsMessage(chatID, lang, fmt.Sprintf("https://t.me/%s?start=%d", b.api.Self.UserName, userID))
			} else {
				b.sendMessage(chatID, T(lang, "error"))
			}
			return
		}
	}

	outputFile, err := b.services.File().GetByID(context.Background(), *outputFileID)
	if err != nil {
		b.sendMessage(chatID, T(lang, "error"))
		return
	}

	fi, _ := os.Stat(outputFile.FilePath)
	sizeKB := float64(fi.Size()) / 1024
	caption := fmt.Sprintf("%s\n"+T(lang, "file_size"), T(lang, "success"), sizeKB)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(T(lang, "back_to_menu"), "menu_back"),
		),
	)

	doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(outputFile.FilePath))
	doc.Caption = caption
	doc.ReplyMarkup = keyboard
	b.api.Send(doc)
}

func (b *Bot) handleAdminStats(ctx context.Context, message *tgbotapi.Message, lang string) {
	if !b.isAdmin(message.From.ID) {
		b.sendMessage(message.Chat.ID, T(lang, "admin_only"))
		return
	}

	stats, _ := b.services.PublicStats().GetStats(ctx)
	pending, _ := b.services.File().ListPendingDeletionFiles(ctx, 5)

	date := time.Now().Format("02.01.2006 15:04:05")
	text := fmt.Sprintf(`📊 *Statistics*

👥 Users: *%d*
🛠️ Tools: *%d*
📈 Total jobs: *%d*
🗑️ Pending cleanup: *%d* files
♻️ Auto-cleanup: *Active (every 5 mins)*
🕒 Server time: *%s*`,
		stats.TotalUsers, stats.ToolsCount, stats.TotalUsage, len(pending), date)

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

func (b *Bot) handleAdminCleanup(ctx context.Context, message *tgbotapi.Message, lang string) {
	if !b.isAdmin(message.From.ID) {
		b.sendMessage(message.Chat.ID, T(lang, "admin_only"))
		return
	}

	files, _ := b.services.File().ListPendingDeletionFiles(ctx, 0)
	count := 0
	for _, f := range files {
		os.Remove(f.FilePath)
		b.services.File().Delete(ctx, f.ID)
		count++
	}

	b.sendMessage(message.Chat.ID, fmt.Sprintf("✅ Cleaned *%d* files", count))
}

func (b *Bot) handleAdminAddCoins(ctx context.Context, message *tgbotapi.Message, lang string) {
	if !b.isAdmin(message.From.ID) {
		b.sendMessage(message.Chat.ID, T(lang, "admin_only"))
		return
	}

	args := strings.Fields(message.CommandArguments())
	if len(args) < 2 {
		b.sendMessage(message.Chat.ID, "Format: /addcoins @username 100")
		return
	}

	username := strings.TrimPrefix(args[0], "@")
	amount, _ := strconv.Atoi(args[1])
	if amount <= 0 {
		amount = 10
	}

	user, _ := b.services.BotUser().AddCoinsByUsername(ctx, username, amount)
	if user == nil {
		b.sendMessage(message.Chat.ID, T(lang, "user_not_found"))
		return
	}

	b.sendMessage(message.Chat.ID, fmt.Sprintf(T(lang, "coins_added"), username, amount, user.Coins))
	b.sendMessage(user.TelegramID, fmt.Sprintf(T(b.getUserLang(ctx, user.TelegramID), "coins_received"), amount, user.Coins))
}

func (b *Bot) handleAdminTop(ctx context.Context, message *tgbotapi.Message, lang string) {
	if !b.isAdmin(message.From.ID) {
		b.sendMessage(message.Chat.ID, T(lang, "admin_only"))
		return
	}

	users, _ := b.services.BotUser().GetTopUsers(ctx, 20)

	text := "👥 *Top 20 Active Users*\n\n"
	for i, u := range users {
		name := u.FirstName
		if u.Username != "" {
			name = "@" + u.Username
		}
		bar := strings.Repeat("█", min(u.TotalUsed/5, 10))
		text += fmt.Sprintf("%d. %s\n   📊 %d jobs %s\n", i+1, name, u.TotalUsed, bar)
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

func (b *Bot) ensureUser(ctx context.Context, user *tgbotapi.User, refID *int64) {
	username := user.UserName
	b.services.BotUser().EnsureUser(ctx, user.ID, username, user.FirstName, refID)
}

func (b *Bot) handlePhoto(ctx context.Context, message *tgbotapi.Message) {
	lang := b.getUserLang(ctx, message.From.ID)
	userID := message.From.ID

	if state, ok := b.userStates.Load(userID); !ok || state != StateJPGToPDFUpload {
		b.startJPGToPDFFlow(ctx, message.Chat.ID, userID)
	}

	if _, loaded := b.processingUsers.LoadOrStore(userID, true); loaded {
		b.sendMessage(message.Chat.ID, T(lang, "processing_wait"))
		return
	}
	defer b.processingUsers.Delete(userID)

	canUse, coins, isPremium, _ := b.services.BotUser().CanUseService(ctx, userID)
	if !canUse {
		refLink := fmt.Sprintf("https://t.me/%s?start=ref%d", b.botUsername, userID)
		b.sendNoCoinsMessage(message.Chat.ID, lang, refLink)
		return
	}
	if !isPremium && coins <= 5 && coins > 0 {
		b.sendMessage(message.Chat.ID, fmt.Sprintf(T(lang, "coins_left"), coins))
	}

	// b.sendMessage(message.Chat.ID, T(lang, "file_received"))

	photos := message.Photo
	photo := photos[len(photos)-1]

	fileID, err := b.savePhoto(ctx, photo.FileID, message.From.ID)
	if err != nil {
		b.log.Error("savePhoto failed", logger.Error(err))
		b.sendMessage(message.Chat.ID, T(lang, "error")+" (photo)")
		return
	}

	b.log.Info("Photo saved", logger.String("fileID", fileID))

	var fileIDs []string
	if list, ok := b.jpgContext.Load(userID); ok {
		fileIDs = list.([]string)
	}
	fileIDs = append(fileIDs, fileID)
	b.jpgContext.Store(userID, fileIDs)

	count := len(fileIDs)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Tayyor (Done)", "jpg_done"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Bekor qilish", "jpg_cancel"),
		),
	)

	msgText := fmt.Sprintf("📸 *%d ta rasm qabul qilindi.*\n\nYana rasm yuboring yoki barchasini PDF qilish uchun *Tayyor* tugmasini bosing.", count)
	if lang != "uz" {
		msgText = fmt.Sprintf("📸 *%d photos received.*\n\nSend more photos or click *Done* to convert all to PDF.", count)
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

func (b *Bot) startJPGToPDFFlow(ctx context.Context, chatID, userID int64) {
	b.services.BotUser().SetCurrentAction(ctx, userID, "image")
	b.userStates.Store(userID, StateJPGToPDFUpload)
	b.jpgContext.Delete(userID)
	lang := b.getUserLang(ctx, userID)
	b.sendMessage(chatID, T(lang, "select_file")+" (JPG, PNG)")
}

func (b *Bot) finishJPGToPDFFlow(ctx context.Context, chatID, userID int64) {
	lang := b.getUserLang(ctx, userID)

	if _, loaded := b.processingUsers.LoadOrStore(userID, true); loaded {
		b.sendMessage(chatID, T(lang, "processing_wait"))
		return
	}
	defer b.processingUsers.Delete(userID)

	list, ok := b.jpgContext.Load(userID)
	if !ok || list == nil {
		b.sendMessage(chatID, "❌ Fayllar topilmadi / No files found")
		return
	}
	fileIDs := list.([]string)
	if len(fileIDs) == 0 {
		b.sendMessage(chatID, "❌ Fayllar topilmadi / No files found")
		return
	}

	msgWait := tgbotapi.NewMessage(chatID, T(lang, "file_received")) // "Ishlanmoqda..."
	b.api.Send(msgWait)

	jobID, err := b.services.JPGToPDF().CreateJob(ctx, nil, fileIDs)
	if err != nil {
		b.log.Error("JPGToPDF failed", logger.Error(err))
		b.sendMessage(chatID, T(lang, "error")+" (convert)")
		return
	}

	job, _ := b.services.JPGToPDF().GetJobByID(ctx, jobID)
	if job == nil || job.OutputFileID == nil {
		b.sendMessage(chatID, T(lang, "error"))
		return
	}

	// Clean up
	b.jpgContext.Delete(userID)
	b.userStates.Delete(userID)
	b.services.BotUser().SetCurrentAction(ctx, userID, "")

	b.sendResult(chatID, userID, job.OutputFileID, lang)
}

func (b *Bot) savePhoto(ctx context.Context, telegramFileID string, userID int64) (string, error) {
	filePath, err := b.downloadFile(telegramFileID, "photo.jpg")
	if err != nil {
		return "", err
	}

	fi, _ := os.Stat(filePath)
	fileID := uuid.NewString()

	file := models.File{
		ID:         fileID,
		FileName:   "photo.jpg",
		FilePath:   filePath,
		FileType:   "image/jpeg",
		FileSize:   fi.Size(),
		UploadedAt: time.Now(),
	}

	b.services.File().Save(ctx, file)
	return fileID, nil
}

func (b *Bot) saveUploadedFile(ctx context.Context, message *tgbotapi.Message) (string, error) {
	filePath, err := b.downloadFile(message.Document.FileID, message.Document.FileName)
	if err != nil {
		return "", err
	}

	fi, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %w", err)
	}
	fileID := uuid.NewString()

	file := models.File{
		ID:         fileID,
		FileName:   message.Document.FileName,
		FilePath:   filePath,
		FileType:   message.Document.MimeType,
		FileSize:   fi.Size(),
		UploadedAt: time.Now(),
	}

	b.services.File().Save(ctx, file)
	return fileID, nil
}

func (b *Bot) downloadFile(fileID, fileName string) (string, error) {
	file, err := b.api.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		if strings.Contains(err.Error(), "file is too big") {
			return "", fmt.Errorf("file is too big (limit 20MB)")
		}
		return "", err
	}

	resp, err := http.Get(file.Link(b.api.Token))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	dir := filepath.Join("storage", "bot_uploads")
	os.MkdirAll(dir, os.ModePerm)

	filePath := filepath.Join(dir, uuid.NewString()+"_"+fileName)
	out, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	io.Copy(out, resp.Body)
	return filePath, nil
}

func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (b *Bot) sendAdminPanel(chatID int64) {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📊 Global Stats"),
			tgbotapi.NewKeyboardButton("➕ Add Coins"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("👥 Users List"),
			tgbotapi.NewKeyboardButton("� Backup"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("↩️ User Mode"),
		),
	)
	keyboard.ResizeKeyboard = true

	msg := tgbotapi.NewMessage(chatID, "👨‍⚖️ Admin Panel")
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

func (b *Bot) sendNoCoinsMessage(chatID int64, lang string, refLink string) {
	text := T(lang, "no_coins") + "\n\n📎 *Referal:* " + refLink

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(T(lang, "buy_coins"), "buy_coins"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(T(lang, "referral"), "referral"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(T(lang, "back_to_menu"), "menu_back"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

func (b *Bot) sendPartnershipInfo(chatID int64, lang string) {
	text := T(lang, "partnership_msg")

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("📩 "+T(lang, "contact_admin"), "https://t.me/zarifjorayev"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(T(lang, "back_to_menu"), "menu_back"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

func (b *Bot) handleAdminStates(ctx context.Context, update *tgbotapi.Update, state string) {
	userID := update.Message.From.ID
	text := update.Message.Text

	if text == "/cancel" || text == "↩️ User Mode" {
		b.userStates.Delete(userID)
		b.adminContext.Delete(userID)
		b.sendAdminPanel(update.Message.Chat.ID)
		return
	}

	switch state {
	case StateAddCoinsUser:
		var targetUser *models.BotUser
		var err error

		if strings.HasPrefix(text, "@") {
			username := strings.TrimPrefix(text, "@")
			targetUser, err = b.services.BotUser().AddCoinsByUsername(ctx, username, 0)
		} else {
			id, _ := strconv.ParseInt(text, 10, 64)
			if id > 0 {
				targetUser, err = b.services.BotUser().GetUser(ctx, id)
			}
		}

		if err != nil {
			b.sendMessage(update.Message.Chat.ID, "❌ Xatolik: "+err.Error())
			return
		}
		if targetUser == nil {
			b.sendMessage(update.Message.Chat.ID, "❌ Foydalanuvchi topilmadi. Qayta urinib ko'ring yoki /cancel:")
			return
		}

		b.adminContext.Store(userID, targetUser)
		b.userStates.Store(userID, StateAddCoinsAmount)
		b.sendMessage(update.Message.Chat.ID, fmt.Sprintf("✅ Foydalanuvchi: %s (ID: %d)\n💰 Qancha coin qo'shmoqchisiz?", targetUser.FirstName, targetUser.TelegramID))

	case StateAddCoinsAmount:
		amount, err := strconv.Atoi(text)
		if err != nil || amount <= 0 {
			b.sendMessage(update.Message.Chat.ID, "❌ Raqam kiriting (musbat):")
			return
		}

		val, ok := b.adminContext.Load(userID)
		if !ok {
			b.userStates.Delete(userID)
			b.sendMessage(update.Message.Chat.ID, "❌ Sessiya eskirgan")
			return
		}
		targetUser := val.(*models.BotUser)

		err = b.services.BotUser().AddCoins(ctx, targetUser.TelegramID, amount)
		if err != nil {
			b.sendMessage(update.Message.Chat.ID, "❌ Xatolik: "+err.Error())
			return
		}

		b.sendMessage(update.Message.Chat.ID, fmt.Sprintf("✅ Muvaffaqiyatli! %d coin qo'shildi.", amount))
		b.sendMessage(targetUser.TelegramID, fmt.Sprintf("🎁 Sizga admin tomonidan %d coin berildi!", amount))

		b.userStates.Delete(userID)
		b.adminContext.Delete(userID)
		b.sendAdminPanel(update.Message.Chat.ID)
	}
}

func (b *Bot) startMergeFlow(ctx context.Context, chatID, userID int64) {
	b.userStates.Store(userID, StateMergeUpload)
	b.mergeContext.Delete(userID)

	lang := b.getUserLang(ctx, userID)

	cancelBtnText := "❌ Bekor qilish"
	if userID == b.adminUserID {
		cancelBtnText = "↩️ User Mode"
	}

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Done"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(cancelBtnText),
		),
	)

	msg := tgbotapi.NewMessage(chatID, T(lang, "merge_upload_prompt"))
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

func (b *Bot) finishMergeFlow(ctx context.Context, chatID, userID int64) {
	filesInterface, ok := b.mergeContext.Load(userID)
	if !ok {
		b.sendMessage(chatID, "Hech qanday fayl yuklanmadi.")
		return
	}
	fileIDs := filesInterface.([]string)

	if len(fileIDs) < 2 {
		b.sendMessage(chatID, "Birlashtirish uchun kamida 2 ta fayl kerak!")
		return
	}

	b.sendMessage(chatID, "⏳ Birlashtirilmoqda...")
	b.api.Send(tgbotapi.NewChatAction(chatID, tgbotapi.ChatUploadDocument))

	job := models.MergeJob{
		UserID:       userID,
		InputFileIDs: fileIDs,
		Status:       "pending",
	}

	jobID, err := b.services.Merge().Create(ctx, job)
	if err != nil {
		b.log.Error("Failed to create merge job", logger.Error(err))
		b.sendMessage(chatID, "❌ Ish yaratishda xatolik: "+err.Error())
		return
	}

	select {
	case b.workerPool <- struct{}{}:
		defer func() { <-b.workerPool }()
	default:
		b.sendMessage(chatID, "⚠️ Server ayni damda band. Iltimos 1 daqiqadan so'ng urinib ko'ring.")
		return
	}

	resultFile, err := b.services.Merge().MergeFiles(ctx, jobID)
	if err != nil {
		b.log.Error("Failed to merge files", logger.Error(err))
		b.sendMessage(chatID, "Birlashtirishda xatolik: "+err.Error())
		return
	}

	lang := b.getUserLang(ctx, userID)
	b.sendResult(chatID, userID, &resultFile.ID, lang)

	b.userStates.Delete(userID)
	b.mergeContext.Delete(userID)
	b.sendMainMenu(chatID, lang)
}

func (b *Bot) startSplitFlow(ctx context.Context, chatID, userID int64) {
	lang := b.getUserLang(ctx, userID)

	b.userStates.Store(userID, StateSplitUpload)
	b.splitContext.Delete(userID)

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("↩️ " + T(lang, "back_to_menu")),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "📄 Bo'lish uchun PDF faylni yuboring:")
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

func (b *Bot) finishSplitFlow(ctx context.Context, chatID, userID int64, pagesInput string) {
	lang := b.getUserLang(ctx, userID)

	if pagesInput == "↩️ "+T(lang, "back_to_menu") || pagesInput == "/cancel" {
		b.userStates.Delete(userID)
		b.splitContext.Delete(userID)
		b.sendMainMenu(chatID, lang)
		return
	}

	fileIDInterface, ok := b.splitContext.Load(userID)
	if !ok {
		b.sendMessage(chatID, "❌ Avval PDF fayl yuboring!")
		b.userStates.Store(userID, StateSplitUpload)
		return
	}
	fileID := fileIDInterface.(string)

	req := models.CreateSplitJobRequest{
		InputFileID: fileID,
		SplitRanges: pagesInput,
	}

	select {
	case b.workerPool <- struct{}{}:
		defer func() { <-b.workerPool }()
	default:
		b.sendMessage(chatID, "⚠️ Server band. Qayta urinib ko'ring.")
		return
	}

	jobID, err := b.services.Split().Create(ctx, req, nil)

	if err != nil {
		b.log.Error("Split failed", logger.Error(err))
		b.sendMessage(chatID, "❌ Bo'lishda xatolik: "+err.Error())
		b.userStates.Delete(userID)
		b.splitContext.Delete(userID)
		b.sendMainMenu(chatID, lang)
		return
	}

	job, err := b.services.Split().GetByID(ctx, jobID)
	if err != nil || job == nil || len(job.OutputFileIDs) == 0 {
		b.sendMessage(chatID, "❌ Natija topilmadi")
		b.userStates.Delete(userID)
		b.splitContext.Delete(userID)
		b.sendMainMenu(chatID, lang)
		return
	}

	b.services.BotUser().UseService(ctx, userID)

	for _, outFileID := range job.OutputFileIDs {
		b.sendResult(chatID, userID, &outFileID, lang)
	}

	b.sendMessage(chatID, fmt.Sprintf(T(lang, "split_success"), len(job.OutputFileIDs)))

	b.userStates.Delete(userID)
	b.splitContext.Delete(userID)
	b.sendMainMenu(chatID, lang)
}

func (b *Bot) sendBuyCoinsMenu(chatID int64, lang string) {
	text := "💰 *Coin Sotib Olish*\n\n" +
		"⭐ 1 Star = 1 Coin = 1 Job\n\n" +
		"Tugmani bosing:"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⭐ 1 Star = 1 Coin", "buy_1"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(T(lang, "back_to_menu"), "menu_back"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

func (b *Bot) sendStarsInvoice(chatID, userID int64, lang string, stars, coins int, paymentType string) {
	title := fmt.Sprintf("⭐ %d Star = %d Coin", stars, coins)
	description := fmt.Sprintf("%d ta coin sotib olish (1 coin = 1 job)", coins)
	payload := fmt.Sprintf("%s:%d:%d", paymentType, userID, coins)

	invoice := tgbotapi.InvoiceConfig{
		BaseChat:    tgbotapi.BaseChat{ChatID: chatID},
		Title:       title,
		Description: description,
		Payload:     payload,
		Currency:    "XTR",
		Prices: []tgbotapi.LabeledPrice{
			{
				Label:  title,
				Amount: stars,
			},
		},
	}

	_, err := b.api.Send(invoice)
	if err != nil {
		b.log.Error("Failed to send invoice", logger.Error(err))
		b.sendMessage(chatID, T(lang, "error"))
	}
}

func (b *Bot) handlePreCheckout(query *tgbotapi.PreCheckoutQuery) {
	callback := tgbotapi.PreCheckoutConfig{
		PreCheckoutQueryID: query.ID,
		OK:                 true,
	}
	b.api.Request(callback)
	b.log.Info("PreCheckout approved",
		logger.Int64("user_id", query.From.ID),
		logger.String("payload", query.InvoicePayload))
}

func (b *Bot) handleSuccessfulPayment(ctx context.Context, message *tgbotapi.Message) {
	payment := message.SuccessfulPayment
	userID := message.From.ID
	lang := b.getUserLang(ctx, userID)

	parts := strings.Split(payment.InvoicePayload, ":")
	if len(parts) < 3 {
		b.log.Error("Invalid payment payload", logger.String("payload", payment.InvoicePayload))
		return
	}

	coins, err := strconv.Atoi(parts[2])
	if err != nil {
		b.log.Error("Failed to parse coins", logger.Error(err))
		return
	}

	err = b.services.BotUser().AddCoins(ctx, userID, coins)
	if err != nil {
		b.log.Error("Failed to add coins", logger.Error(err))
		b.sendMessage(message.Chat.ID, "❌ Xatolik yuz berdi. Iltimos admin bilan bog'laning.")
		return
	}

	user, _ := b.services.BotUser().GetUser(ctx, userID)
	newBalance := 0
	if user != nil {
		newBalance = user.Coins
	}

	b.log.Info("Payment successful",
		logger.Int64("user_id", userID),
		logger.Int("coins_added", coins),
		logger.Int("new_balance", newBalance),
		logger.Int("stars_paid", payment.TotalAmount))

	text := fmt.Sprintf("✅ *To'lov muvaffaqiyatli!*\n\n"+
		"💰 Qo'shildi: *%d* coin\n"+
		"💳 Yangi balans: *%d* coin\n\n"+
		"Rahmat! Endi PDF amallarini bajarishingiz mumkin.", coins, newBalance)

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	b.api.Send(msg)

	b.sendMainMenu(message.Chat.ID, lang)
}

func (b *Bot) startRotateFlow(ctx context.Context, chatID, userID int64) {
	lang := b.getUserLang(ctx, userID)

	b.userStates.Store(userID, StateRotateUpload)

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("↩️ " + T(lang, "back_to_menu")),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "🔄 *Rotate PDF*\n\nAylantirish uchun PDF faylni yuboring:")
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

func (b *Bot) sendRotateAngleSelection(chatID int64, fileID, lang string) {
	text := "🔄 *Aylantirish burchagini tanlang:*"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("↪️ 90°", fmt.Sprintf("rotate:%s:90", fileID)),
			tgbotapi.NewInlineKeyboardButtonData("🔃 180°", fmt.Sprintf("rotate:%s:180", fileID)),
			tgbotapi.NewInlineKeyboardButtonData("↩️ 270°", fmt.Sprintf("rotate:%s:270", fileID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(T(lang, "back_to_menu"), "menu_back"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

func (b *Bot) processRotate(ctx context.Context, chatID, userID int64, fileID string, angle int, lang string) {
	canUse, coins, isPremium, _ := b.services.BotUser().CanUseService(ctx, userID)
	if !canUse {
		refLink := fmt.Sprintf("https://t.me/%s?start=ref%d", b.botUsername, userID)
		b.sendNoCoinsMessage(chatID, lang, refLink)
		return
	}
	if !isPremium && coins <= 5 && coins > 0 {
		b.sendMessage(chatID, fmt.Sprintf(T(lang, "coins_left"), coins))
	}

	b.sendMessage(chatID, "⏳ Aylantirilmoqda...")
	b.api.Send(tgbotapi.NewChatAction(chatID, tgbotapi.ChatUploadDocument))

	req := models.RotateRequest{
		InputFileID: fileID,
		Angle:       angle,
		Pages:       "all",
	}

	jobID, err := b.services.Rotate().Create(ctx, req, nil)
	if err != nil {
		b.log.Error("Rotate failed", logger.Error(err))
		b.sendMessage(chatID, "❌ Aylantirishda xatolik: "+err.Error())
		b.sendMainMenu(chatID, lang)
		return
	}

	job, err := b.services.Rotate().GetByID(ctx, jobID)
	if err != nil || job == nil || job.OutputFileID == nil {
		b.sendMessage(chatID, "❌ Natija topilmadi")
		b.sendMainMenu(chatID, lang)
		return
	}

	b.services.BotUser().UseService(ctx, userID)

	b.sendResult(chatID, userID, job.OutputFileID, lang)
	b.sendMessage(chatID, fmt.Sprintf("✅ PDF %d° ga aylantirildi!", angle))
	b.sendMainMenu(chatID, lang)
}

func (b *Bot) startWatermarkFlow(ctx context.Context, chatID, userID int64) {
	lang := b.getUserLang(ctx, userID)

	b.userStates.Store(userID, StateWatermarkUpload)

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("↩️ " + T(lang, "back_to_menu")),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "💧 *Watermark*\n\nSuv belgisi qo'shish uchun PDF faylni yuboring:")
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

func (b *Bot) processWatermark(ctx context.Context, chatID, userID int64, fileID, text, lang string) {
	canUse, coins, isPremium, _ := b.services.BotUser().CanUseService(ctx, userID)
	if !canUse {
		refLink := fmt.Sprintf("https://t.me/%s?start=ref%d", b.botUsername, userID)
		b.sendNoCoinsMessage(chatID, lang, refLink)
		return
	}
	if !isPremium && coins <= 5 && coins > 0 {
		b.sendMessage(chatID, fmt.Sprintf(T(lang, "coins_left"), coins))
	}

	b.sendMessage(chatID, "⏳ Suv belgisi qo'shilmoqda...")
	b.api.Send(tgbotapi.NewChatAction(chatID, tgbotapi.ChatUploadDocument))

	req := models.WatermarkRequest{
		InputFileID: fileID,
		Text:        text,
		FontSize:    48,
		Position:    "diagonal",
		Opacity:     0.3,
	}

	jobID, err := b.services.Watermark().Create(ctx, req, nil)
	if err != nil {
		b.log.Error("Watermark failed", logger.Error(err))
		b.sendMessage(chatID, "❌ Xatolik: "+err.Error())
		b.sendMainMenu(chatID, lang)
		return
	}

	job, err := b.services.Watermark().GetByID(ctx, jobID)
	if err != nil || job == nil || job.OutputFileID == nil {
		b.sendMessage(chatID, "❌ Natija topilmadi")
		b.sendMainMenu(chatID, lang)
		return
	}

	b.services.BotUser().UseService(ctx, userID)
	b.sendResult(chatID, userID, job.OutputFileID, lang)
	b.sendMessage(chatID, "✅ Suv belgisi qo'shildi!")
	b.sendMainMenu(chatID, lang)
}

func (b *Bot) startUnlockFlow(ctx context.Context, chatID, userID int64) {
	lang := b.getUserLang(ctx, userID)

	b.services.BotUser().SetCurrentAction(ctx, userID, "unlock")

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("↩️ " + T(lang, "back_to_menu")),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "🔓 *Unlock PDF*\n\nParolni olib tashlash uchun PDF faylni yuboring:")
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

func (b *Bot) processUnlock(ctx context.Context, chatID, userID int64, fileID, lang string) {
	canUse, coins, isPremium, _ := b.services.BotUser().CanUseService(ctx, userID)
	if !canUse {
		refLink := fmt.Sprintf("https://t.me/%s?start=ref%d", b.botUsername, userID)
		b.sendNoCoinsMessage(chatID, lang, refLink)
		return
	}
	if !isPremium && coins <= 5 && coins > 0 {
		b.sendMessage(chatID, fmt.Sprintf(T(lang, "coins_left"), coins))
	}

	b.sendMessage(chatID, "⏳ Parol olib tashlanmoqda...")
	b.api.Send(tgbotapi.NewChatAction(chatID, tgbotapi.ChatUploadDocument))

	req := models.UnlockRequest{
		InputFileID: fileID,
		Password:    "",
	}

	jobID, err := b.services.Unlock().Create(ctx, req, nil)
	if err != nil {
		b.log.Error("Unlock failed", logger.Error(err))
		b.sendMessage(chatID, "❌ Xatolik: "+err.Error())
		b.sendMainMenu(chatID, lang)
		return
	}

	job, err := b.services.Unlock().GetByID(ctx, jobID)
	if err != nil || job == nil || job.OutputFileID == nil {
		b.sendMessage(chatID, "❌ Natija topilmadi")
		b.sendMainMenu(chatID, lang)
		return
	}

	b.services.BotUser().UseService(ctx, userID)
	b.sendResult(chatID, userID, job.OutputFileID, lang)
	b.sendMessage(chatID, "✅ PDF qulflangan emas yoki parol olib tashlandi!")
	b.sendMainMenu(chatID, lang)
}

func (b *Bot) startPDFToJPGFlow(ctx context.Context, chatID, userID int64) {
	lang := b.getUserLang(ctx, userID)
	b.userStates.Store(userID, StatePDFToJPGUpload)

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("↩️ " + T(lang, "back_to_menu")),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "📷 *PDF to JPG*\n\nPDF faylni yuboring, biz uni rasmlarga (JPG) aylantirib beramiz:")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

func (b *Bot) processPDFToJPG(ctx context.Context, chatID, userID int64, fileID, lang string) {
	canUse, coins, isPremium, _ := b.services.BotUser().CanUseService(ctx, userID)
	if !canUse {
		refLink := fmt.Sprintf("https://t.me/%s?start=ref%d", b.botUsername, userID)
		b.sendNoCoinsMessage(chatID, lang, refLink)
		return
	}
	if !isPremium && coins <= 5 && coins > 0 {
		b.sendMessage(chatID, fmt.Sprintf(T(lang, "coins_left"), coins))
	}

	b.sendMessage(chatID, "⏳ PDF rasmlarga aylantirilmoqda (bir oz vaqt olishi mumkin)...")
	b.api.Send(tgbotapi.NewChatAction(chatID, tgbotapi.ChatUploadDocument))

	uid := strconv.FormatInt(userID, 10)
	req := models.PDFToJPGRequest{
		InputFileID: fileID,
		Quality:     75,
	}
	jobID, err := b.services.PDFToJPG().Create(ctx, req, &uid)
	if err != nil {
		b.log.Error("PDF to JPG failed", logger.Error(err))
		b.sendMessage(chatID, "❌ Xatolik: "+err.Error())
		b.sendMainMenu(chatID, lang)
		return
	}

	job, err := b.services.PDFToJPG().GetByID(ctx, jobID)
	if err != nil || job == nil {
		b.sendMessage(chatID, "❌ Natija topilmadi")
		b.sendMainMenu(chatID, lang)
		return
	}

	b.services.BotUser().UseService(ctx, userID)

	if job.ZipFileID != nil {
		b.sendResult(chatID, userID, job.ZipFileID, lang)
		b.sendMessage(chatID, "✅ Barcha rasmlar ZIP arxivda yuborildi.")
	} else {
		b.sendMessage(chatID, "⚠️ Rasmlar topilmadi yoki xatolik.")
	}
}

func (b *Bot) checkSubscription(ctx context.Context, userID int64) (bool, string) {
	if b.requiredChannel == "" {
		return true, "no_channel_required"
	}

	if b.isAdmin(userID) {
		return true, "admin"
	}

	memberConfig := tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			SuperGroupUsername: b.requiredChannel,
			UserID:             userID,
		},
	}

	member, err := b.api.GetChatMember(memberConfig)
	if err != nil {
		b.log.Error("Failed to check subscription",
			logger.String("channel", b.requiredChannel),
			logger.Int64("user_id", userID),
			logger.Error(err),
		)

		return false, fmt.Sprintf("Error: %v", err)
	}

	if member.Status == "member" || member.Status == "creator" || member.Status == "administrator" || member.Status == "restricted" {
		return true, member.Status
	}
	return false, fmt.Sprintf("Status: %s", member.Status)
}

func (b *Bot) sendSubscriptionRequest(chatID int64, lang string) {
	text := T(lang, "sub_required")

	channelURL := "https://t.me/" + strings.TrimPrefix(b.requiredChannel, "@")

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL(T(lang, "sub_button"), channelURL),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(T(lang, "check_sub"), "check_sub"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

func (b *Bot) sendUserList(ctx context.Context, chatID int64) {
	users, err := b.services.BotUser().GetAllUsers(ctx)
	if err != nil {
		b.sendMessage(chatID, "❌ Xatolik yuz berdi: "+err.Error())
		return
	}

	if len(users) == 0 {
		b.sendMessage(chatID, "ℹ️ Foydalanuvchilar topilmadi")
		return
	}

	count := len(users)
	limit := 30
	start := 0
	if count > limit {
		start = count - limit
	}

	// Create text report
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("👥 *Foydalanuvchilar ro'yxati* (Jami: %d)\n\n", count))
	sb.WriteString("_Oxirgi qo'shilganlar:_\n\n")

	// Reverse iterate for newest first
	for i := count - 1; i >= start; i-- {
		u := users[i]
		username := ""
		if u.Username != "" {
			username = "@" + u.Username
		} else {
			username = u.FirstName
		}

		line := fmt.Sprintf("%d. *%s* | 🆔 `%d` | 💰 %d\n",
			count-i,
			username,
			u.TelegramID,
			u.Coins,
		)
		sb.WriteString(line)
	}

	if count > limit {
		sb.WriteString(fmt.Sprintf("\n... va yana %d ta eskiroq (to'liq ro'yxat uchun *Backup* ni bosing)", start))
	}

	msg := tgbotapi.NewMessage(chatID, sb.String())
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

func (b *Bot) sendBackupExcel(ctx context.Context, chatID int64) {
	b.sendMessage(chatID, "⏳ Backup fayl tayyorlanmoqda...")

	users, err := b.services.BotUser().GetAllUsers(ctx)
	if err != nil {
		b.sendMessage(chatID, "❌ Xatolik yuz berdi: "+err.Error())
		return
	}

	if len(users) == 0 {
		b.sendMessage(chatID, "ℹ️ Foydalanuvchilar topilmadi")
		return
	}

	// CSV fayl yaratish
	fileName := fmt.Sprintf("backup_users_%s.csv", time.Now().Format("2006-01-02_15-04-05"))
	csvPath := filepath.Join("/tmp", fileName)

	file, err := os.Create(csvPath)
	if err != nil {
		b.sendMessage(chatID, "❌ Fayl yaratishda xatolik!")
		b.log.Error("Create file error", logger.Error(err))
		return
	}
	defer func() {
		file.Close()
		os.Remove(csvPath)
	}()

	// UTF-8 BOM for Excel compatibility
	file.WriteString("\xEF\xBB\xBF")

	// CSV Header
	file.WriteString("Telegram ID,Username,Ism,Familiya,Til,Coinlar,Ishlatilgan,Referal,Ro'yxatdan o'tgan\n")

	// Data writing
	for _, u := range users {
		refBy := ""
		if u.ReferredBy != nil {
			refBy = fmt.Sprintf("%d", *u.ReferredBy)
		}

		// Familiya va ism ajratish
		firstName := u.FirstName
		lastName := ""
		if parts := strings.SplitN(u.FirstName, " ", 2); len(parts) > 1 {
			firstName = parts[0]
			lastName = parts[1]
		}

		line := fmt.Sprintf("%d,%q,%q,%q,%s,%d,%d,%s,%s\n",
			u.TelegramID,
			u.Username,
			firstName,
			lastName,
			u.Language,
			u.Coins,
			u.TotalUsed,
			refBy,
			u.CreatedAt.Format("2006-01-02 15:04:05"),
		)
		file.WriteString(line)
	}

	// Yuborish
	doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(csvPath))
	doc.Caption = fmt.Sprintf("🗄 <b>Full Backup (Excel/CSV)</b>\n👥 Foydalanuvchilar: %d ta\n📅 Sana: %s", len(users), time.Now().Format("2006-01-02 15:04"))
	doc.ParseMode = "HTML"

	if _, err := b.api.Send(doc); err != nil {
		b.sendMessage(chatID, "❌ Yuborishda xatolik: "+err.Error())
	}
}

func (b *Bot) resetMenuButton() {
	// "menu_button": {"type": "default"} => Reset to standard menu button
	params := tgbotapi.Params{
		"menu_button": `{"type": "default"}`,
	}
	_, err := b.api.MakeRequest("setChatMenuButton", params)
	if err != nil {
		b.log.Error("Failed to reset chat menu button", logger.Error(err))
	} else {
		b.log.Info("Chat menu button reset to default")
	}
}
