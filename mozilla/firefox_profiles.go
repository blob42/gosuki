package mozilla

import (
	"errors"
	"fmt"
	"gomark/config"
	"gomark/profiles"
	"gomark/utils"
	"os"
	"path/filepath"
	"regexp"

	ini "gopkg.in/ini.v1"
)

type ProfileManager = profiles.ProfileManager
type INIProfileLoader = profiles.INIProfileLoader
type PathGetter = profiles.PathGetter

const (
	ProfilesFile = "profiles.ini"
)

var (
	ReIniProfiles = regexp.MustCompile(`(?i)profile`)

	firefoxProfile = &INIProfileLoader{
		//BasePath to be set at runtime in init
		ProfilesFile: ProfilesFile,
	}

	FirefoxProfileManager = &FFProfileManager{
		pathGetter: firefoxProfile,
	}

	ErrProfilesIni      = errors.New("Could not parse Firefox profiles.ini file")
	ErrNoDefaultProfile = errors.New("No default profile found")
)

type FFProfileManager struct {
	profilesFile *ini.File
	pathGetter   PathGetter
	ProfileManager
}

func (pm *FFProfileManager) loadProfile() error {

	log.Debugf("loading profile from <%s>", pm.pathGetter.Get())
	pFile, err := ini.Load(pm.pathGetter.Get())
	if err != nil {
		return err
	}

	pm.profilesFile = pFile
	return nil
}

func (pm *FFProfileManager) GetProfiles() ([]*profiles.Profile, error) {
	pm.loadProfile()
	sections := pm.profilesFile.Sections()
	var filtered []*ini.Section
	var result []*profiles.Profile
	for _, section := range sections {
		if ReIniProfiles.MatchString(section.Name()) {
			filtered = append(filtered, section)

			p := &profiles.Profile{
				Id: section.Name(),
			}

			err := section.MapTo(p)
			if err != nil {
				return nil, err
			}

			result = append(result, p)

		}
	}

	return result, nil
}

func (pm *FFProfileManager) GetDefaultProfilePath() (string, error) {
	log.Debugf("using config dir %s", ConfigFolder)
	p, err := pm.GetDefaultProfile()
	if err != nil {
		return "", err
	}
	return filepath.Join(ConfigFolder, p.Path), nil
}

func (pm *FFProfileManager) GetProfileByName(name string) (*profiles.Profile, error) {
	profs, err := pm.GetProfiles()
	if err != nil {
		return nil, err
	}

	for _, p := range profs {
		if p.Name == name {
			return p, nil
		}
	}

	return nil, fmt.Errorf("Profile %s not found", name)
}

func (pm *FFProfileManager) GetDefaultProfile() (*profiles.Profile, error) {
	profs, err := pm.GetProfiles()
	if err != nil {
		return nil, err
	}

	log.Debugf("looking for profile %s", Config.DefaultProfile)
	for _, p := range profs {
		if p.Name == Config.DefaultProfile {
			return p, nil
		}
	}

	return nil, ErrNoDefaultProfile
}

func (pm *FFProfileManager) ListProfiles() ([]string, error) {
	pm.loadProfile()
	sections := pm.profilesFile.SectionStrings()
	var result []string
	for _, s := range sections {
		if ReIniProfiles.MatchString(s) {
			result = append(result, s)
		}
	}

	if len(result) == 0 {
		return nil, ErrProfilesIni
	}

	return result, nil
}

func initFirefoxConfig() {
	log.Debug("initializing firefox config")
	ConfigFolder = filepath.Join(os.ExpandEnv(ConfigFolder))

	// Check if base folder exists
	configFolderExists, err := utils.CheckDirExists(ConfigFolder)
	if !configFolderExists {
		log.Criticalf("the base firefox folder <%s> does not exist",
			ConfigFolder)
	}

	if err != nil {
		log.Critical(err)
	}

	firefoxProfile.BasePath = ConfigFolder

	//log.Debug(Config)
	bookmarkDir, err := FirefoxProfileManager.GetDefaultProfilePath()
	if err != nil {
		log.Error(err)
	}

	log.Debugf("Using default profile %s", bookmarkDir)
	SetBookmarkDir(bookmarkDir)
}

func init() {
	config.RegisterConfReadyHooks(initFirefoxConfig)
}

//TODO: fix error bookmark path on loading
