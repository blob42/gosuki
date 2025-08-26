//
// Copyright (c) 2023-2025 Chakib Ben Ziane <contact@blob42.xyz> and [`GoSuki` contributors]
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
	"bufio"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/blob42/gosuki/pkg/logging"
)

var (
	TMPDIR = ""
	// hidden logger
	log = logging.GetLogger("")
)

func CopyFileToDst(src string, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE, 0644)
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
		err = CopyFileToDst(v, dstFile)
		if err != nil {
			return err
		}

	}

	return nil

}

func CopyFileToTmp(source, tmpDir, filename string) error {
	srcFile := filepath.Clean(source)
	dstFile := filepath.Join(tmpDir, filename)

	input, err := os.ReadFile(srcFile)
	if err != nil {
		return err
	}

	return os.WriteFile(dstFile, input, 0644)
}

// TEST:
func CleanFiles() {
	log.Debugf("cleaning files <%s>", TMPDIR)
	err := os.RemoveAll(TMPDIR)
	if err != nil {
		log.Fatal(err)
	}
	err = filepath.Walk("/tmp", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Ignore errors when accessing files we don't have permission for
		}
		if matched, _ := filepath.Match("/tmp/gosuki*", path); matched {
			log.Debugf("removing glob file: %s", path)
			err = os.RemoveAll(path)
			if err != nil {
				return nil // Ignore errors when removing files (e.g., permission issues)
			}
		}
		return nil // Ensure only the first match is processed
	})
	if err != nil && err != filepath.SkipDir {
		log.Fatal(err)
	}
}

func CountLines(f *os.File) (int, error) {
	scanner := bufio.NewScanner(f)
	// 1mb
	scanner.Buffer([]byte{}, 1073741824)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return lineCount, nil
}

func init() {
	var err error
	TMPDIR, err = os.MkdirTemp("", "gosuki*")
	if err != nil {
		log.Fatal(err)
	}
}
