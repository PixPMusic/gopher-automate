package tray

import (
	_ "embed"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/PixPMusic/gopher-automate/internal/config"
	"github.com/PixPMusic/gopher-automate/internal/startup"
)

//go:embed icon-white.png
var iconWhiteData []byte

// Callbacks for tray menu actions
type Callbacks struct {
	OnOpen func()
	OnQuit func()
}

// Setup initializes the system tray using Fyne's built-in support
func Setup(app fyne.App, cfg *config.Config, callbacks Callbacks) {
	// Check if we're running as a desktop app
	if desk, ok := app.(desktop.App); ok {
		// Create menu items
		openItem := fyne.NewMenuItem("Open GopherAutomate", func() {
			if callbacks.OnOpen != nil {
				callbacks.OnOpen()
			}
		})

		startupItem := fyne.NewMenuItem("Open at Startup", nil)
		if cfg.OpenAtStartup {
			startupItem.Checked = true
		}

		quitItem := fyne.NewMenuItem("Quit", func() {
			if callbacks.OnQuit != nil {
				callbacks.OnQuit()
			}
		})

		menu := fyne.NewMenu("GopherAutomate",
			openItem,
			fyne.NewMenuItemSeparator(),
			startupItem,
			fyne.NewMenuItemSeparator(),
			quitItem,
		)

		// Set the action after menu is created so we can refresh it
		startupItem.Action = func() {
			if startupItem.Checked {
				startupItem.Checked = false
				cfg.OpenAtStartup = false
				_ = startup.Disable()
			} else {
				startupItem.Checked = true
				cfg.OpenAtStartup = true
				_ = startup.Enable()
			}
			_ = cfg.Save()
			menu.Refresh()
		}

		// Set the system tray menu
		desk.SetSystemTrayMenu(menu)

		// Create a resource from the embedded icon
		// User requested to use the white icon for both modes (macOS style)
		iconResource := fyne.NewStaticResource("icon.png", iconWhiteData)
		desk.SetSystemTrayIcon(iconResource)
	}
}
