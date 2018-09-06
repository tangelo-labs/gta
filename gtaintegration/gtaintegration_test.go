// +build integration

package gtaintegration

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"go/build"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"do/tools/build/gta"

	"github.com/go-test/deep"
	"github.com/pkg/errors"
)

const (
	repoRoot = "testdata/gtaintegration"
)

func TestMain(m *testing.M) {
	if err := testMain(m); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func TestPackageRemoval(t *testing.T) {
	ctx := context.Background()
	if _, err := runGit(ctx, ".", "checkout", "-b", t.Name(), "master"); err != nil {
		t.Fatal(err)
	}

	// delete all go files from gofilesdeleted
	deleteGoFilesDir, err := os.Open(filepath.Clean("src/gtaintegration/gofilesdeleted"))

	if err != nil {
		t.Fatal(err)
	}
	defer deleteGoFilesDir.Close()

	names, err := deleteGoFilesDir.Readdirnames(-1)
	if err != nil {
		t.Fatal(err)
	}

	for _, fn := range names {
		if filepath.Ext(fn) == ".go" {
			err := os.Remove(filepath.Join(filepath.Clean("src/gtaintegration/gofilesdeleted"), fn))
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	// fully delete deleted
	if err := os.RemoveAll(filepath.Clean("src/gtaintegration/deleted")); err != nil {
		t.Fatal(err)
	}

	// move some files to a different directory
	if _, err := runGit(ctx, ".", "mv", "src/gtaintegration/movedfrom", "src/gtaintegration/movedto"); err != nil {
		t.Fatal(err)
	}

	if testing.Verbose() {
		out, err := runGit(ctx, ".", "status", "--short")
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("\n%s", out)
	}

	if _, err := runGit(ctx, ".", "commit", "-a", "-m", "delete some stuff"); err != nil {
		t.Fatal(err)
	}

	if testing.Verbose() {
		out, err := runGit(ctx, ".", "diff", "origin/master...HEAD", "--name-only", "--no-renames")
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("\n%s", out)
	}
	options := []gta.Option{
		gta.SetDiffer(gta.NewDiffer(false)),
		gta.SetPrefixes("gtaintegration"),
	}

	gt, err := gta.New(options...)
	if err != nil {
		t.Fatalf("can't prepare gta: %v", err)
	}

	want := &gta.Packages{
		Dependencies: map[string][]*build.Package{
			"gtaintegration/deleted": []*build.Package{
				&build.Package{
					ImportPath: "gtaintegration/deletedclient",
				},
			},
			"gtaintegration/gofilesdeleted": []*build.Package{
				&build.Package{
					ImportPath: "gtaintegration/gofilesdeletedclient",
				},
			},
			"gtaintegration/movedfrom": []*build.Package{
				&build.Package{
					ImportPath: "gtaintegration/movedfromclient",
				},
			},
		},
		Changes: []*build.Package{
			&build.Package{
				ImportPath: "gtaintegration/deleted",
			},
			&build.Package{
				ImportPath: "gtaintegration/gofilesdeleted",
			},
			&build.Package{
				ImportPath: "gtaintegration/movedfrom",
			},
			&build.Package{
				ImportPath: "gtaintegration/movedto",
			},
		},
		AllChanges: []*build.Package{
			&build.Package{
				ImportPath: "gtaintegration/deleted",
			},
			&build.Package{
				ImportPath: "gtaintegration/deletedclient",
			},
			&build.Package{
				ImportPath: "gtaintegration/gofilesdeleted",
			},
			&build.Package{
				ImportPath: "gtaintegration/gofilesdeletedclient",
			},
			&build.Package{
				ImportPath: "gtaintegration/movedfrom",
			},
			&build.Package{
				ImportPath: "gtaintegration/movedfromclient",
			},
			&build.Package{
				ImportPath: "gtaintegration/movedto",
			},
		},
	}

	got, err := gt.ChangedPackages()
	if err != nil {
		t.Fatalf("err = %q; want nil", err)
	}

	if diff := deep.Equal(mapFromPackages(t, got), mapFromPackages(t, want)); diff != nil {
		t.Error(diff)
	}
}

func TestPackageRemoval_AllGoFilesDeleted(t *testing.T) {
	ctx := context.Background()
	if _, err := runGit(ctx, ".", "checkout", "-b", t.Name(), "master"); err != nil {
		t.Fatal(err)
	}

	// delete all go files from gofilesdeleted
	deleteGoFilesDir, err := os.Open(filepath.Clean("src/gtaintegration/gofilesdeleted"))

	if err != nil {
		t.Fatal(err)
	}
	defer deleteGoFilesDir.Close()

	names, err := deleteGoFilesDir.Readdirnames(-1)
	if err != nil {
		t.Fatal(err)
	}

	for _, fn := range names {
		if filepath.Ext(fn) == ".go" {
			err := os.Remove(filepath.Join(filepath.Clean("src/gtaintegration/gofilesdeleted"), fn))
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	if testing.Verbose() {
		out, err := runGit(ctx, ".", "status", "--short")
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("\n%s", out)
	}

	if _, err := runGit(ctx, ".", "commit", "-a", "-m", "delete some stuff"); err != nil {
		t.Fatal(err)
	}

	if testing.Verbose() {
		out, err := runGit(ctx, ".", "diff", "origin/master...HEAD", "--name-only", "--no-renames")
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("\n%s", out)
	}
	options := []gta.Option{
		gta.SetDiffer(gta.NewDiffer(false)),
		gta.SetPrefixes("gtaintegration"),
	}

	gt, err := gta.New(options...)
	if err != nil {
		t.Fatalf("can't prepare gta: %v", err)
	}

	want := &gta.Packages{
		Dependencies: map[string][]*build.Package{
			"gtaintegration/gofilesdeleted": []*build.Package{
				&build.Package{
					ImportPath: "gtaintegration/gofilesdeletedclient",
				},
			},
		},
		Changes: []*build.Package{
			&build.Package{
				ImportPath: "gtaintegration/gofilesdeleted",
			},
		},
		AllChanges: []*build.Package{
			&build.Package{
				ImportPath: "gtaintegration/gofilesdeleted",
			},
			&build.Package{
				ImportPath: "gtaintegration/gofilesdeletedclient",
			},
		},
	}

	got, err := gt.ChangedPackages()
	if err != nil {
		t.Fatalf("err = %q; want nil", err)
	}

	if diff := deep.Equal(mapFromPackages(t, got), mapFromPackages(t, want)); diff != nil {
		t.Error(diff)
	}
}

func TestPackageRemoval_RemoveDirectory(t *testing.T) {
	ctx := context.Background()
	if _, err := runGit(ctx, ".", "checkout", "-b", t.Name(), "master"); err != nil {
		t.Fatal(err)
	}

	// fully delete deleted
	if err := os.RemoveAll(filepath.Clean("src/gtaintegration/deleted")); err != nil {
		t.Fatal(err)
	}

	if testing.Verbose() {
		out, err := runGit(ctx, ".", "status", "--short")
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("\n%s", out)
	}

	if _, err := runGit(ctx, ".", "commit", "-a", "-m", "delete some stuff"); err != nil {
		t.Fatal(err)
	}

	if testing.Verbose() {
		out, err := runGit(ctx, ".", "diff", "origin/master...HEAD", "--name-only", "--no-renames")
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("\n%s", out)
	}
	options := []gta.Option{
		gta.SetDiffer(gta.NewDiffer(false)),
		gta.SetPrefixes("gtaintegration"),
	}

	gt, err := gta.New(options...)
	if err != nil {
		t.Fatalf("can't prepare gta: %v", err)
	}

	want := &gta.Packages{
		Dependencies: map[string][]*build.Package{
			"gtaintegration/deleted": []*build.Package{
				&build.Package{
					ImportPath: "gtaintegration/deletedclient",
				},
			},
		},
		Changes: []*build.Package{
			&build.Package{
				ImportPath: "gtaintegration/deleted",
			},
		},
		AllChanges: []*build.Package{
			&build.Package{
				ImportPath: "gtaintegration/deleted",
			},
			&build.Package{
				ImportPath: "gtaintegration/deletedclient",
			},
		},
	}

	got, err := gt.ChangedPackages()
	if err != nil {
		t.Fatalf("err = %q; want nil", err)
	}

	if diff := deep.Equal(mapFromPackages(t, got), mapFromPackages(t, want)); diff != nil {
		t.Error(diff)
	}
}

func TestPackageRemoval_MovePackage(t *testing.T) {
	ctx := context.Background()
	if _, err := runGit(ctx, ".", "checkout", "-b", t.Name(), "master"); err != nil {
		t.Fatal(err)
	}

	// move some files to a different directory
	if _, err := runGit(ctx, ".", "mv", "src/gtaintegration/movedfrom", "src/gtaintegration/movedto"); err != nil {
		t.Fatal(err)
	}

	if testing.Verbose() {
		out, err := runGit(ctx, ".", "status", "--short")
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("\n%s", out)
	}

	if _, err := runGit(ctx, ".", "commit", "-a", "-m", "delete some stuff"); err != nil {
		t.Fatal(err)
	}

	if testing.Verbose() {
		out, err := runGit(ctx, ".", "diff", "origin/master...HEAD", "--name-only", "--no-renames")
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("\n%s", out)
	}
	options := []gta.Option{
		gta.SetDiffer(gta.NewDiffer(false)),
		gta.SetPrefixes("gtaintegration"),
	}

	gt, err := gta.New(options...)
	if err != nil {
		t.Fatalf("can't prepare gta: %v", err)
	}

	want := &gta.Packages{
		Dependencies: map[string][]*build.Package{
			"gtaintegration/movedfrom": []*build.Package{
				&build.Package{
					ImportPath: "gtaintegration/movedfromclient",
				},
			},
		},
		Changes: []*build.Package{
			&build.Package{
				ImportPath: "gtaintegration/movedfrom",
			},
			&build.Package{
				ImportPath: "gtaintegration/movedto",
			},
		},
		AllChanges: []*build.Package{
			&build.Package{
				ImportPath: "gtaintegration/movedfrom",
			},
			&build.Package{
				ImportPath: "gtaintegration/movedfromclient",
			},
			&build.Package{
				ImportPath: "gtaintegration/movedto",
			},
		},
	}

	got, err := gt.ChangedPackages()
	if err != nil {
		t.Fatalf("err = %q; want nil", err)
	}

	if diff := deep.Equal(mapFromPackages(t, got), mapFromPackages(t, want)); diff != nil {
		t.Error(diff)
	}
}

func TestNonPackageRemoval(t *testing.T) {
	ctx := context.Background()
	if _, err := runGit(ctx, ".", "checkout", "-b", t.Name(), "master"); err != nil {
		t.Fatal(err)
	}

	// fully delete nogodeleted
	if err := os.RemoveAll(filepath.Clean("src/gtaintegration/nogodeleted")); err != nil {
		t.Fatal(err)
	}

	if testing.Verbose() {
		out, err := runGit(ctx, ".", "status", "--short")
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("\n%s", out)
	}

	if _, err := runGit(ctx, ".", "commit", "-a", "-m", "delete some stuff"); err != nil {
		t.Fatal(err)
	}

	if testing.Verbose() {
		out, err := runGit(ctx, ".", "diff", "origin/master...HEAD", "--name-only", "--no-renames")
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("\n%s", out)
	}
	options := []gta.Option{
		gta.SetDiffer(gta.NewDiffer(false)),
		gta.SetPrefixes("gtaintegration"),
	}

	gt, err := gta.New(options...)
	if err != nil {
		t.Fatalf("can't prepare gta: %v", err)
	}

	want := &gta.Packages{
		Dependencies: map[string][]*build.Package{},
		Changes:      []*build.Package{},
		AllChanges:   []*build.Package{},
	}

	got, err := gt.ChangedPackages()
	if err != nil {
		t.Fatalf("err = %q; want nil", err)
	}

	if diff := deep.Equal(mapFromPackages(t, got), mapFromPackages(t, want)); diff != nil {
		t.Error(diff)
	}
}
func testMain(m *testing.M) error {
	flag.Parse()

	// copy all of testdata to a temporary directory, because we're going to
	// mutate it.
	wd := prepareTemp()

	// change to the temporary directory
	err := os.Chdir(wd)
	if err != nil {
		return err
	}
	defer os.RemoveAll(wd)

	// configure the repository.
	err = createRepo(repoRoot)
	if err != nil {
		return err
	}

	// change to the repository for the remainder of the tests
	err = os.Chdir(repoRoot)
	if err != nil {
		return err
	}

	wd, err = os.Getwd()
	if err != nil {
		return err
	}

	// This is super jank, but the default packager uses build.Default, so just set GOPATH in it...
	build.Default.GOPATH = wd

	if i := m.Run(); i != 0 {
		return errors.New("failed")
	}

	return nil
}

func createRepo(path string) error {
	if testing.Verbose() {
		log.Println("creating repo in " + path)
	}

	ctx := context.Background()
	// git init
	if _, err := runGit(ctx, path, "init"); err != nil {
		return err
	}

	if _, err := runGit(ctx, path, "add", "src"); err != nil {
		return err
	}

	if _, err := runGit(ctx, path, "commit", "-m", "initial commit"); err != nil {
		return err
	}

	// create an origin/master branch from master to simulate a remote.
	if _, err := runGit(ctx, path, "branch", "origin/master"); err != nil {
		return err
	}

	return nil
}

func runGit(ctx context.Context, wd string, args ...string) (string, error) {
	args = append([]string{"-c", "user.email=gtaintegration@example.com", "-c", "user.name=gtaintegration test"}, args...)

	cmd := exec.CommandContext(ctx, "git", args...)
	wd = abs(wd)

	cmd.Dir = wd

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.New(string(out))
	}
	return string(out), nil
}

func abs(path string) string {
	path, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}

	return path
}

func prepareTemp() string {
	d, err := ioutil.TempDir("", "gta-integration-tests")
	if err != nil {
		panic(err)
	}

	err = filepath.Walk("testdata", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			dir := filepath.Join(d, path)
			err := os.Mkdir(dir, info.Mode())
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("could not create directory (%s)", dir))
			}
			return nil
		}

		src, err := os.Open(path)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("could not open file (%s)", path))
		}
		defer src.Close()

		dst, err := os.Create(filepath.Join(d, path))
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("could not open file (%s)", path))
		}
		defer dst.Close()

		_, err = io.Copy(dst, src)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("could not copy file (%s)", path))
		}

		err = os.Chmod(dst.Name(), info.Mode())
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("could not set file permission (%s)", dst.Name()))
		}
		return nil
	})

	if err != nil {
		os.RemoveAll(d)
		panic(err)
	}

	return d
}

// setenv sets an environment variable, name, to value and returns a function
// to restore the environment variable to its former value.
func setEnv(t *testing.T, name, value string) func() {
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

func mapFromPackages(t *testing.T, pkg *gta.Packages) map[string]interface{} {
	t.Helper()

	m := make(map[string]interface{})

	b, err := json.Marshal(pkg)
	if err != nil {
		t.Fatal(err)
	}

	err = json.Unmarshal(b, &m)
	if err != nil {
		t.Fatal(err)
	}

	return m
}
