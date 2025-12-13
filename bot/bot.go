package bot

import (
	"context"
	"fmt"
	"html"
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
	StateNone            = ""
	StateAddCoinsUser    = "admin_add_coins_user"
	StateAddCoinsAmount  = "admin_add_coins_amount"
	StateMergeUpload     = "merge_upload_files"
	StateSplitUpload     = "split_upload_file"   // Split: fayl kutish
	StateSplitPages      = "split_enter_pages"   // Split: sahifalar kutish
	StateRotateUpload    = "rotate_upload_file"  // Rotate: fayl kutish
	StateWatermarkUpload = "watermark_upload"    // Watermark: fayl kutish
	StateWatermarkText   = "watermark_enter_text" // Watermark: matn kutish
	StateUnlockUpload    = "unlock_upload_file"   // Unlock: fayl kutish
	StatePDFToJPGUpload  = "pdf_to_jpg_upload"    // PDF to JPG: fayl kutish
)

type Bot struct {
	api              *tgbotapi.BotAPI
	services         service.IServiceManager
	log              logger.ILogger
	adminUserID      int64
	botUsername      string
	requiredChannel  string   // Majburiy kanal
	processingUsers  sync.Map
	userStates       sync.Map
	adminContext     sync.Map
	mergeContext     sync.Map
	splitContext     sync.Map // Split: fayl ID saqlash
	mediaGroupWarned sync.Map // Bir vaqtda ko'p fayl yuborishda ogohlantirish uchun
	workerPool       chan struct{} // Concurrent ishlarni cheklash
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
		workerPool:      make(chan struct{}, 10), // 10 ta concurrent ish
	}, nil
}

// GetAPI - Bot API ni qaytarish (CronService uchun)
func (b *Bot) GetAPI() *tgbotapi.BotAPI {
	return b.api
}

func (b *Bot) Start(ctx context.Context) {
	b.log.Info("Bot started", logger.String("username", b.api.Self.UserName))
	b.log.Info("Configured Admin ID", logger.Int64("admin_id", b.adminUserID))

	// Bot commands ro'yxatini sozlash
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
	
	// Reset Menu Button (remove Web App button if exists)
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
	// PreCheckoutQuery - to'lov oldidan tasdiqlash
	if update.PreCheckoutQuery != nil {
		b.handlePreCheckout(update.PreCheckoutQuery)
		return
	}

	// Callback query (inline button)
	if update.CallbackQuery != nil {
		b.handleCallback(ctx, update.CallbackQuery)
		return
	}

	if update.Message == nil {
		return
	}

	// SuccessfulPayment - to'lov muvaffaqiyatli bo'lganda
	if update.Message.SuccessfulPayment != nil {
		b.handleSuccessfulPayment(ctx, update.Message)
		return
	}

	// Media group (bir vaqtda ko'p fayl) tekshirish
	if update.Message.MediaGroupID != "" {
		lang := b.getUserLang(ctx, update.Message.From.ID)
		// Faqat birinchi faylga javob berish (takroriy xabarlarni oldini olish)
		if _, exists := b.mediaGroupWarned.LoadOrStore(update.Message.MediaGroupID, true); !exists {
			b.sendMessage(update.Message.Chat.ID, T(lang, "only_one_file"))
			// 30 soniyadan keyin xotirada tozalash
			go func(groupID string) {
				time.Sleep(30 * time.Second)
				b.mediaGroupWarned.Delete(groupID)
			}(update.Message.MediaGroupID)
		}
		return
	}

	// Foydalanuvchini ro'yxatdan o'tkazish
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

	// Kanalga obuna bo'lishni tekshirish
	isSubscribed, _ := b.checkSubscription(ctx, update.Message.From.ID)
	if !isSubscribed {
		lang := b.getUserLang(ctx, update.Message.From.ID)
		b.sendSubscriptionRequest(update.Message.Chat.ID, lang)
		return
	}

	// Yangi foydalanuvchi - til tanlash
	user, _ := b.services.BotUser().GetUser(ctx, update.Message.From.ID)
	if user != nil && user.Language == "" && update.Message.Command() == "start" {
		b.sendLanguageSelection(update.Message.Chat.ID)
		return
	}

	// Referal orqali kelgan
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

	// Photo (rasm) qabul qilish
	if update.Message.Photo != nil && len(update.Message.Photo) > 0 {
		b.handlePhoto(ctx, update.Message)
		return
	}

	// Text message handling (Menu clicks)
	if update.Message.Text != "" {
		lang := b.getUserLang(ctx, update.Message.From.ID)
		// 1. Check Admin States
		if state, ok := b.userStates.Load(update.Message.From.ID); ok && state != StateNone {
			b.handleAdminStates(ctx, &update, state.(string))
			return
		}

		// 2. Check Admin Commands
		if update.Message.From.ID == b.adminUserID {
			switch update.Message.Text {
			case "👨‍⚖️ Admin Panel":
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
				lang := b.getUserLang(ctx, update.Message.From.ID)
				b.sendUserMenu(update.Message.Chat.ID, lang)
				return
			}
		}

		// 4. Merge Handling (Text click)
		if key := GetKey(lang, update.Message.Text); key == "merge_pdf" {
			b.startMergeFlow(ctx, update.Message.Chat.ID, update.Message.From.ID)
			return
		}

		// 4.1 Split Handling (Text click)
		if key := GetKey(lang, update.Message.Text); key == "split_pdf" {
			b.startSplitFlow(ctx, update.Message.Chat.ID, update.Message.From.ID)
			return
		}

		// 4.2 Rotate PDF Handling
		if key := GetKey(lang, update.Message.Text); key == "rotate_pdf" {
			b.startRotateFlow(ctx, update.Message.Chat.ID, update.Message.From.ID)
			return
		}

		// 4.3 Watermark Handling
		if key := GetKey(lang, update.Message.Text); key == "watermark" {
			b.startWatermarkFlow(ctx, update.Message.Chat.ID, update.Message.From.ID)
			return
		}

		// 4.4 Unlock PDF Handling
		if key := GetKey(lang, update.Message.Text); key == "unlock_pdf" {
			b.startUnlockFlow(ctx, update.Message.Chat.ID, update.Message.From.ID)
			return
		}

		// 4.5 PDF to JPG Handling
		if key := GetKey(lang, update.Message.Text); key == "pdf_to_jpg" {
			b.startPDFToJPGFlow(ctx, update.Message.Chat.ID, update.Message.From.ID)
			return
		}

		// 5. Done command for merge
		if update.Message.Text == "Done" || update.Message.Text == "/done" {
			if state, ok := b.userStates.Load(update.Message.From.ID); ok && state == StateMergeUpload {
				b.finishMergeFlow(ctx, update.Message.Chat.ID, update.Message.From.ID)
				return
			}
		}

		// 5.1 Split pages state - sahifalar kiritilganda
		if state, ok := b.userStates.Load(update.Message.From.ID); ok && state == StateSplitPages {
			b.finishSplitFlow(ctx, update.Message.Chat.ID, update.Message.From.ID, update.Message.Text)
			return
		}

		// 6. Normal Menu Handling
		if key := GetKey(lang, update.Message.Text); key != "" {
			switch key {
			case "compress_pdf":
				b.services.BotUser().SetCurrentAction(ctx, update.Message.From.ID, "compress")
				b.sendMessage(update.Message.Chat.ID, T(lang, "select_file"))
			case "word_to_pdf":
				b.services.BotUser().SetCurrentAction(ctx, update.Message.From.ID, "word")
				b.sendMessage(update.Message.Chat.ID, T(lang, "select_file"))
			case "excel_to_pdf":
				b.services.BotUser().SetCurrentAction(ctx, update.Message.From.ID, "excel")
				b.sendMessage(update.Message.Chat.ID, T(lang, "select_file"))
			case "ppt_to_pdf":
				b.services.BotUser().SetCurrentAction(ctx, update.Message.From.ID, "ppt")
				b.sendMessage(update.Message.Chat.ID, T(lang, "select_file"))
			case "image_to_pdf":
				b.services.BotUser().SetCurrentAction(ctx, update.Message.From.ID, "image")
				b.sendMessage(update.Message.Chat.ID, T(lang, "select_file"))
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
			}
			return
		}

		// Handle "Menu" button specifically if it's not in i18n or just text match
		if update.Message.Text == "🏠 Menu" || update.Message.Text == "🏠 Asosiy menu" {
			b.sendMainMenu(update.Message.Chat.ID, lang)
			return
		}
	}

	lang := b.getUserLang(ctx, update.Message.From.ID)
	b.sendMainMenu(update.Message.Chat.ID, lang)
}

// ==================== CALLBACK (INLINE BUTTONS) ====================

func (b *Bot) handleCallback(ctx context.Context, callback *tgbotapi.CallbackQuery) {
	data := callback.Data
	chatID := callback.Message.Chat.ID
	userID := callback.From.ID

	// Obunani tekshirish tugmasi
	if data == "check_sub" {
		lang := b.getUserLang(ctx, userID)
		isSubscribed, status := b.checkSubscription(ctx, userID)
		if isSubscribed {
			b.api.Request(tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID))
			b.sendMessage(chatID, T(lang, "sub_success"))
			b.sendMainMenu(chatID, lang)
		} else {
			// Reasonni ko'rsatish
			msg := fmt.Sprintf("%s (%s)", T(lang, "sub_not_found"), status)
			b.api.Request(tgbotapi.NewCallback(callback.ID, msg))
		}
		return
	}

	// Til tanlashdan boshqa holatlarda obuna kerak
	if !strings.HasPrefix(data, "lang_") {
		isSubscribed, _ := b.checkSubscription(ctx, userID)
		if !isSubscribed {
			lang := b.getUserLang(ctx, userID)
			b.api.Request(tgbotapi.NewCallback(callback.ID, T(lang, "sub_required")))
			b.sendSubscriptionRequest(chatID, lang)
			return
		}
	}

	// Callback javobini yuborish (muvaffaqiyatli bo'lsa)
	b.api.Request(tgbotapi.NewCallback(callback.ID, ""))

	// Til tanlash
	if strings.HasPrefix(data, "lang_") {
		lang := strings.TrimPrefix(data, "lang_")
		b.services.BotUser().SetLanguage(ctx, userID, lang)
		
		// Xabarni o'chirish va yangi menu yuborish
		b.api.Request(tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID))
		b.sendMessage(chatID, T(lang, "lang_selected"))
		b.sendMainMenu(chatID, lang)
		return
	}

	// Rotate callback: rotate:fileID:angle
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

	// Menu actions
	switch data {
	case "menu_compress":
		b.services.BotUser().SetCurrentAction(ctx, userID, "compress")
		b.sendMessage(chatID, T(lang, "select_file"))
	case "menu_word":
		b.services.BotUser().SetCurrentAction(ctx, userID, "word")
		b.sendMessage(chatID, T(lang, "select_file"))
	case "menu_excel":
		b.services.BotUser().SetCurrentAction(ctx, userID, "excel")
		b.sendMessage(chatID, T(lang, "select_file"))
	case "menu_ppt":
		b.services.BotUser().SetCurrentAction(ctx, userID, "ppt")
		b.sendMessage(chatID, T(lang, "select_file"))
	case "menu_image":
		b.services.BotUser().SetCurrentAction(ctx, userID, "image")
		b.sendMessage(chatID, T(lang, "select_file"))
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
	}
}

// ==================== LANGUAGE SELECTION ====================

func (b *Bot) sendLanguageSelection(chatID int64) {
	// 20 ta til uchun keyboard
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

// ==================== MAIN MENU ====================

func (b *Bot) sendMainMenu(chatID int64, lang string) {
	// If Admin, show Admin Panel
	if chatID == b.adminUserID {
		b.sendAdminPanel(chatID)
		return
	}
	// Otherwise show User Menu
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

// ==================== COMMANDS ====================

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

	// PDF amallar
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
		b.services.BotUser().SetCurrentAction(ctx, message.From.ID, "image")
		b.sendMessage(message.Chat.ID, T(lang, "select_file"))

	// Admin commands - faqat admin uchun
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

// ==================== COINS & REFERRAL ====================

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

// ==================== FILE HANDLING ====================

func (b *Bot) handleDocument(ctx context.Context, message *tgbotapi.Message) {
	lang := b.getUserLang(ctx, message.From.ID)
	userID := message.From.ID

	if _, loaded := b.processingUsers.LoadOrStore(userID, true); loaded {
		b.sendMessage(message.Chat.ID, T(lang, "processing_wait"))
		return
	}
	defer b.processingUsers.Delete(userID)

	// File Size Check (e.g. 50MB)
	if message.Document != nil && message.Document.FileSize > 50*1024*1024 {
		b.sendMessage(message.Chat.ID, "❌ Fayl hajmi juda katta! Maksimal 50MB.")
		return
	}

	// Coin tekshirish
	canUse, coins, isPremium, _ := b.services.BotUser().CanUseService(ctx, userID)
	if !canUse {
		refLink := fmt.Sprintf("https://t.me/%s?start=ref%d", b.botUsername, userID)
		b.sendNoCoinsMessage(message.Chat.ID, lang, refLink)
		return
	}

	// Premium bo'lmasa va coin kam bo'lsa ogohlantirish
	if !isPremium && coins <= 5 && coins > 0 {
		b.sendMessage(message.Chat.ID, fmt.Sprintf(T(lang, "coins_left"), coins))
	}

	// Foydalanuvchi actionini olish
	user, _ := b.services.BotUser().GetUser(ctx, userID)
	action := ""
	if user != nil {
		action = user.CurrentAction
	}

	// 1. Check Merge State
	if state, ok := b.userStates.Load(userID); ok && state == StateMergeUpload {
		fileID, err := b.saveUploadedFile(ctx, message)
		if err != nil {
			b.sendMessage(message.Chat.ID, "❌ Faylni saqlashda xatolik!")
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

	// 2. Check Split State - fayl kutilayotgan bo'lsa
	if state, ok := b.userStates.Load(userID); ok && state == StateSplitUpload {
		doc := message.Document
		ext := strings.ToLower(filepath.Ext(doc.FileName))
		if ext != ".pdf" {
			b.sendMessage(message.Chat.ID, "❌ Faqat PDF fayl yuboring!")
			return
		}

		fileID, err := b.saveUploadedFile(ctx, message)
		if err != nil {
			b.sendMessage(message.Chat.ID, "❌ Faylni saqlashda xatolik!")
			return
		}

		b.splitContext.Store(userID, fileID)
		b.userStates.Store(userID, StateSplitPages)
		b.sendMessage(message.Chat.ID, T(lang, "split_enter_pages"))
		return
	}

	// 3. Check Rotate State - fayl kutilayotgan bo'lsa
	if state, ok := b.userStates.Load(userID); ok && state == StateRotateUpload {
		doc := message.Document
		ext := strings.ToLower(filepath.Ext(doc.FileName))
		if ext != ".pdf" {
			b.sendMessage(message.Chat.ID, "❌ Faqat PDF fayl yuboring!")
			return
		}

		fileID, err := b.saveUploadedFile(ctx, message)
		if err != nil {
			b.sendMessage(message.Chat.ID, "❌ Faylni saqlashda xatolik!")
			return
		}

		// Burchakni tanlash uchun inline keyboard
		b.sendRotateAngleSelection(message.Chat.ID, fileID, lang)
		b.userStates.Delete(userID)
		return
	}

	// 4. Check Watermark State - fayl kutilayotgan bo'lsa
	if state, ok := b.userStates.Load(userID); ok && state == StateWatermarkUpload {
		doc := message.Document
		ext := strings.ToLower(filepath.Ext(doc.FileName))
		if ext != ".pdf" {
			b.sendMessage(message.Chat.ID, "❌ Faqat PDF fayl yuboring!")
			return
		}

		fileID, err := b.saveUploadedFile(ctx, message)
		if err != nil {
			b.sendMessage(message.Chat.ID, "❌ Faylni saqlashda xatolik!")
			return
		}

		// Matn so'rash
		b.splitContext.Store(userID, fileID) // fileID ni saqlash
		b.userStates.Store(userID, StateWatermarkText)
		b.sendMessage(message.Chat.ID, "💧 Suv belgisi uchun matn kiriting (masalan: CONFIDENTIAL):")
		return
	}

	// 5. Check Unlock State
	if state, ok := b.userStates.Load(userID); ok && state == StateUnlockUpload {
		doc := message.Document
		ext := strings.ToLower(filepath.Ext(doc.FileName))
		if ext != ".pdf" {
			b.sendMessage(message.Chat.ID, "❌ Faqat PDF fayl yuboring!")
			return
		}

		fileID, err := b.saveUploadedFile(ctx, message)
		if err != nil {
			b.sendMessage(message.Chat.ID, "❌ Faylni saqlashda xatolik!")
			return
		}

		b.userStates.Delete(userID)
		go b.processUnlock(ctx, message.Chat.ID, userID, fileID, lang)
		return
	}

	// 6. Check PDF to JPG State
	if state, ok := b.userStates.Load(userID); ok && state == StatePDFToJPGUpload {
		doc := message.Document
		ext := strings.ToLower(filepath.Ext(doc.FileName))
		if ext != ".pdf" {
			b.sendMessage(message.Chat.ID, "❌ Faqat PDF fayl yuboring!")
			return
		}

		fileID, err := b.saveUploadedFile(ctx, message)
		if err != nil {
			b.sendMessage(message.Chat.ID, "❌ Faylni saqlashda xatolik!")
			return
		}

		b.userStates.Delete(userID)
		go b.processPDFToJPG(ctx, message.Chat.ID, userID, fileID, lang)
		return
	}

	// Agar action tanlanmagan bo'lsa - avtomatik aniqlash (Default flow)
	doc := message.Document
	ext := strings.ToLower(filepath.Ext(doc.FileName))

	if action == "" {
		switch ext {
		case ".pdf":
			action = "compress"
		case ".doc", ".docx":
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

	// Process file
	b.processFile(ctx, message, action, lang)

	// Reset action
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
	
	// Show "uploading file..." status
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

	// Check payment
	isAdmin := userID == b.adminUserID
	if isAdmin {
		// Admin: Bypass payment but INCREMENT usage stats
		b.services.BotUser().IncrementUsage(context.Background(), userID)
	} else {
		// User: Must pay
		err := b.services.BotUser().UseService(context.Background(), userID)
		if err != nil {
			// Failed (likely no coins)
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

	// Add Back to Menu button
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

// ==================== ADMIN ====================

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

// ==================== HELPERS ====================

func (b *Bot) ensureUser(ctx context.Context, user *tgbotapi.User, refID *int64) {
	username := user.UserName
	b.services.BotUser().EnsureUser(ctx, user.ID, username, user.FirstName, refID)
}

// handlePhoto - Photo (rasm) ni PDF ga aylantirish
func (b *Bot) handlePhoto(ctx context.Context, message *tgbotapi.Message) {
	lang := b.getUserLang(ctx, message.From.ID)
	userID := message.From.ID

	if _, loaded := b.processingUsers.LoadOrStore(userID, true); loaded {
		b.sendMessage(message.Chat.ID, T(lang, "processing_wait"))
		return
	}
	defer b.processingUsers.Delete(userID)

	// Coin tekshirish
	canUse, coins, isPremium, _ := b.services.BotUser().CanUseService(ctx, userID)
	if !canUse {
		refLink := fmt.Sprintf("https://t.me/%s?start=ref%d", b.botUsername, userID)
		b.sendNoCoinsMessage(message.Chat.ID, lang, refLink)
		return
	}
	if !isPremium && coins <= 5 && coins > 0 {
		b.sendMessage(message.Chat.ID, fmt.Sprintf(T(lang, "coins_left"), coins))
	}

	b.sendMessage(message.Chat.ID, T(lang, "file_received"))

	// Eng katta rasmni tanlash
	photos := message.Photo
	photo := photos[len(photos)-1]

	// Rasmni yuklab olish va saqlash
	fileID, err := b.savePhoto(ctx, photo.FileID, message.From.ID)
	if err != nil {
		b.log.Error("savePhoto failed", logger.Error(err))
		b.sendMessage(message.Chat.ID, T(lang, "error")+" (photo)")
		return
	}

	b.log.Info("Photo saved", logger.String("fileID", fileID))

	// PDF ga aylantirish
	jobID, err := b.services.JPGToPDF().CreateJob(ctx, nil, []string{fileID})
	if err != nil {
		b.log.Error("JPGToPDF failed", logger.Error(err))
		b.sendMessage(message.Chat.ID, T(lang, "error")+" (convert)")
		return
	}

	job, _ := b.services.JPGToPDF().GetJobByID(ctx, jobID)
	if job == nil || job.OutputFileID == nil {
		b.sendMessage(message.Chat.ID, T(lang, "error"))
		return
	}

	b.sendResult(message.Chat.ID, userID, job.OutputFileID, lang)
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

// isAdmin already declared above

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

// sendPartnershipInfo - Hamkorlik uchun ma'lumot yuborish
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
	b.mergeContext.Delete(userID) // Clear previous files

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Done"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("↩️ User Mode"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "Birlashtirish uchun PDF fayllarni birin-ketin yuboring.\nBarchasini yuborib bo'lgach *Done* tugmasini bosing.")
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

	// Create Job
	job := models.MergeJob{
		UserID:       userID,
		InputFileIDs: fileIDs,
		Status:       "pending",
	}

	jobID, err := b.services.Merge().Create(ctx, job)
	if err != nil {
		b.log.Error("Failed to create merge job", logger.Error(err))
		b.sendMessage(chatID, "Xatolik yuz berdi.")
		return
	}

	// ⬇️ CONCURRENCY LIMIT START
	select {
	case b.workerPool <- struct{}{}:
		// Slot band qilindi
		defer func() { <-b.workerPool }() // Ish tugagach slotni bo'shatish
	default:
		// Slot yo'q = server band
		b.sendMessage(chatID, "⚠️ Server ayni damda band. Iltimos 1 daqiqadan so'ng urinib ko'ring.")
		return
	}
	// ⬆️ CONCURRENCY LIMIT END

	resultFile, err := b.services.Merge().MergeFiles(ctx, jobID)
	if err != nil {
		b.log.Error("Failed to merge files", logger.Error(err))
		b.sendMessage(chatID, "Birlashtirishda xatolik: "+err.Error())
		return
	}

	// Send Result
	lang := b.getUserLang(ctx, userID)
	b.sendResult(chatID, userID, &resultFile.ID, lang)
	
	// Clean up
	b.userStates.Delete(userID)
	b.mergeContext.Delete(userID)
	b.sendMainMenu(chatID, lang)
}

// ==================== SPLIT PDF ====================

func (b *Bot) startSplitFlow(ctx context.Context, chatID, userID int64) {
	lang := b.getUserLang(ctx, userID)

	b.userStates.Store(userID, StateSplitUpload)
	b.splitContext.Delete(userID) // Clear previous

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

	// Bekor qilish
	if pagesInput == "↩️ "+T(lang, "back_to_menu") || pagesInput == "/cancel" {
		b.userStates.Delete(userID)
		b.splitContext.Delete(userID)
		b.sendMainMenu(chatID, lang)
		return
	}

	// Fayl ID ni olish
	fileIDInterface, ok := b.splitContext.Load(userID)
	if !ok {
		b.sendMessage(chatID, "❌ Avval PDF fayl yuboring!")
		b.userStates.Store(userID, StateSplitUpload)
		return
	}
	fileID := fileIDInterface.(string)

	// Split service ni chaqirish
	req := models.CreateSplitJobRequest{
		InputFileID: fileID,
		SplitRanges: pagesInput,
	}

	// ⬇️ CONCURRENCY LIMIT START
	select {
	case b.workerPool <- struct{}{}:
		defer func() { <-b.workerPool }()
	default:
		b.sendMessage(chatID, "⚠️ Server band. Qayta urinib ko'ring.")
		return
	}
	// ⬆️ CONCURRENCY LIMIT END

	jobID, err := b.services.Split().Create(ctx, req, nil)

	if err != nil {
		b.log.Error("Split failed", logger.Error(err))
		b.sendMessage(chatID, "❌ Bo'lishda xatolik: "+err.Error())
		b.userStates.Delete(userID)
		b.splitContext.Delete(userID)
		b.sendMainMenu(chatID, lang)
		return
	}

	// Natijalarni olish
	job, err := b.services.Split().GetByID(ctx, jobID)
	if err != nil || job == nil || len(job.OutputFileIDs) == 0 {
		b.sendMessage(chatID, "❌ Natija topilmadi")
		b.userStates.Delete(userID)
		b.splitContext.Delete(userID)
		b.sendMainMenu(chatID, lang)
		return
	}

	// Coin ayirish
	b.services.BotUser().UseService(ctx, userID)

	// Har bir fayl uchun yuborish
	for _, outFileID := range job.OutputFileIDs {
		b.sendResult(chatID, userID, &outFileID, lang)
	}

	b.sendMessage(chatID, fmt.Sprintf(T(lang, "split_success"), len(job.OutputFileIDs)))

	// Clean up
	b.userStates.Delete(userID)
	b.splitContext.Delete(userID)
	b.sendMainMenu(chatID, lang)
}

// ==================== BUY COINS / STARS PAYMENT ====================

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
		Currency:    "XTR", // Telegram Stars
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

// ==================== PAYMENT HANDLERS ====================

// handlePreCheckout - to'lovni tasdiqlash
func (b *Bot) handlePreCheckout(query *tgbotapi.PreCheckoutQuery) {
	// Barcha to'lovlarni tasdiqlash
	callback := tgbotapi.PreCheckoutConfig{
		PreCheckoutQueryID: query.ID,
		OK:                 true,
	}
	b.api.Request(callback)
	b.log.Info("PreCheckout approved", 
		logger.Int64("user_id", query.From.ID),
		logger.String("payload", query.InvoicePayload))
}

// handleSuccessfulPayment - muvaffaqiyatli to'lovni qayta ishlash
func (b *Bot) handleSuccessfulPayment(ctx context.Context, message *tgbotapi.Message) {
	payment := message.SuccessfulPayment
	userID := message.From.ID
	lang := b.getUserLang(ctx, userID)

	// Payload ni parse qilish: "coins:userID:amount"
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

	// Coinlarni qo'shish
	err = b.services.BotUser().AddCoins(ctx, userID, coins)
	if err != nil {
		b.log.Error("Failed to add coins", logger.Error(err))
		b.sendMessage(message.Chat.ID, "❌ Xatolik yuz berdi. Iltimos admin bilan bog'laning.")
		return
	}

	// Yangi balansni olish
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

	// Muvaffaqiyatli xabar
	text := fmt.Sprintf("✅ *To'lov muvaffaqiyatli!*\n\n"+
		"💰 Qo'shildi: *%d* coin\n"+
		"💳 Yangi balans: *%d* coin\n\n"+
		"Rahmat! Endi PDF amallarini bajarishingiz mumkin.", coins, newBalance)

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	b.api.Send(msg)

	b.sendMainMenu(message.Chat.ID, lang)
}

// ==================== ROTATE PDF ====================

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

	b.sendMessage(chatID, "⏳ Aylantirilmoqda...")

	// Rotate service ni chaqirish
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

	// Natijani olish
	job, err := b.services.Rotate().GetByID(ctx, jobID)
	if err != nil || job == nil || job.OutputFileID == nil {
		b.sendMessage(chatID, "❌ Natija topilmadi")
		b.sendMainMenu(chatID, lang)
		return
	}

	// Coin ayirish
	b.services.BotUser().UseService(ctx, userID)

	// Natijani yuborish
	b.sendResult(chatID, userID, job.OutputFileID, lang)
	b.sendMessage(chatID, fmt.Sprintf("✅ PDF %d° ga aylantirildi!", angle))
	b.sendMainMenu(chatID, lang)
}

// ==================== WATERMARK PDF ====================

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

	b.sendMessage(chatID, "⏳ Suv belgisi qo'shilmoqda...")

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

// ==================== UNLOCK PDF ====================

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

	b.sendMessage(chatID, "⏳ Parol olib tashlanmoqda...")

	req := models.UnlockRequest{
		InputFileID: fileID,
		Password:    "", // Bo'sh parol
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

	b.sendMessage(chatID, "⏳ PDF rasmlarga aylantirilmoqda (bir oz vaqt olishi mumkin)...")

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
		// Agar ZIP bo'lmasa (masalan 0 rasm), xabar beramiz
		b.sendMessage(chatID, "⚠️ Rasmlar topilmadi yoki xatolik.")
	}
}

// ==================== SUBSCRIPTION ====================

func (b *Bot) checkSubscription(ctx context.Context, userID int64) (bool, string) {
	if b.requiredChannel == "" {
		return true, "no_channel_required"
	}

	// Adminlar uchun tekin
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
		// Xatolik bo'lsa (bot admin emas yoki kanal noto'g'ri), log yozamiz
		b.log.Error("Failed to check subscription", 
			logger.String("channel", b.requiredChannel),
			logger.Int64("user_id", userID),
			logger.Error(err),
		)
		// Xavfsizlik uchun bloklaymiz, admin to'g'irlashi kerak
		// Agar bot kanaldan chiqib ketgan bo'lsa yoki admin bo'lmasa, xato beradi
		return false, fmt.Sprintf("Error: %v", err)
	}



	if member.Status == "member" || member.Status == "creator" || member.Status == "administrator" || member.Status == "restricted" {
		return true, member.Status
	}
	return false, fmt.Sprintf("Status: %s", member.Status)
}

func (b *Bot) sendSubscriptionRequest(chatID int64, lang string) {
	text := T(lang, "sub_required")

	// Kanal havolasi
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
	// Barcha foydalanuvchilarni olish
	users, err := b.services.BotUser().GetAllUsers(ctx)
	if err != nil {
		b.sendMessage(chatID, "❌ Xatolik yuz berdi: "+err.Error())
		return
	}

	if len(users) == 0 {
		b.sendMessage(chatID, "ℹ️ Foydalanuvchilar topilmadi")
		return
	}

	// Oxirgi 30 ta user (yoki kamroq)
	count := len(users)
	limit := 30
	start := 0
	if count > limit {
		start = count - limit
	}
	
	// Create text report
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("👥 <b>Foydalanuvchilar ro'yxati</b> (Jami: %d)\n\n", count))
	sb.WriteString("<i>Oxirgi qo'shilganlar:</i>\n\n")

	// Reverse iterate for newest first if array is ascending by date
	for i := count - 1; i >= start; i-- {
		u := users[i]
		username := ""
		if u.Username != "" {
			username = "@" + u.Username
		} else {
			username = u.FirstName // No username
		}
		
		line := fmt.Sprintf("%d. <b>%s</b> | 🆔 %d | 💰 %d\n", 
			i+1, 
			html.EscapeString(username), 
			u.TelegramID, 
			u.Coins,
		)
		sb.WriteString(line)
	}

	if count > limit {
		sb.WriteString(fmt.Sprintf("\n... va yana %d ta eskiroq (to'liq ro'yxat uchun 'Backup' ni bosing)", start))
	}

	msg := tgbotapi.NewMessage(chatID, sb.String())
	msg.ParseMode = "HTML"
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

	// CSV Header
	file.WriteString("TelegramID,Username,FirstName,Coins,TotalUsed,Language,ReferredBy,CreatedAt\n")

	// Data writing
	for _, u := range users {
		refBy := ""
		if u.ReferredBy != nil {
			refBy = fmt.Sprintf("%d", *u.ReferredBy)
		}

		// %q puts quotes around strings, which is good for CSV (handles commas inside names)
		line := fmt.Sprintf("%d,%q,%q,%d,%d,%s,%s,%s\n",
			u.TelegramID,
			u.Username,
			u.FirstName,
			u.Coins,
			u.TotalUsed,
			u.Language,
			refBy,
			u.CreatedAt.Format("2006-01-02 15:04:05"),
		)
		file.WriteString(line)
	}

	// Yuborish
	doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(csvPath))
	doc.Caption = fmt.Sprintf("🗄 <b>Full Backup (CSV)</b>\n👥 Users: %d\n📅 Date: %s", len(users), time.Now().Format("2006-01-02 15:04"))
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
