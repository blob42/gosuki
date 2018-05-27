package main

func logPanic(err error) {
	if err != nil {
		log.Panic(err)
	}
}

func sqlErrorMsg(err error, msg string) {
	if err != nil {
		//sqliteErr := err.(sqlite3.Error)

		//if sqliteErr.Code == sqlite3.ErrConstraint {
		//// Ingore ErrConstraint for now
		////log.Warningf("%s : %s", err, msg)
		//return
		//}

		log.Error(err)
	}
}

func logError(err error) {
	if err != nil {
		log.Errorf("%s", err)
	}
}

func logErrorMsg(err error, msg string) {
	if err != nil {
		log.Errorf("%s : %s", err, msg)
	}
}
