package rotatelogs_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestSatisfiesIOWriter(t *testing.T) {
	var w io.Writer
	w, _ = rotatelogs.New("/foo/bar")
	_ = w
}

func TestSatisfiesIOCloser(t *testing.T) {
	var c io.Closer
	c, _ = rotatelogs.New("/foo/bar")
	_ = c
}

func TestLogRotate(t *testing.T) {
	dir, err := ioutil.TempDir("", "file-rotatelogs-test")
	if !assert.NoError(t, err, "creating temporary directory should succeed") {
		return
	}
	defer os.RemoveAll(dir)

	// Change current time, so we can safely purge old logs
	dummyTime := time.Now().Add(-7 * 24 * time.Hour)
	dummyTime = dummyTime.Add(time.Duration(-1 * dummyTime.Nanosecond()))
	clock := clockwork.NewFakeClockAt(dummyTime)
	linkName := filepath.Join(dir, "log")
	rl, err := rotatelogs.New(
		filepath.Join(dir, "log%Y%m%d%H%M%S"),
		rotatelogs.WithClock(clock),
		rotatelogs.WithMaxAge(24*time.Hour),
		rotatelogs.WithLinkName(linkName),
	)
	if !assert.NoError(t, err, `rotatelogs.New should succeed`) {
		return
	}
	defer rl.Close()

	str := "Hello, World"
	n, err := rl.Write([]byte(str))
	if !assert.NoError(t, err, "rl.Write should succeed") {
		return
	}

	if !assert.Len(t, str, n, "rl.Write should succeed") {
		return
	}

	fn := rl.CurrentFileName()
	if fn == "" {
		t.Errorf("Could not get filename %s", fn)
	}

	content, err := ioutil.ReadFile(fn)
	if err != nil {
		t.Errorf("Failed to read file %s: %s", fn, err)
	}

	if string(content) != str {
		t.Errorf(`File content does not match (was "%s")`, content)
	}

	err = os.Chtimes(fn, dummyTime, dummyTime)
	if err != nil {
		t.Errorf("Failed to change access/modification times for %s: %s", fn, err)
	}

	fi, err := os.Stat(fn)
	if err != nil {
		t.Errorf("Failed to stat %s: %s", fn, err)
	}

	if !fi.ModTime().Equal(dummyTime) {
		t.Errorf("Failed to chtime for %s (expected %s, got %s)", fn, fi.ModTime(), dummyTime)
	}

	clock.Advance(time.Duration(7 * 24 * time.Hour))

	// This next Write() should trigger Rotate()
	rl.Write([]byte(str))
	newfn := rl.CurrentFileName()
	if newfn == fn {
		t.Errorf(`New file name and old file name should not match ("%s" != "%s")`, fn, newfn)
	}

	content, err = ioutil.ReadFile(newfn)
	if err != nil {
		t.Errorf("Failed to read file %s: %s", newfn, err)
	}

	if string(content) != str {
		t.Errorf(`File content does not match (was "%s")`, content)
	}

	time.Sleep(time.Second)

	// fn was declared above, before mocking CurrentTime
	// Old files should have been unlinked
	_, err = os.Stat(fn)
	if !assert.Error(t, err, "os.Stat should have failed") {
		return
	}

	linkDest, err := os.Readlink(linkName)
	if err != nil {
		t.Errorf("Failed to readlink %s: %s", linkName, err)
	}

	if linkDest != newfn {
		t.Errorf(`Symlink destination does not match expected filename ("%s" != "%s")`, newfn, linkDest)
	}
}

func CreateRotationTestFile(dir string, base time.Time, d time.Duration, n int) {
	timestamp := base
	for i := 0; i < n; i++ {
		// %Y%m%d%H%M%S
		suffix := timestamp.Format("20060102150405")
		path := filepath.Join(dir, "log"+suffix)
		ioutil.WriteFile(path, []byte("rotation test file\n"), os.ModePerm)
		os.Chtimes(path, timestamp, timestamp)
		timestamp = timestamp.Add(d)
	}
}

func TestLogRotationCount(t *testing.T) {
	dir, err := ioutil.TempDir("", "file-rotatelogs-rotationcount-test")
	if !assert.NoError(t, err, "creating temporary directory should succeed") {
		return
	}
	defer os.RemoveAll(dir)

	dummyTime := time.Now().Add(-7 * 24 * time.Hour)
	dummyTime = dummyTime.Add(time.Duration(-1 * dummyTime.Nanosecond()))
	clock := clockwork.NewFakeClockAt(dummyTime)

	t.Run("Either maxAge or rotationCount should be set", func(t *testing.T) {
		rl, err := rotatelogs.New(
			filepath.Join(dir, "log%Y%m%d%H%M%S"),
			rotatelogs.WithClock(clock),
			rotatelogs.WithMaxAge(time.Duration(0)),
			rotatelogs.WithRotationCount(0),
		)
		if !assert.NoError(t, err, `Both of maxAge and rotationCount is disabled`) {
			return
		}
		defer rl.Close()
	})

	t.Run("Either maxAge or rotationCount should be set", func(t *testing.T) {
		rl, err := rotatelogs.New(
			filepath.Join(dir, "log%Y%m%d%H%M%S"),
			rotatelogs.WithClock(clock),
			rotatelogs.WithMaxAge(1),
			rotatelogs.WithRotationCount(1),
		)
		if !assert.Error(t, err, `Both of maxAge and rotationCount is enabled`) {
			return
		}
		if rl != nil {
			defer rl.Close()
		}
	})

	t.Run("Only latest log file is kept", func(t *testing.T) {
		rl, err := rotatelogs.New(
			filepath.Join(dir, "log%Y%m%d%H%M%S"),
			rotatelogs.WithClock(clock),
			rotatelogs.WithMaxAge(-1),
			rotatelogs.WithRotationCount(1),
		)
		if !assert.NoError(t, err, `rotatelogs.New should succeed`) {
			return
		}
		defer rl.Close()

		n, err := rl.Write([]byte("dummy"))
		if !assert.NoError(t, err, "rl.Write should succeed") {
			return
		}
		if !assert.Len(t, "dummy", n, "rl.Write should succeed") {
			return
		}
		time.Sleep(time.Second)
		files, err := filepath.Glob(filepath.Join(dir, "log*"))
		if !assert.Equal(t, 1, len(files), "Only latest log is kept") {
			return
		}
	})

	t.Run("Old log files are purged except 2 log files", func(t *testing.T) {
		CreateRotationTestFile(dir, dummyTime, time.Duration(time.Hour), 5)
		rl, err := rotatelogs.New(
			filepath.Join(dir, "log%Y%m%d%H%M%S"),
			rotatelogs.WithClock(clock),
			rotatelogs.WithMaxAge(-1),
			rotatelogs.WithRotationCount(2),
		)
		if !assert.NoError(t, err, `rotatelogs.New should succeed`) {
			return
		}
		defer rl.Close()

		n, err := rl.Write([]byte("dummy"))
		if !assert.NoError(t, err, "rl.Write should succeed") {
			return
		}
		if !assert.Len(t, "dummy", n, "rl.Write should succeed") {
			return
		}
		time.Sleep(time.Second)
		files, err := filepath.Glob(filepath.Join(dir, "log*"))
		if !assert.Equal(t, 2, len(files), "One file is kept") {
			return
		}
	})

}

func TestLogSetOutput(t *testing.T) {
	dir, err := ioutil.TempDir("", "file-rotatelogs-test")
	if err != nil {
		t.Errorf("Failed to create temporary directory: %s", err)
	}
	defer os.RemoveAll(dir)

	rl, err := rotatelogs.New(filepath.Join(dir, "log%Y%m%d%H%M%S"))
	if !assert.NoError(t, err, `rotatelogs.New should succeed`) {
		return
	}
	defer rl.Close()

	log.SetOutput(rl)
	defer log.SetOutput(os.Stderr)

	str := "Hello, World"
	log.Print(str)

	fn := rl.CurrentFileName()
	if fn == "" {
		t.Errorf("Could not get filename %s", fn)
	}

	content, err := ioutil.ReadFile(fn)
	if err != nil {
		t.Errorf("Failed to read file %s: %s", fn, err)
	}

	if !strings.Contains(string(content), str) {
		t.Errorf(`File content does not contain "%s" (was "%s")`, str, content)
	}
}

func TestGHIssue16(t *testing.T) {
	defer func() {
		if v := recover(); v != nil {
			assert.NoError(t, errors.Errorf("%s", v), "error should be nil")
		}
	}()

	dir, err := ioutil.TempDir("", "file-rotatelogs-gh16")
	if !assert.NoError(t, err, `creating temporary directory should succeed`) {
		return
	}
	defer os.RemoveAll(dir)

	rl, err := rotatelogs.New(
		filepath.Join(dir, "log%Y%m%d%H%M%S"),
		rotatelogs.WithLinkName("./test.log"),
		rotatelogs.WithRotationTime(10*time.Second),
		rotatelogs.WithRotationCount(3),
		rotatelogs.WithMaxAge(-1),
	)
	if !assert.NoError(t, err, `rotatelogs.New should succeed`) {
		return
	}

	if !assert.NoError(t, rl.Rotate(), "rl.Rotate should succeed") {
		return
	}
	defer rl.Close()
}

func TestRotationGenerationalNames(t *testing.T) {
	dir, err := ioutil.TempDir("", "file-rotatelogs-generational")
	if !assert.NoError(t, err, `creating temporary directory should succeed`) {
		return
	}
	defer os.RemoveAll(dir)

	t.Run("Rotate over unchanged pattern", func(t *testing.T) {
		rl, err := rotatelogs.New(
			filepath.Join(dir, "unchanged-pattern.log"),
		)
		if !assert.NoError(t, err, `rotatelogs.New should succeed`) {
			return
		}

		seen := map[string]struct{}{}
		for i := 0; i < 10; i++ {
			rl.Write([]byte("Hello, World!"))
			if !assert.NoError(t, rl.Rotate(), "rl.Rotate should succeed") {
				return
			}

			// Because every call to Rotate should yield a new log file,
			// and the previous files already exist, the filenames should share
			// the same prefix and have a unique suffix
			fn := filepath.Base(rl.CurrentFileName())
			if !assert.True(t, strings.HasPrefix(fn, "unchanged-pattern.log"), "prefix for all filenames should match") {
				return
			}
			rl.Write([]byte("Hello, World!"))
			suffix := strings.TrimPrefix(fn, "unchanged-pattern.log")
			expectedSuffix := fmt.Sprintf(".%d", i+1)
			if !assert.True(t, suffix == expectedSuffix, "expected suffix %s found %s", expectedSuffix, suffix) {
				return
			}
			assert.FileExists(t, rl.CurrentFileName(), "file does not exist %s", rl.CurrentFileName())
			stat, err := os.Stat(rl.CurrentFileName())
			if err == nil {
				if !assert.True(t, stat.Size() == 13, "file %s size is %d, expected 13", rl.CurrentFileName(), stat.Size()) {
					return
				}
			} else {
				assert.Failf(t, "could not stat file %s", rl.CurrentFileName())
				return
			}

			if _, ok := seen[suffix]; !assert.False(t, ok, `filename suffix %s should be unique`, suffix) {
				return
			}
			seen[suffix] = struct{}{}
		}
		defer rl.Close()
	})
	t.Run("Rotate over pattern change over every second", func(t *testing.T) {
		rl, err := rotatelogs.New(
			filepath.Join(dir, "every-second-pattern-%Y%m%d%H%M%S.log"),
			rotatelogs.WithRotationTime(time.Nanosecond),
		)
		if !assert.NoError(t, err, `rotatelogs.New should succeed`) {
			return
		}

		for i := 0; i < 10; i++ {
			time.Sleep(time.Second)
			rl.Write([]byte("Hello, World!"))
			if !assert.NoError(t, rl.Rotate(), "rl.Rotate should succeed") {
				return
			}

			// because every new Write should yield a new logfile,
			// every rorate should be create a filename ending with a .1
			if !assert.True(t, strings.HasSuffix(rl.CurrentFileName(), ".1"), "log name should end with .1") {
				return
			}
		}
		defer rl.Close()
	})
}

type ClockFunc func() time.Time

func (f ClockFunc) Now() time.Time {
	return f()
}

func TestGHIssue23(t *testing.T) {
	dir, err := ioutil.TempDir("", "file-rotatelogs-generational")
	if !assert.NoError(t, err, `creating temporary directory should succeed`) {
		return
	}
	defer os.RemoveAll(dir)

	for _, locName := range []string{"Asia/Tokyo", "Pacific/Honolulu"} {
		loc, _ := time.LoadLocation(locName)
		tests := []struct {
			Expected string
			Clock    rotatelogs.Clock
		}{
			{
				Expected: filepath.Join(dir, strings.ToLower(strings.Replace(locName, "/", "_", -1)) + ".201806010000.log"),
				Clock: ClockFunc(func() time.Time {
					return time.Date(2018, 6, 1, 3, 18, 0, 0, loc)
				}),
			},
			{
				Expected: filepath.Join(dir, strings.ToLower(strings.Replace(locName, "/", "_", -1)) + ".201712310000.log"),
				Clock: ClockFunc(func() time.Time {
					return time.Date(2017, 12, 31, 23, 52, 0, 0, loc)
				}),
			},
		}
		for _, test := range tests {
			t.Run(fmt.Sprintf("location = %s, time = %s", locName, test.Clock.Now().Format(time.RFC3339)), func(t *testing.T) {
				template := strings.ToLower(strings.Replace(locName, "/", "_", -1)) + ".%Y%m%d%H%M.log"
				rl, err := rotatelogs.New(
					filepath.Join(dir, template),
					rotatelogs.WithClock(test.Clock), // we're not using WithLocation, but it's the same thing
				)
				if !assert.NoError(t, err, "rotatelogs.New should succeed") {
					return
				}

				t.Logf("expected %s", test.Expected)
				rl.Rotate()
				if !assert.Equal(t, test.Expected, rl.CurrentFileName(), "file names should match") {
					return
				}
			})
		}
	}
}

func TestForceNewFile(t *testing.T) {
	dir, err := ioutil.TempDir("", "file-rotatelogs-force-new-file")
	if !assert.NoError(t, err, `creating temporary directory should succeed`) {
		return
	}
	defer os.RemoveAll(dir)

	t.Run("Force a new file", func(t *testing.T) {

		rl, err := rotatelogs.New(
			filepath.Join(dir, "force-new-file.log"),
			rotatelogs.ForceNewFile(),
		)
		if !assert.NoError(t, err, "rotatelogs.New should succeed") {
			return
		}
		rl.Write([]byte("Hello, World!"))
		rl.Close()

		for i := 0; i < 10; i++ {
			baseFn := filepath.Join(dir, "force-new-file.log")
			rl, err := rotatelogs.New(
				baseFn,
				rotatelogs.ForceNewFile(),
			)
			if !assert.NoError(t, err, "rotatelogs.New should succeed") {
				return
			}
			rl.Write([]byte("Hello, World"))
			rl.Write([]byte(fmt.Sprintf("%d", i)))
			rl.Close()

			fn := filepath.Base(rl.CurrentFileName())
			suffix := strings.TrimPrefix(fn, "force-new-file.log")
			expectedSuffix := fmt.Sprintf(".%d", i+1)
			if !assert.True(t, suffix == expectedSuffix, "expected suffix %s found %s", expectedSuffix, suffix) {
				return
			}
			assert.FileExists(t, rl.CurrentFileName(), "file does not exist %s", rl.CurrentFileName())
			content, err := ioutil.ReadFile(rl.CurrentFileName())
			if !assert.NoError(t, err, "ioutil.ReadFile %s should succeed", rl.CurrentFileName()) {
				return
			}
			str := fmt.Sprintf("Hello, World%d", i)
			if !assert.Equal(t, str, string(content), "read %s from file %s, not expected %s", string(content), rl.CurrentFileName(), str) {
				return
			}

			assert.FileExists(t, baseFn, "file does not exist %s", baseFn)
			content, err = ioutil.ReadFile(baseFn)
			if !assert.NoError(t, err, "ioutil.ReadFile should succeed") {
				return
			}
			if !assert.Equal(t, "Hello, World!", string(content), "read %s from file %s, not expected Hello, World!", string(content), baseFn) {
				return
			}
		}

		})

	t.Run("Force a new file with Rotate", func(t *testing.T) {

		baseFn := filepath.Join(dir, "force-new-file-rotate.log")
		rl, err := rotatelogs.New(
			baseFn,
			rotatelogs.ForceNewFile(),
		)
		if !assert.NoError(t, err, "rotatelogs.New should succeed") {
			return
		}
		rl.Write([]byte("Hello, World!"))

		for i := 0; i < 10; i++ {
			if !assert.NoError(t, rl.Rotate(), "rl.Rotate should succeed") {
				return
			}
			rl.Write([]byte("Hello, World"))
	  		rl.Write([]byte(fmt.Sprintf("%d", i)))
			assert.FileExists(t, rl.CurrentFileName(), "file does not exist %s", rl.CurrentFileName())
			content, err := ioutil.ReadFile(rl.CurrentFileName())
			if !assert.NoError(t, err, "ioutil.ReadFile %s should succeed", rl.CurrentFileName()) {
				return
			}
			str := fmt.Sprintf("Hello, World%d", i)
			if !assert.Equal(t, str, string(content), "read %s from file %s, not expected %s", string(content), rl.CurrentFileName(), str) {
				return
			}

			assert.FileExists(t, baseFn, "file does not exist %s", baseFn)
			content, err = ioutil.ReadFile(baseFn)
			if !assert.NoError(t, err, "ioutil.ReadFile should succeed") {
				return
			}
			if !assert.Equal(t, "Hello, World!", string(content), "read %s from file %s, not expected Hello, World!", string(content), baseFn) {
				return
			}
		}
	})
}

