package mozilla

import (
	"bufio"
	"fmt"

	"os"
	"testing"
	"time"
)

const (
	TempFileName = "prefs-test.js"
)

var (
	TestPrefs = map[string]Pref{
		"BOOL": {
			name:   "test.pref.bool",
			value:  true,
			rawval: "true",
		},
		"TRUE": {
			name:   "test.pref.bool.true",
			value:  true,
			rawval: "true",
		},
		"FALSE": {
			name:   "test.pref.bool.false",
			value:  false,
			rawval: "false",
		},
		"NUMBER": {
			name:   "test.pref.number",
			value:  42,
			rawval: "42",
		},
		"STRING": {
			name:   "test.pref.string",
			value:  "test string",
			rawval: "test string",
		},
	}

	TestPrefsBool = map[string]Pref{
		"TRUE":  TestPrefs["TRUE"],
		"FALSE": TestPrefs["FALE"],
	}

	prefsTempFile *os.File
)

type Pref struct {
	name   string
	value  any
	rawval string
}

func writeTestPrefFile(f *os.File, p Pref) {
	switch v := p.value.(type) {
	case string:
		fmt.Fprintf(f, "user_pref(\"%s\", \"%s\");\n", p.name, v)
	case bool:
		fmt.Fprintf(f, "user_pref(\"%s\", %t);\n", p.name, v)
	case int:
		fmt.Fprintf(f, "user_pref(\"%s\", %d);\n", p.name, v)
	default:
		fmt.Fprintf(f, "user_pref(\"%s\", %v);\n", p.name, v)

	}

	err := f.Sync()
	if err != nil {
		panic(err)
	}
}

func resetTestPrefFile(f *os.File) {
	err := f.Truncate(0)
	if err != nil {
		panic(err)
	}

	f.Seek(0, 0)
	f.Sync()
}

func TestFindPref(t *testing.T) {
	resetTestPrefFile(prefsTempFile)

	for _, c := range TestPrefs {
		// Write the pref to pref file
		writeTestPrefFile(prefsTempFile, c)

		t.Run(c.name, func(t *testing.T) {
			res, err := FindPref(prefsTempFile.Name(), c.name)
			if err != nil {
				t.Error(err)
			}

			if res != c.rawval {
				t.Fail()
			}
		})
	}
}

func TestGetPrefBool(t *testing.T) {
	resetTestPrefFile(prefsTempFile)

	for _, c := range []string{"TRUE", "FALSE"} {
		writeTestPrefFile(prefsTempFile, TestPrefs[c])

		t.Run(c, func(t *testing.T) {
			res, err := GetPrefBool(prefsTempFile.Name(), TestPrefs[c].name)
			if err != nil {
				t.Error(err)
			}

			if res != TestPrefs[c].value {
				t.Fail()
			}
		})
	}

	// Not a boolean
	writeTestPrefFile(prefsTempFile, TestPrefs["STRING"])
	t.Run("NOTBOOL", func(t *testing.T) {

		_, err := GetPrefBool(prefsTempFile.Name(), TestPrefs["STRING"].name)
		if err != nil &&
			err != ErrPrefNotBool {
			t.Error(err)
		}
	})

	// Should return false for undefined pref
	t.Run("NOTDEFINED", func(t *testing.T) {

		val, err := GetPrefBool(prefsTempFile.Name(), "not.exists.bool")
		if err != nil && err != ErrPrefNotFound {
			t.Error(err)
		}

		if val != false {
			t.Fail()
		}
	})
}

func TestSetPrefBool(t *testing.T) {

	t.Run("APPEND", func(t *testing.T) {

		resetTestPrefFile(prefsTempFile)

		// Write some data to test the append behavior
		writeTestPrefFile(prefsTempFile, TestPrefs["STRING"])

		setVal, _ := TestPrefs["TRUE"].value.(bool)

		err := SetPrefBool(prefsTempFile.Name(), TestPrefs["TRUE"].name, setVal)

		if err != nil {
			t.Error(err)
		}

		res, err := GetPrefBool(prefsTempFile.Name(), TestPrefs["TRUE"].name)
		if err != nil {
			t.Error(err)
		}

		if res != setVal {
			t.Fail()
		}
	})

	t.Run("REPLACE", func(t *testing.T) {
		resetTestPrefFile(prefsTempFile)
		scanner := bufio.NewScanner(prefsTempFile)

		writeTestPrefFile(prefsTempFile, TestPrefs["STRING"])
		writeTestPrefFile(prefsTempFile, TestPrefs["FALSE"])

		prefsTempFile.Seek(0, 0)

		// Check if line was replaces
		var lines int
		for scanner.Scan() {
			lines++
		}

		err := SetPrefBool(prefsTempFile.Name(), TestPrefs["FALSE"].name, true)
		if err != nil {
			t.Error(err)
		}

		prefsTempFile.Seek(0, 0)
		scanner = bufio.NewScanner(prefsTempFile)
		// Check if line was replaces
		for lines = 0; scanner.Scan(); {
			lines++
		}

		if lines != 2 {
			t.Error("SetPrefBool should replace existing Pref")
		}

		res, err := GetPrefBool(prefsTempFile.Name(), TestPrefs["FALSE"].name)
		if err != nil {
			t.Error(err)
		}

		if !res {
			t.Fail()
		}

		time.Sleep(4 * time.Second)
	})
}

func TestHasPref(t *testing.T) {
	resetTestPrefFile(prefsTempFile)

	writeTestPrefFile(prefsTempFile, TestPrefs["STRING"])

	res, err := HasPref(prefsTempFile.Name(), TestPrefs["STRING"].name)
	if err != nil {
		t.Error(err)
	}

	if !res {
		t.Fail()
	}

}
