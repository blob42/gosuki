//
//  Copyright (c) 2024-2025 Chakib Ben Ziane <contact@blob42.xyz>  and [`gosuki` contributors](https://github.com/blob42/gosuki/graphs/contributors).
//  All rights reserved.
//
//  SPDX-License-Identifier: AGPL-3.0-or-later
//
//  This file is part of GoSuki.
//
//  GoSuki is free software: you can redistribute it and/or modify
//  it under the terms of the GNU Affero General Public License as
//  published by the Free Software Foundation, either version 3 of the
//  License, or (at your option) any later version.
//
//  GoSuki is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of
//  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//  GNU Affero General Public License for more details.
//
//  You should have received a copy of the GNU Affero General Public License
//  along with gosuki.  If not, see <http://www.gnu.org/licenses/>.
//

package main

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gofrs/uuid"

	"github.com/blob42/gosuki/internal/database"
	"github.com/blob42/gosuki/internal/utils"
	"github.com/blob42/gosuki/internal/webui"
	"github.com/blob42/gosuki/pkg/build"
	"github.com/blob42/gosuki/pkg/config"
	"github.com/blob42/gosuki/pkg/events"
	"github.com/blob42/gosuki/pkg/logging"
	"github.com/blob42/gosuki/pkg/manager"
	"github.com/blob42/gosuki/pkg/modules"
	"github.com/blob42/gosuki/pkg/profiles"
	"github.com/blob42/gosuki/pkg/watch"
)

const (
	maxWidth   = 80
	tickRate   = 20
	nLogLines  = 10
	statusChar = "•"
)

var (
	ModMsgQ    = make(chan modules.ModMsg, 64)
	tuiOptions = []tea.ProgramOption{
		// tea.WithAltScreen(),
	}
)

type module struct {
	modules.Module
	id modules.ModID

	progress progress.Model
	modState
}

type modState struct {
	totalCount uint
	curCount   uint
	enabled    bool
}

// Meta struct that holds progress for all profiles belonging to
// a browser. A browser flavour is considered its own browser instance
type browser struct {
	id string

	// Covers both profiles and flavours
	// profiles  []progress.Model
	instances     []modules.BrowserModule
	profileStates map[modules.Module]*modState
	progress      progress.Model
	modState
}

func updateBrowserProgress(b *browser, msg events.ProgressUpdateMsg) tea.Cmd {
	profState, exists := b.profileStates[msg.Instance]
	if !exists {
		panic("instance does exist")
	}

	if msg.NewBk {
		profState.curCount += 1
		profState.totalCount = profState.curCount
		b.totalCount += 1
	} else {
		profState.curCount = msg.CurrentCount
		profState.totalCount = msg.Total
	}

	var newCurCount uint
	for _, prf := range b.profileStates {
		newCurCount += prf.curCount
	}
	b.curCount = newCurCount
	b.totalCount = 0
	for _, p := range b.profileStates {
		b.totalCount += p.totalCount
	}

	// logging.FDebugf("/tmp/gosuki_cur_count", "browser count: %d", newCurCount)

	percent := float64(newCurCount) / float64(b.totalCount)

	return b.progress.SetPercent(percent)

}

func handleModMsg(m tuiModel, msg modules.ModMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case modules.MsgSyncPeers:
		if peers, ok := msg.Payload.(map[uuid.UUID]string); ok {
			m.syncPeers = peers
		}
	}

	return m, nil
}

type winSize struct {
	width  int
	height int
}

type daemonState int

const (
	DaemonLoading = iota
	DaemonReady
	DaemonFail
)

type dbState struct {
	init     bool
	totalBks uint
}

type tuiModel struct {
	ctx         context.Context
	db          dbState
	initFunc    initFunc
	logBuffer   *logging.TailBuffer
	showLog     bool
	collapse    bool
	manager     *manager.Manager
	modules     map[string]*module
	modKeys     []string
	browsers    map[string]*browser
	browserKeys []string
	windowSize  winSize
	keymap      keymap
	help        help.Model
	daemon      daemonState
	syncPeers   map[uuid.UUID]string
}

type keymap struct {
	quit      key.Binding
	toggleLog key.Binding
	collapse  key.Binding
	expand    key.Binding
}

type ErrMsg error
type TickMsg time.Time
type DBTickMsg time.Time
type DaemonStartedMsg struct{}
type DaemonStoppedMsg struct{}
type initFunc func(tea.Model) tea.Cmd

var (
	progressOpts = []progress.Option{
		progress.WithDefaultGradient(),
		progress.WithFillCharacters('━', '╺'),
		progress.WithoutPercentage(),
	}

	defaultTextColor = lipgloss.NewStyle().Foreground(
		lipgloss.AdaptiveColor{Light: "240", Dark: "255"},
	)

	logSectionStyle = lipgloss.NewStyle().
		// Background(lipgloss.Color("22")).
		MarginTop(2).
		Padding(1, 0, 0, 2).
		Width(maxWidth)

	titleStyle = defaultTextColor.
			PaddingLeft(2).
			MarginTop(1).
			MarginBottom(1)

	headerPadding = 1
	headerStyle   = lipgloss.NewStyle().
			Background(lipgloss.Color("60")).
			PaddingLeft(headerPadding).
			PaddingRight(1).
			MarginBottom(2)
		// Border(lipgloss.NormalBorder(), false, false, true)

	infoLabelStyle = defaultTextColor.
			AlignHorizontal(lipgloss.Right).
		// Background(lipgloss.Color("73")).
		MarginTop(1).
		Width(12).
		MarginLeft(2).
		MarginRight(3)

	labelStyle = defaultTextColor.
			MarginLeft(1).
			Width(10).
		// Background(lipgloss.Color("63")).
		Align(lipgloss.Right)

	urlCountStyle = defaultTextColor.
			PaddingLeft(1).
		// Background(lipgloss.Color("22")).
		Width(20)

	totalLabelStyle = defaultTextColor.
			Width(0).
			MarginLeft(labelStyle.GetWidth() + 2).
		// Background(lipgloss.Color("22")).
		MarginTop(1)

	ProgressSectionStyle = lipgloss.NewStyle().
				MarginTop(2)
		// Background(lipgloss.Color("24"))
		// Width(60)
		// MarginBottom(1)

	ProgressBarStyle = lipgloss.NewStyle().
				PaddingLeft(2)

	faintTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	moduleStatusStyle = lipgloss.NewStyle().
				Align(lipgloss.Left).
				Bold(true)

	moduleOnStyle = moduleStatusStyle.
			Foreground(lipgloss.Color("120"))

	moduleOffStyle = moduleStatusStyle.
			Foreground(lipgloss.Color("203"))

	moduleStates = map[bool]lipgloss.Style{
		true:  moduleOnStyle,
		false: moduleOffStyle,
	}

	helpStyle = lipgloss.NewStyle().Align(lipgloss.Bottom).
			MarginTop(1).
			PaddingLeft(4)
)

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*tickRate, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func dbTickCmd() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return DBTickMsg(t)
	})
}

// Init implements tea.Model.
func (m tuiModel) Init() tea.Cmd {
	return tea.Batch(
		tea.ClearScreen,
		m.initFunc(m),
		tickCmd(),
		dbTickCmd(),
	)
}

func (m tuiModel) HelpView() string {
	return "\n" + m.help.ShortHelpView([]key.Binding{
		m.keymap.quit,
		m.keymap.toggleLog,
		m.keymap.collapse,
		m.keymap.expand,
	})
}

func formatSyncPeers(m tuiModel) string {
	peers := slices.Collect(maps.Values(m.syncPeers))
	slices.Sort(peers)
	return strings.Join(peers, ", ")

}

func setupModProgress(m tuiModel, r watch.WatchRunner) (tea.Model, tea.Cmd) {
	mod, ok := r.(modules.Module)
	if !ok {
		return m, nil
	}

	id := string(mod.ModInfo().ID)
	br, isB := mod.(modules.BrowserModule)
	if isB {
		// _, isProfMgr := br.(profiles.ProfileManager)

		if _, exists := m.browsers[id]; !exists {
			panic("missing browser map entry")
		}

		// if isProfMgr {
		// flavs := pm.ListFlavours()
		// for _, f := range flavs {
		// 	_, err := pm.GetProfiles(f.Name)
		// 	if err != nil {
		// 		panic(err)
		// 	}
		// }
		// }

		// track this instance (ie profile)
		m.browsers[id].instances = append(m.browsers[id].instances, br)

		// init profile state
		m.browsers[id].profileStates[br] = &modState{}
	} else {
		if _, exists := m.modules[id]; !exists {
			panic("missing module map entry")
		}

		m.modules[id] = &module{
			Module:   mod,
			progress: progress.New(progressOpts...),
			id:       mod.ModInfo().ID,
		}
	}

	return m, nil
}

// Update implements tea.Model.
func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.quit):
			return m, tea.Sequence(func() tea.Msg {
				log.Info("stopping GoSuki ...")
				go m.manager.Shutdown()
				<-m.manager.Quit
				utils.CleanFiles()
				return nil
			}, tea.Quit)
		case key.Matches(msg, m.keymap.toggleLog):
			m.showLog = !m.showLog
		case key.Matches(msg, m.keymap.expand):
			m.collapse = false
			m.keymap.expand.SetEnabled(false)
			m.keymap.collapse.SetEnabled(true)
		case key.Matches(msg, m.keymap.collapse):
			m.collapse = true
			m.keymap.expand.SetEnabled(true)
			m.keymap.collapse.SetEnabled(false)
		}

	case TickMsg:
		cmds := []tea.Cmd{tickCmd()}

		// watch for events
		cmds = append(
			cmds,
			// watch tui event bus
			func() tea.Msg { return <-events.TUIBus },

			// watch module messages
			func() tea.Msg { return <-ModMsgQ },
		)

		return m, tea.Batch(cmds...)

	// ticker for checking db data
	case DBTickMsg:
		var err error
		m.db.totalBks, err = database.L2Cache.TotalBookmarks(m.ctx)
		if err != nil {
			log.Errorf("counting bookmarks from cache: %s", err)
		}

		return m, dbTickCmd()

	case modules.ModMsg:
		if msg.To == "tui" {
			return handleModMsg(m, msg)
		}
		return m, nil

	// New module instance
	case events.RunnerStarted:
		return setupModProgress(m, msg.WatchRunner)

	case events.StartedLoadingMsg:
		_, isBr := m.browsers[string(msg.ID)]

		// simple module
		if !isBr {
			_, isMod := m.modules[string(msg.ID)]
			if !isMod {
				errMsg := fmt.Sprintf("%s: module not recognized", msg.ID)
				panic(errMsg)
			}
			m.modules[string(msg.ID)].enabled = true
		} else {
			m.browsers[string(msg.ID)].enabled = true
		}

		return m, nil

	case events.ProgressUpdateMsg:

		// logging.FDebugf("/tmp/gosuki_progress", "%#v", msg)
		// browser
		if b, ok := m.browsers[string(msg.ID)]; ok {
			return m, updateBrowserProgress(b, msg)

			// simple module
		} else if mod, ok := m.modules[string(msg.ID)]; ok {
			m.modules[string(msg.ID)].enabled = true
			mod.curCount = msg.CurrentCount
			mod.totalCount = msg.Total
			percent := float64(mod.curCount) / float64(mod.totalCount)
			return m, mod.progress.SetPercent(percent)
		} else {
			log.Error("unrecognized module", "module", msg.ID)
		}

	case tea.WindowSizeMsg:
		titleStyle = titleStyle.Width(msg.Width)

		for _, m := range m.browsers {
			m.progress.Width = min(int(math.Min(float64(msg.Width/2), 80)), maxWidth)
		}

		for _, m := range m.modules {
			m.progress.Width = min(int(math.Min(float64(msg.Width/2), 80)), maxWidth)
		}

		m.windowSize.height = msg.Height
		m.windowSize.width = msg.Width

		logSectionStyle = logSectionStyle.Width(msg.Width)
		// ProgressSectionStyle = ProgressBarStyle.Width(msg.Width - 10)
		return m, nil

	case ErrMsg:
		fmt.Fprintf(os.Stderr, "tui error: %s", msg.Error())
		return m, tea.Quit

	case DaemonStartedMsg:
		m.daemon = DaemonReady

	case progress.FrameMsg:
		cmds := []tea.Cmd{}
		for _, b := range m.browsers {
			progressModel, cmd := b.progress.Update(msg)
			b.progress = progressModel.(progress.Model)
			cmds = append(cmds, cmd)
		}

		for _, m := range m.modules {
			progressModel, cmd := m.progress.Update(msg)
			m.progress = progressModel.(progress.Model)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	default:
		return m, nil
	}

	return m, nil
}

// View implements tea.Model.
func (m tuiModel) View() string {
	doc := strings.Builder{}

	doc.WriteString(m.headerView())
	// doc.WriteString(titleStyle.Render(fmt.Sprintf("gosuki %s", build.Version())))

	modStatus := map[string]bool{}

	// get web UI status
	for name := range m.manager.Units() {
		if strings.HasPrefix(name, "webui") {
			modStatus["webui"] = true
		}
		if strings.HasPrefix(name, "p2p-sync") {
			modStatus["p2p-sync"] = true
		}
	}

	uiSection := strings.Builder{}
	uiSection.WriteString(infoLabelStyle.Render(
		defaultTextColor.Render("web-ui"),
		moduleStates[modStatus["webui"]].Render(statusChar),
	))

	// uiSection.WriteString(infoLabelStyle.Render(
	// 	fmt.Sprintf("%s web ui :", webUILabel)))
	uiSection.WriteString(defaultTextColor.Render(fmt.Sprintf("http://%s", webui.BindAddr)))
	p2psyncSec := strings.Builder{}
	p2psyncSec.WriteString(infoLabelStyle.Render(
		defaultTextColor.Render("p2p-sync"),
		moduleStates[modStatus["p2p-sync"]].Render(statusChar),
	))

	if len(m.syncPeers) > 0 {
		p2psyncSec.WriteString(defaultTextColor.Render("synced with: "))
		p2psyncSec.WriteString(defaultTextColor.Render(formatSyncPeers(m)))
	}

	progressSection := strings.Builder{}

	var totalURLCount uint

	// Browser modules
	for _, name := range m.browserKeys {
		br, ok := m.browsers[name]
		if !ok || !br.enabled {
			continue
		}
		totalURLCount += uint(br.totalCount)
		progressSection.WriteString(labelStyle.Render(name))
		progressSection.WriteString(ProgressBarStyle.Render(br.progress.View()))
		progressSection.WriteString(urlCountStyle.Render(
			fmt.Sprintf("%4d/%-4d", br.curCount, br.totalCount)),
		)

		// handle multiple browser instances such as profiles and flavours
		if len(br.instances) > 1 && !m.collapse {
			for _, brr := range br.instances {
				bpm, ok := brr.(profiles.ProfileManager)
				if !ok {
					continue
				}
				profile := bpm.GetProfile()
				// if profile == nil {
				// 	continue
				// }

				progressSection.WriteString("\n")

				labelLineStyle := labelStyle.
					Foreground(lipgloss.Color("99"))

				progressSection.WriteString(labelLineStyle.Render("└─") + " ")

				if profile.IsCustom {
					progressSection.WriteString(defaultTextColor.Render("(custom)"))
				}
				progressSection.WriteString(faintTextStyle.Render(profile.ShortBaseDir() + " "))
				progressSection.WriteString(defaultTextColor.Render(profile.Path))

				// if cfg.Flavour != nil {
				// 	progressSection.WriteString(cfg.Flavour.Name + " ")
				// }
			}
			progressSection.WriteByte('\n')
		} else {
			progressSection.WriteByte('\n')
		}
	}

	// Simple modules
	for _, name := range m.modKeys {
		mod := m.modules[name]
		if !mod.enabled {
			continue
		}
		totalURLCount += uint(mod.totalCount)
		progressSection.WriteString(labelStyle.Render(name))
		progressSection.WriteString(ProgressBarStyle.
			Render(mod.progress.View()))
		progressSection.WriteString(urlCountStyle.Render(
			fmt.Sprintf("%4d/%-4d", mod.curCount, mod.totalCount)),
		)
		progressSection.WriteByte('\n')
	}

	doc.WriteString(uiSection.String())
	if modStatus["p2p-sync"] {
		doc.WriteString(p2psyncSec.String())
	}
	// doc.WriteString(infoLabelStyle.Render("modules  "))
	// doc.WriteString(defaultTextColor.Render(fmt.Sprintf("%d", len(m.modules)+len(m.browsers))))
	doc.WriteString(ProgressSectionStyle.
		// Height(m.windowSize.height / 2).
		Render(progressSection.String()))

	totalLabelStyle = totalLabelStyle.MarginLeft(labelStyle.GetWidth())
	doc.WriteString(totalLabelStyle.Render(fmt.Sprintf("bookmarks: %d loaded / %d (db)", totalURLCount, m.db.totalBks)))
	// doc.WriteString(fmt.Sprintf("%d", totalUrlCount))

	if m.showLog {
		doc.WriteString(logSectionStyle.Render(strings.Join(m.logBuffer.Lines(), "\n")) + "\n")
	}

	doc.WriteString(helpStyle.Render(m.HelpView()))
	return doc.String()
}

func (m tuiModel) headerView() string {
	logo := lipgloss.NewStyle().Render(fmt.Sprintf("gosuki:%s", build.Version()))

	dbPath := lipgloss.NewStyle().Render(utils.Shorten(config.DBPath))

	bookmarks := lipgloss.NewStyle().
		PaddingRight(headerPadding * 2).
		Render(fmt.Sprintf(" • bks: %4d", m.db.totalBks))

	space := strings.Repeat(" ", max(
		0,
		m.windowSize.width-(lipgloss.Width(logo+bookmarks+dbPath)),
	))
	line := lipgloss.JoinHorizontal(lipgloss.Center, logo, space, dbPath, bookmarks)
	return headerStyle.Render(line)
}

type tui struct {
	model tuiModel
	opts  []tea.ProgramOption
}

func NewTUI(
	ctx context.Context,
	initFunc initFunc,
	manager *manager.Manager,
	opts ...tea.ProgramOption,
) (*tui, error) {
	var err error

	mods := map[string]*module{}
	browsers := make(map[string]*browser)
	browserKeys := []string{}
	modKyes := []string{}

	for _, mod := range modules.GetModules() {
		modInfo := mod.ModInfo()
		name := string(modInfo.ID)

		// Extract module/browser name, convert flavours to name as well
		b, isBrowser := modInfo.New().(modules.BrowserModule)
		if isBrowser {
			browserKeys = append(browserKeys, b.Config().Name)
			browsers[b.Config().Name] = &browser{
				id:       b.Config().Name,
				progress: progress.New(progressOpts...),
				// profiles:  []progress.Model{},
				instances:     []modules.BrowserModule{},
				profileStates: map[modules.Module]*modState{},
			}
		} else {
			modKyes = append(modKyes, name)
			mods[name] = &module{
				id:       modules.ModID(name),
				Module:   mod,
				progress: progress.New(progressOpts...),
			}
		}

		modLabelWidth := labelStyle.GetWidth()
		modLabelWidth = int(math.Max(float64(modLabelWidth), float64(len(name))))
		labelStyle = labelStyle.Width(modLabelWidth)
	}

	tui := &tui{
		model: tuiModel{
			ctx:         ctx,
			db:          dbState{},
			initFunc:    initFunc,
			manager:     manager,
			logBuffer:   logging.NewTailBuffer(nLogLines),
			modules:     mods,
			browsers:    browsers,
			browserKeys: browserKeys,
			modKeys:     modKyes,
			windowSize:  winSize{},
			showLog:     false,
			collapse:    false,
			keymap: keymap{
				quit: key.NewBinding(
					key.WithKeys("q", "esc", "ctrl+c"),
					key.WithHelp("q/esc", "quit"),
				),
				toggleLog: key.NewBinding(
					key.WithKeys("l"),
					key.WithHelp("l", "show log"),
				),
				expand: key.NewBinding(
					key.WithKeys(" "),
					key.WithHelp("space", "expand"),
					key.WithDisabled(),
				),
				collapse: key.NewBinding(
					key.WithKeys(" "),
					key.WithHelp("space", "collapse"),
				),
			},
			help:   help.New(),
			daemon: DaemonLoading,
		},
		opts: opts,
	}
	tui.model.db.totalBks, err = database.L2Cache.TotalBookmarks(ctx)
	if err != nil {
		return nil, err
	}

	return tui, nil
}

func (tui *tui) Run() error {
	_, err := tea.NewProgram(tui.model, tui.opts...).Run()
	if err != nil {
		return errors.New("could not start TUI")
	}
	return nil
}
