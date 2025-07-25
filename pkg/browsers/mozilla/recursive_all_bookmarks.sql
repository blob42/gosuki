--
-- Copyright (c) 2023-2025 Chakib Ben Ziane <contact@blob42.xyz> and [`GoSuki` contributors](https://github.com/blob42/gosuki/graphs/contributors).
--
-- All rights reserved.
--
-- SPDX-License-Identifier: AGPL-3.0-or-later
--
-- This file is part of GoSuki.
--
-- GoSuki is free software: you can redistribute it and/or modify it under the terms of the GNU Affero General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.
--
-- GoSuki is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU Affero General Public License for more details.
--
-- You should have received a copy of the GNU Affero General Public License along with gosuki.  If not, see <http://www.gnu.org/licenses/>. 

-- name: recursive-all-bookmarks
-- every bookmark that has a tag and is inside a folder has three entries:

WITH RECURSIVE
	-- default mozilla marketing bookmarks
	marketing_marks(id, url)
	AS (
			SELECT moz_bookmarks.id, moz_places.url FROM moz_places
			JOIN moz_bookmarks ON moz_bookmarks.fk = moz_places.id
			WHERE moz_bookmarks.id < 20
			AND moz_places.url LIKE '%mozilla.org%'
	),
	folder_marks(bid, type, title, folder, parent) 
	AS (
		SELECT id, type, title, title as folder, parent FROM moz_bookmarks 
                WHERE fk IS NULL and parent not in (4,0) -- get all folders
		UNION ALL
		SELECT id, moz_bookmarks.type, moz_bookmarks.title, folder, moz_bookmarks.parent -- get all bookmarks with folder parents
				FROM  moz_bookmarks
				JOIN folder_marks ON moz_bookmarks.parent=bid	
				WHERE id NOT IN (SELECT id FROM marketing_marks) --ignore native mozilla folders
	),
	
	bk_in_folders(id, type, fk, title, parent) AS(
	-- select out all bookmarks inside folders
			SELECT id, type, fk,  title, parent  FROM moz_bookmarks 
					WHERE type = 1 
					AND parent IN (SELECT id FROM moz_bookmarks WHERE fk ISNULL and parent NOT IN (4,0)) -- parent is a folder 
	),
	
	tags AS (
		SELECT id, type, fk, title FROM moz_bookmarks WHERE type = 2 AND parent IN (4,0)
		),
	
	marks(id, type, fk, title, tags, parent, folder)
		AS (
			SELECT id, type, fk, title, title as tags, parent, parent as folder FROM bk_in_folders -- bookmarks			

 			UNION
			-- links between bookmarks and tags
			SELECT id, type, fk, NULL, NULL, parent, parent FROM moz_bookmarks WHERE type = 1 AND fk IS NOT NULL
					
			UNION
			-- get all tags which are tags of bookmarks in folders (pre selected)
			SELECT moz.id, t.type, m.fk, moz.title, t.title, moz.parent, (SELECT title FROM moz_bookmarks WHERE id = moz.parent)
				FROM tags as t 
				JOIN marks as m ON t.id = m.parent
				JOIN  moz_bookmarks as moz ON m.fk = moz.fk
				
		),
		folder_bookmarks_pre(id, type, title, folder, parent, plId)
		AS(
		-- get all bookmarks within folders (moz_bookmarks.fk = null)
			SELECT fm.bid, fm.type, fm.title, fm.folder, fm.parent, moz_places.id as plId
			FROM folder_marks as fm
			JOIN moz_bookmarks ON fm.bid=moz_bookmarks.id
			JOIN  moz_places ON moz_bookmarks.fk = moz_places.id
		),
		folder_bookmarks(id, type, plId, title, tags, folders, parent )
		AS(
			SELECT id, type, plId, title, NULL, group_concat(folder) as folders, parent
				FROM folder_bookmarks_pre GROUP BY plId 
		),
		all_bookmarks AS (
		-- all bookmarks with tags and optionally within folders at the same time
			SELECT 
				marks.fk as placeId,
				marks.title,
				group_concat(marks.tags) as tags,
				marks.parent as parentFolderId,
				group_concat(marks.folder) as folders,
				places.url,
				places.description as plDesc

				FROM marks
				JOIN bk_in_folders ON marks.id = bk_in_folders.id
				JOIN moz_places as places ON marks.fk = places.id
				WHERE marks.type = 2
				GROUP BY placeId

			UNION ALL
			-- All bookmarks only within folders
			SELECT
				fbm.plId as placeId,
				fbm.title,
				NULL, -- bookmarks within folders only do not have tags
				fbm.parent as parentFolderId,
				fbm.folders,
				places.url,
				places.description as plDesc
				
				FROM folder_bookmarks as fbm
				JOIN moz_places as places ON fbm.plId = places.id
		)
		

SELECT
 placeId as plId,
 ifnull(title, "") as title,
 ifnull(group_concat(tags), "") as tags,
 parentfolderId,
(SELECT moz_bookmarks.title FROM moz_bookmarks WHERE id = parentFolderId) as parentFolder,
 group_concat(folders) as folders,
 url,
 ifnull(plDesc, "") as plDesc,
 (SELECT max(moz_bookmarks.lastModified) FROM moz_bookmarks WHERE fk=placeId ) as lastModified
 FROM all_bookmarks
GROUP BY placeId
ORDER BY lastModified
