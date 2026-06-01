//
//  Copyright (c) 2025 Chakib Ben Ziane <contact@blob42.xyz>  and [`gosuki` contributors](https://github.com/blob42/gosuki/graphs/contributors).
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

//go:build unix

package database

import (
	"os"

	"golang.org/x/sys/unix"
)

type LockChecker interface {
	Locked() (bool, error)
}

type VFSLockChecker struct {
	path string
}

func (checker *VFSLockChecker) Locked() (bool, error) {

	f, err := os.Open(checker.path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	// Get the the lock mode
	var lock unix.Flock_t
	// See man (fcntl)
	unix.FcntlFlock(f.Fd(), unix.F_GETLK, &lock)

	// Check if lock is F_RDLCK (non-exclusive) or F_WRLCK (exclusive)
	if lock.Type == unix.F_RDLCK {
		//log.Debug("Lock is F_RDLCK")
		return false, nil
	}

	if lock.Type == unix.F_WRLCK {
		//log.Debug("Lock is F_WRLCK (locked !)")
		return true, nil
	}

	return false, nil

}
