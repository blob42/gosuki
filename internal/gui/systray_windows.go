//go:build windows && systray
// +build windows,systray

package gui

import (
	"os"
	"os/signal"
	"syscall"
	
	"github.com/energye/systray"
)

// WindowsRunSystray démarre la systray pour Windows
func WindowsRunSystray(manager interface{}) {
	// Gérer les signaux système
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-c
		systray.Quit()
	}()
	
	// Démarrer la systray
	systray.Run(onReady, onExit)
}

func onReady() {
	// Titre et tooltip de base
	systray.SetTitle("Gosuki")
	systray.SetTooltip("Gosuki - Browser Bookmark Manager")
	
	// Menu basique
	mQuit := systray.AddMenuItem("Quit", "Quit Gosuki")
	
	// Gestion du clic Quit
	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
	}()
}

func onExit() {
	// Nettoyage si nécessaire
}