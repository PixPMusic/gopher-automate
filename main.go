package main

import (
	"log"

	"fyne.io/fyne/v2/app"
	"github.com/PixPMusic/gopher-automate/internal/config"
	"github.com/PixPMusic/gopher-automate/internal/midi"
	"github.com/PixPMusic/gopher-automate/internal/tray"
	"github.com/PixPMusic/gopher-automate/internal/window"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize MIDI manager
	midiManager := midi.NewManager()
	defer midiManager.Close()

	// Create Fyne app
	fyneApp := app.NewWithID("com.pixpmusic.gopherautomate")

	// Create main window with device configuration UI
	mainWindow := window.NewMainWindow(fyneApp, cfg, midiManager, func() {
		// Called when config is saved
	})

	// Setup system tray
	tray.Setup(fyneApp, cfg, tray.Callbacks{
		OnOpen: func() {
			mainWindow.Show()
		},
		OnQuit: func() {
			fyneApp.Quit()
		},
	})

	// Initialize devices on startup (activate programmer mode and send current layout)
	mainWindow.InitializeDevices()

	// Show window if first launch, otherwise run in background
	if !cfg.FirstLaunchCompleted {
		cfg.FirstLaunchCompleted = true
		if err := cfg.Save(); err != nil {
			log.Printf("Failed to save config: %v", err)
		}
		mainWindow.Show()
	}

	// Run the Fyne app (this blocks until app.Quit is called)
	fyneApp.Run()
}
