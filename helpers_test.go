package gta

import (
	"os"
	"strings"
	"testing"
)

// Setenv sets an environment variable, name, to value and returns a function
// to restore the environment variable to its former value.
func Setenv(t *testing.T, name, value string) func() {
	t.Helper()

	orig, ok := os.LookupEnv(name)

	if err := os.Setenv(name, value); err != nil {
		t.Fatal(err)
	}

	return func() {
		if !ok {
			if err := os.Unsetenv(name); err != nil {
				t.Fatal(err)
			}

			return
		}

		if err := os.Setenv(name, orig); err != nil {
			t.Fatal(err)
		}
	}
}

func AllSetenv(t *testing.T, env []string) func() {
	t.Helper()

	var fns []func()
	for _, v := range env {
		parts := strings.SplitN(v, "=", 2)
		fn := Setenv(t, parts[0], parts[1])
		fns = append(fns, fn)
	}

	return func() {
		for _, f := range fns {
			f()
		}
	}
}

// chdir changes to dir and returns a function that will restore the current
// working directory to its previous value.
func chdir(t *testing.T, dir string) func() {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	err = os.Chdir(dir)
	if err != nil {
		t.Fatal(err)
	}

	return func() {
		err = os.Chdir(wd)
		if err != nil {
			t.Fatal(err)
		}
	}
}
