// Copyright (c) 2025 Chakib Ben Ziane <contact@blob42.xyz>  and [`gosuki` contributors](https://github.com/blob42/gosuki/graphs/contributors).
// All rights reserved.
//
// All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// This file is part of GoSuki.
//
// GoSuki is free software: you can redistribute it and/or modify it under the terms of
// the GNU Affero General Public License as published by the Free Software Foundation,
// either version 3 of the License, or (at your option) any later version.
//
// GoSuki is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
// without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR
// PURPOSE.  See the GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License along with
// gosuki.  If not, see <http://www.gnu.org/licenses/>.

// https://github.com/lightningnetwork/lnd/blob/master/build/version.go#L66
// https://raw.githubusercontent.com/lightningnetwork/lnd/refs/heads/master/build/version.go
package build

import (
	"runtime"
	"runtime/debug"
	"strings"
)

// These constants define the application version and follow the semantic
// versioning 2.0.0 spec (http://semver.org/).
var (
	// Describe stores the current commit of this build, which includes the
	// most recent tag, the number of commits since that tag (if non-zero),
	// the commit hash, and a dirty marker. This should be set using the
	// -ldflags during compilation.
	Describe string

	// CommitHash stores the current commit hash of this build.
	CommitHash string

	// RawTags contains the raw set of build tags, separated by commas.
	RawTags string

	// GoVersion stores the go version that the executable was compiled
	// with.
	GoVersion string

	// PackageVersion stores the version of the package itself.
	PackageVersion string
)

// Version returns the application version as a properly formed string per the
// semantic versioning 2.0.0 spec (http://semver.org/).
func Version() string {
	if Describe == "" {
		return PackageVersion
	}

	return Describe
}

func DebugInfo() {
	println("GoSuki Version:", Version())
	println("Commit Hash:", CommitHash)
	println("Go Version:", GoVersion)
	println("Package Version:", PackageVersion)
	println("Build Tags:", strings.Join(Tags(), ", "))
	println("OS:", runtime.GOOS)
	println("Architecture:", runtime.GOARCH)
}

// Tags returns the list of build tags that were compiled into the executable.
func Tags() []string {
	if len(RawTags) == 0 {
		return []string{}
	}

	return strings.Split(RawTags, ",")
}

// Get build information from the runtime.
func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		GoVersion = info.GoVersion
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				CommitHash = setting.Value

			case "-tags":
				RawTags = setting.Value
			}
		}
		if info.Main.Version != "" {
			PackageVersion = info.Main.Version
		}
	}
}
