package errno_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/openGemini/openGemini/lib/errno"
	"pgregory.net/rapid"
)

func TestNewError_ErrnoPreservation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		e := rapid.Uint16().Draw(t, "errno")
		args := rapid.SliceOf(rapid.String()).Draw(t, "args")

		interfaces := make([]interface{}, len(args))
		for i, a := range args {
			interfaces[i] = a
		}

		err := errno.NewError(errno.Errno(e), interfaces...)
		if err.Errno() != errno.Errno(e) {
			t.Fatalf("Errno mismatch: got %d, want %d", err.Errno(), e)
		}
	})
}

func TestEqual_Reflexivity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		e := rapid.Uint16().Draw(t, "errno")
		err := errno.NewError(errno.Errno(e))
		if !errno.Equal(err, errno.Errno(e)) {
			t.Fatalf("Equal(err, err.Errno()) = false for errno %d", e)
		}
	})
}

func TestEqual_NilAlwaysFalse(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		errnos := rapid.SliceOf(rapid.Uint16()).Draw(t, "errnos")
		enos := make([]errno.Errno, len(errnos))
		for i, e := range errnos {
			enos[i] = errno.Errno(e)
		}
		if errno.Equal(nil, enos...) {
			t.Fatalf("Equal(nil, ...) should be false")
		}
	})
}

func TestEqual_NonTypeError(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		errnos := rapid.SliceOf(rapid.Uint16()).Draw(t, "errnos")
		enos := make([]errno.Errno, len(errnos))
		for i, e := range errnos {
			enos[i] = errno.Errno(e)
		}
		plainErr := fmt.Errorf("plain error")
		if errno.Equal(plainErr, enos...) {
			t.Fatalf("Equal(plainError, ...) should be false")
		}
	})
}

func TestSetModule_Roundtrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		e := rapid.Uint16().Draw(t, "errno")
		mod := rapid.Int8Range(0, 27).Draw(t, "module")

		err := errno.NewError(errno.Errno(e))
		err.SetModule(errno.Module(mod))
		if err.Module() != errno.Module(mod) {
			t.Fatalf("SetModule(%d) then Module() = %d", mod, err.Module())
		}
	})
}

func TestSetErrno_Roundtrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		e1 := rapid.Uint16().Draw(t, "errno1")
		e2 := rapid.Uint16().Draw(t, "errno2")

		err := errno.NewError(errno.Errno(e1))
		err.SetErrno(errno.Errno(e2))
		if err.Errno() != errno.Errno(e2) {
			t.Fatalf("SetErrno(%d) then Errno() = %d", e2, err.Errno())
		}
	})
}

func TestLevelSetters(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		e := rapid.Uint16().Draw(t, "errno")
		err := errno.NewError(errno.Errno(e))

		err.SetToNotice()
		if err.Level() != errno.LevelNotice {
			t.Fatalf("SetToNotice: level = %d", err.Level())
		}

		err.SetToWarn()
		if err.Level() != errno.LevelWarn {
			t.Fatalf("SetToWarn: level = %d", err.Level())
		}

		err.SetToFatal()
		if err.Level() != errno.LevelFatal {
			t.Fatalf("SetToFatal: level = %d", err.Level())
		}
	})
}

func TestNewBuiltIn_PreservesPtrError(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		e := rapid.Uint16().Draw(t, "errno")
		origErr := errno.NewError(errno.Errno(e))
		result := errno.NewBuiltIn(origErr, errno.ModuleQueryEngine)
		if result != origErr {
			t.Fatalf("NewBuiltIn should return same pointer for *Error input")
		}
	})
}

func TestNewThirdParty_PreservesPtrError(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		e := rapid.Uint16().Draw(t, "errno")
		origErr := errno.NewError(errno.Errno(e))
		result := errno.NewThirdParty(origErr, errno.ModuleQueryEngine)
		if result != origErr {
			t.Fatalf("NewThirdParty should return same pointer for *Error input")
		}
	})
}

func TestNewBuiltIn_WrapsPlainError(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		msg := rapid.String().Draw(t, "msg")
		plainErr := errors.New(msg)
		result := errno.NewBuiltIn(plainErr, errno.ModuleQueryEngine)
		if result.Errno() != errno.BuiltInError {
			t.Fatalf("NewBuiltIn(plainErr).Errno() = %d; want BuiltInError=%d", result.Errno(), errno.BuiltInError)
		}
		if result.Error() != msg {
			t.Fatalf("NewBuiltIn(plainErr).Error() = %q; want %q", result.Error(), msg)
		}
	})
}

func TestNewThirdParty_WrapsPlainError(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		msg := rapid.String().Draw(t, "msg")
		plainErr := errors.New(msg)
		result := errno.NewThirdParty(plainErr, errno.ModuleQueryEngine)
		if result.Errno() != errno.ThirdPartyError {
			t.Fatalf("NewThirdParty(plainErr).Errno() = %d; want ThirdPartyError=%d", result.Errno(), errno.ThirdPartyError)
		}
		if result.Error() != msg {
			t.Fatalf("NewThirdParty(plainErr).Error() = %q; want %q", result.Error(), msg)
		}
	})
}

func TestNewRemote_ErrnoPreservation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		msg := rapid.String().Draw(t, "msg")
		e := rapid.Uint16().Draw(t, "errno")
		err := errno.NewRemote(msg, errno.Errno(e))
		if err.Errno() != errno.Errno(e) {
			t.Fatalf("NewRemote errno mismatch: got %d, want %d", err.Errno(), e)
		}
	})
}

func TestLogStack_OnlyFatal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		l := rapid.Uint8Range(0, 2).Draw(t, "level")
		shouldLog := errno.Level(l) >= errno.LevelFatal
		if errno.Level(l).LogStack() != shouldLog {
			t.Fatalf("Level(%d).LogStack() = %v; want %v", l, errno.Level(l).LogStack(), shouldLog)
		}
	})
}

func TestErrs_FirstErrorWins(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 10).Draw(t, "n")
		errs := make([]error, n)
		for i := range errs {
			if rapid.Bool().Draw(t, "isErr") {
				errs[i] = fmt.Errorf("error_%d", i)
			}
		}

		e := errno.NewErrs()
		e.Init(n, nil)
		for _, err := range errs {
			e.Dispatch(err)
		}
		result := e.Err()

		firstIdx := -1
		for i, err := range errs {
			if err != nil {
				firstIdx = i
				break
			}
		}

		if firstIdx == -1 {
			if result != nil {
				t.Fatalf("expected nil, got %v", result)
			}
		} else {
			if result == nil {
				t.Fatalf("expected error, got nil")
			}
			if result.Error() != errs[firstIdx].Error() {
				t.Fatalf("expected %q, got %q", errs[firstIdx].Error(), result.Error())
			}
		}
	})
}
