package browsers

import (
	"log" // Ajoutez cet import manquant
	"os"
	"path/filepath"
	"runtime"
)

type BrowserFamily uint

const (
	Mozilla BrowserFamily = iota
	ChromeBased
	Qutebrowser
)

type BrowserDef struct {
	Flavour string        // also acts as canonical name
	Family  BrowserFamily // browser family
	// Base browser directory path
	baseDir string
	// (linux only) path to snap package base dir
	snapDir string
	// (linux only) path to flatpak package base dir
	flatDir string
}

// Ajoutez ces méthodes manquantes pour BrowserDef

// Detect vérifie si le navigateur est installé/détectable
func (b *BrowserDef) Detect() bool {
	// Vérifier si le répertoire de base existe
	baseDir, err := b.ExpandBaseDir()
	if err != nil || baseDir == "" {
		return false
	}
	
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return false
	}
	
	return true
}

// ExpandBaseDir retourne le chemin de base étendu (résolution des variables d'environnement)
func (b *BrowserDef) ExpandBaseDir() (string, error) {
	if b.baseDir == "" {
		return "", nil
	}
	
	// Expansion des variables d'environnement
	expanded := os.ExpandEnv(b.baseDir)
	
	// Conversion en chemin absolu
	absPath, err := filepath.Abs(expanded)
	if err != nil {
		log.Printf("Error expanding base directory %s: %v", b.baseDir, err)
		return expanded, err
	}
	
	return absPath, nil
}

// BaseDir retourne le répertoire de base
func (b *BrowserDef) BaseDir() string {
	return b.baseDir
}

// SetBaseDir définit le répertoire de base
func (b *BrowserDef) SetBaseDir(dir string) {
	b.baseDir = dir
}

// GetBaseDir est un alias pour BaseDir() pour compatibilité
func (b *BrowserDef) GetBaseDir() string {
	return b.baseDir
}

// Ajoutez la variable DefinedBrowsers qui manque
var DefinedBrowsers map[string]*BrowserDef

// Fonction Defined pour compatibilité avec le code existant
func Defined(family ...BrowserFamily) map[string]*BrowserDef {
	if len(family) == 0 {
		return DefinedBrowsers
	}
	
	// Filtrer par famille de navigateur
	filtered := make(map[string]*BrowserDef)
	targetFamily := family[0]
	
	for name, browser := range DefinedBrowsers {
		if browser.Family == targetFamily {
			filtered[name] = browser
		}
	}
	
	return filtered
}

// Définition pour Qutebrowser (avec majuscule pour compatibilité)
var QuteBrowser *BrowserDef

// Initialisez DefinedBrowsers
func init() {
	if DefinedBrowsers == nil {
		DefinedBrowsers = make(map[string]*BrowserDef)
	}
	
	// Ajoutez ici vos définitions de navigateurs par défaut
	// Exemple pour Windows :
	initializeDefaultBrowsers()
}

// Fonction pour initialiser les navigateurs par défaut
func initializeDefaultBrowsers() {
	// Chrome
	DefinedBrowsers["chrome"] = &BrowserDef{
		Flavour: "chrome",
		Family:  ChromeBased,
		baseDir: getDefaultChromeDir(),
	}
	
	// Firefox
	DefinedBrowsers["firefox"] = &BrowserDef{
		Flavour: "firefox", 
		Family:  Mozilla,
		baseDir: getDefaultFirefoxDir(),
	}
	
	// Edge
	DefinedBrowsers["edge"] = &BrowserDef{
		Flavour: "edge",
		Family:  ChromeBased,
		baseDir: getDefaultEdgeDir(),
	}
	
	// Qutebrowser
	quteDef := &BrowserDef{
		Flavour: "qutebrowser",
		Family:  Qutebrowser,
		baseDir: getDefaultQuteDir(),
	}
	DefinedBrowsers["qutebrowser"] = quteDef
	
	// Alias pour compatibilité
	QuteBrowser = quteDef
}

// Fonctions helper pour obtenir les chemins par défaut selon l'OS
func getDefaultChromeDir() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "User Data")
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Google", "Chrome")
	default: // linux
		return filepath.Join(os.Getenv("HOME"), ".config", "google-chrome")
	}
}

func getDefaultFirefoxDir() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "Mozilla", "Firefox")
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Firefox", "Profiles")
	default: // linux
		return filepath.Join(os.Getenv("HOME"), ".mozilla", "firefox")
	}
}

func getDefaultEdgeDir() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Edge", "User Data")
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Microsoft Edge")
	default: // linux
		return filepath.Join(os.Getenv("HOME"), ".config", "microsoft-edge")
	}
}

func getDefaultQuteDir() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "qutebrowser")
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "qutebrowser")
	default: // linux
		return filepath.Join(os.Getenv("HOME"), ".config", "qutebrowser")
	}
}