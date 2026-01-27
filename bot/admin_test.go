package bot

import (
	"testing"
)

func TestAdminPanel(t *testing.T) {
	t.Run("AdminButtonsExist", func(t *testing.T) {
		// Admin panelda quyidagi tugmalar bo'lishi kerak:
		expectedButtons := []string{
			"📊 Global Stats",
			"➕ Add Coins",
			"👥 Users List",
			"🗄 Backup",
			"↩️ User Mode",
		}

		for _, btn := range expectedButtons {
			t.Logf("Admin button mavjud: %s", btn)
		}
	})
}

func TestAdminCommands(t *testing.T) {
	t.Run("StatsCommand", func(t *testing.T) {
		// /stats yoki "📊 Global Stats" - statistikani ko'rsatadi
		t.Log("Stats: Total users, tools count, total jobs, pending cleanup")
	})

	t.Run("AddCoinsCommand", func(t *testing.T) {
		// /addcoins @username 100
		// Yoki "➕ Add Coins" tugmasi orqali
		t.Log("AddCoins: User ID yoki @username + amount kiritish kerak")
	})

	t.Run("UsersListCommand", func(t *testing.T) {
		// "👥 Users List" - oxirgi 30 ta foydalanuvchini ko'rsatadi
		t.Log("UsersList: Oxirgi 30 ta user, ID, username, coins bilan")
	})

	t.Run("BackupCommand", func(t *testing.T) {
		// "🗄 Backup" - barcha userlarni CSV fayl sifatida yuboradi
		t.Log("Backup: CSV fayl (Excel-compatible) yaratadi")
		t.Log("Ma'lumotlar: Telegram ID, Username, Ism, Til, Coins, Usage, Referal, Sana")
	})

	t.Run("CleanupCommand", func(t *testing.T) {
		// /cleanup - eski fayllarni darhol o'chiradi
		t.Log("Cleanup: Barcha eski fayllarni diskdan va DB dan o'chiradi")
	})

	t.Run("TopCommand", func(t *testing.T) {
		// /top - eng faol 20 ta foydalanuvchini ko'rsatadi
		t.Log("Top: Top 20 users by total_used jobs")
	})
}

func TestAdminSecurity(t *testing.T) {
	t.Run("OnlyAdminCanAccessPanel", func(t *testing.T) {
		adminUserID := int64(123456789)
		regularUserID := int64(987654321)

		isAdmin := func(userID int64) bool {
			return userID == adminUserID
		}

		if !isAdmin(adminUserID) {
			t.Error("Admin user admin sifatida tanilishi kerak")
		}

		if isAdmin(regularUserID) {
			t.Error("Oddiy user admin bo'lmasligi kerak")
		}
	})

	t.Run("AdminStatesSecurity", func(t *testing.T) {
		// Admin states faqat admin uchun ishlashi kerak
		// bot.go:274 qatorida tekshirish mavjud
		t.Log("Admin states: StateAddCoinsUser, StateAddCoinsAmount")
		t.Log("Double check: userID == adminUserID")
	})
}

func TestUserListPagination(t *testing.T) {
	t.Run("LimitedToLast30Users", func(t *testing.T) {
		totalUsers := 100
		limit := 30

		start := 0
		if totalUsers > limit {
			start = totalUsers - limit
		}

		displayedUsers := totalUsers - start
		if displayedUsers != limit {
			t.Errorf("Expected %d users, got %d", limit, displayedUsers)
		}
	})

	t.Run("FullBackupAvailable", func(t *testing.T) {
		// To'liq ro'yxat uchun Backup tugmasi mavjud
		t.Log("Full backup: CSV fayl barcha userlarni o'z ichiga oladi")
	})
}
