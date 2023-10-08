//
// Copyright ⓒ 2023 Chakib Ben Ziane <contact@blob42.xyz> and [`GoSuki` contributors]
// (https://github.com/blob42/gosuki/graphs/contributors).
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

package utils

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/blob42/gosuki/internal/logging"
)

var (
	TMPDIR = ""
	log    = logging.GetLogger("")
)

func copyFileToDst(src string, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return err
	}

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	err = dstFile.Sync()
	if err != nil {
		return err
	}

	return nil

}

// Copy files from src glob to dst folder
func CopyFilesToTmpFolder(srcglob string, dst string) error {
	matches, err := filepath.Glob(os.ExpandEnv(srcglob))
	if err != nil {
		return err
	}

	for _, v := range matches {
		dstFile := path.Join(dst, path.Base(v))
		err = copyFileToDst(v, dstFile)
		if err != nil {
			return err
		}

	}

	return nil

}

//FIX: this is not always working as expected
//TEST:
func CleanFiles() {
	log.Debugf("Cleaning files <%s>", TMPDIR)
	err := os.RemoveAll(TMPDIR)
	if err != nil {
		log.Fatal(err)
	}
}

func init() {
	var err error
	TMPDIR, err = ioutil.TempDir("", "gosuki*")
	if err != nil {
		log.Fatal(err)
	}
}
