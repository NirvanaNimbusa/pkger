package pkgtest

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/markbates/pkger/here"
	"github.com/markbates/pkger/pkging"
	"github.com/markbates/pkger/pkging/pkgutil"
	"github.com/stretchr/testify/require"
)

type Suite struct {
	Name string
	gen  func() (pkging.Pkger, error)
}

func (s Suite) Make() (pkging.Pkger, error) {
	if s.gen == nil {
		return nil, fmt.Errorf("missing generator function")
	}
	return s.gen()
}

func NewSuite(name string, fn func() (pkging.Pkger, error)) (Suite, error) {
	suite := Suite{
		Name: name,
		gen:  fn,
	}
	return suite, nil
}

func (s Suite) Test(t *testing.T) {
	rv := reflect.ValueOf(s)
	rt := rv.Type()
	if rt.NumMethod() == 0 {
		t.Fatalf("something went wrong wrong with %s", s.Name)
	}
	for i := 0; i < rt.NumMethod(); i++ {
		m := rt.Method(i)
		if !strings.HasPrefix(m.Name, "Test_") {
			continue
		}

		s.sub(t, m)
	}
}

func (s Suite) Run(t *testing.T, name string, fn func(t *testing.T)) {
	t.Helper()
	t.Run(name, func(st *testing.T) {
		fn(st)
	})
}

func (s Suite) sub(t *testing.T, m reflect.Method) {
	name := fmt.Sprintf("%s/%s", s.Name, m.Name)
	s.Run(t, name, func(st *testing.T) {
		m.Func.Call([]reflect.Value{
			reflect.ValueOf(s),
			reflect.ValueOf(st),
		})
	})
}

func (s Suite) Test_Create(t *testing.T) {
	r := require.New(t)

	pkg, err := s.Make()
	r.NoError(err)

	app, err := App()
	r.NoError(err)

	table := []struct {
		in string
	}{
		{in: "/public/index.html"},
		{in: app.Info.ImportPath + ":" + "/public/index.html"},
	}

	for _, tt := range table {
		s.Run(t, tt.in, func(st *testing.T) {
			r := require.New(st)

			pt, err := pkg.Parse(tt.in)
			r.NoError(err)

			r.NoError(pkg.MkdirAll(filepath.Dir(pt.Name), 0755))

			f, err := pkg.Create(pt.Name)
			r.NoError(err)
			r.Equal(pt.Name, f.Name())

			fi, err := f.Stat()
			r.NoError(err)
			r.NoError(f.Close())

			r.Equal(pt.Name, fi.Name())
			r.NotZero(fi.ModTime())
			r.NoError(pkg.RemoveAll(pt.String()))
		})
	}
}

func (s Suite) Test_Create_No_MkdirAll(t *testing.T) {
	r := require.New(t)

	app, err := App()
	r.NoError(err)

	ip := app.Info.ImportPath
	mould := "/easy/listening/file.under"

	table := []struct {
		in string
	}{
		{in: mould},
		{in: ip + ":" + mould},
		{in: filepath.Dir(mould)},
		{in: ip + ":" + filepath.Dir(mould)},
	}

	for _, tt := range table {
		s.Run(t, tt.in, func(st *testing.T) {
			r := require.New(st)

			pkg, err := s.Make()
			r.NoError(err)

			pt, err := pkg.Parse(tt.in)
			r.NoError(err)

			_, err = pkg.Create(pt.Name)
			r.Error(err)
		})
	}
}

func (s Suite) Test_Current(t *testing.T) {
	r := require.New(t)

	pkg, err := s.Make()
	r.NoError(err)

	info, err := pkg.Current()
	r.NoError(err)
	r.NotZero(info)
}

func (s Suite) Test_Info(t *testing.T) {
	r := require.New(t)

	pkg, err := s.Make()
	r.NoError(err)

	cur, err := pkg.Current()
	r.NoError(err)

	info, err := pkg.Info(cur.ImportPath)
	r.NoError(err)
	r.NotZero(info)

}

func (s Suite) Test_MkdirAll(t *testing.T) {
	r := require.New(t)

	app, err := App()
	r.NoError(err)

	ip := app.Info.ImportPath
	mould := "/public/index.html"

	table := []struct {
		in string
	}{
		{in: mould},
		{in: ip + ":" + mould},
		{in: filepath.Dir(mould)},
		{in: ip + ":" + filepath.Dir(mould)},
	}

	for _, tt := range table {
		s.Run(t, tt.in, func(st *testing.T) {
			r := require.New(st)

			pkg, err := s.Make()
			r.NoError(err)

			pt, err := pkg.Parse(tt.in)
			r.NoError(err)

			dir := filepath.Dir(pt.Name)
			r.NoError(pkg.MkdirAll(dir, 0755))

			fi, err := pkg.Stat(dir)
			r.NoError(err)

			if runtime.GOOS == "windows" {
				dir = strings.Replace(dir, "\\", "/", -1)
			}
			r.Equal(dir, fi.Name())
			r.NotZero(fi.ModTime())
			r.NoError(pkg.RemoveAll(pt.String()))
		})
	}
}

func (s Suite) Test_Open_File(t *testing.T) {
	r := require.New(t)

	app, err := App()
	r.NoError(err)

	ip := app.Info.ImportPath
	mould := "/public/index.html"

	table := []struct {
		in string
	}{
		{in: mould},
		{in: ip + ":" + mould},
	}

	for _, tt := range table {
		s.Run(t, tt.in, func(st *testing.T) {
			r := require.New(st)

			pkg, err := s.Make()
			r.NoError(err)

			pt, err := pkg.Parse(tt.in)
			r.NoError(err)

			r.NoError(pkg.RemoveAll(pt.String()))
			r.NoError(pkg.MkdirAll(filepath.Dir(pt.Name), 0755))

			body := "!" + pt.String()

			pkgutil.WriteFile(pkg, tt.in, []byte(body), 0644)

			f, err := pkg.Open(tt.in)
			r.NoError(err)

			r.Equal(pt.Name, f.Path().Name)
			b, err := ioutil.ReadAll(f)
			r.NoError(err)
			r.Equal(body, string(b))

			b, err = pkgutil.ReadFile(pkg, tt.in)
			r.NoError(err)
			r.Equal(body, string(b))

			r.NoError(f.Close())
		})
	}
}

func (s Suite) Test_Parse(t *testing.T) {
	r := require.New(t)

	pkg, err := s.Make()
	r.NoError(err)

	cur, err := pkg.Current()
	ip := cur.ImportPath
	mould := "/public/index.html"

	table := []struct {
		in  string
		exp here.Path
	}{
		{in: mould, exp: here.Path{Pkg: ip, Name: mould}},
		{in: filepath.Join(cur.Dir, mould), exp: here.Path{Pkg: ip, Name: mould}},
		{in: ip + ":" + mould, exp: here.Path{Pkg: ip, Name: mould}},
		{in: ip, exp: here.Path{Pkg: ip, Name: "/"}},
	}

	for _, tt := range table {
		s.Run(t, tt.in, func(st *testing.T) {
			r := require.New(st)

			pt, err := pkg.Parse(tt.in)
			r.NoError(err)
			r.Equal(tt.exp, pt)
		})
	}
}

func (s Suite) Test_Stat_Error(t *testing.T) {
	r := require.New(t)

	pkg, err := s.Make()
	r.NoError(err)

	app, err := App()
	r.NoError(err)

	ip := app.Info.ImportPath

	table := []struct {
		in string
	}{
		{in: "/dontexist"},
		{in: ip},
		{in: ip + ":"},
		{in: ip + ":" + "/dontexist"},
	}

	for _, tt := range table {
		s.Run(t, tt.in, func(st *testing.T) {

			r := require.New(st)

			pt, err := pkg.Parse(tt.in)
			r.NoError(err)

			r.NoError(pkg.RemoveAll(pt.String()))

			_, err = pkg.Stat(tt.in)
			r.Error(err)
		})
	}
}

func (s Suite) Test_Stat_Dir(t *testing.T) {
	r := require.New(t)

	pkg, err := s.Make()
	r.NoError(err)

	app, err := App()
	r.NoError(err)

	ip := app.Info.ImportPath
	dir := app.Paths.Public[1]

	table := []struct {
		in string
	}{
		{in: ip},
		{in: dir},
		{in: ip + ":" + dir},
	}

	for _, tt := range table {
		s.Run(t, tt.in, func(st *testing.T) {

			r := require.New(st)

			pt, err := pkg.Parse(tt.in)
			r.NoError(err)

			r.NoError(pkg.RemoveAll(pt.String()))

			r.NoError(pkg.MkdirAll(pt.Name, 0755))
			info, err := pkg.Stat(tt.in)
			r.NoError(err)
			r.Equal(pt.Name, info.Name())
		})
	}
}

func (s Suite) Test_Stat_File(t *testing.T) {
	r := require.New(t)

	app, err := App()
	r.NoError(err)

	ip := app.Info.ImportPath
	mould := "/public/index.html"

	table := []struct {
		in string
	}{
		{in: mould},
		{in: ip + ":" + mould},
	}

	for _, tt := range table {
		s.Run(t, tt.in, func(st *testing.T) {

			r := require.New(st)

			pkg, err := s.Make()
			r.NoError(err)

			pt, err := pkg.Parse(tt.in)
			r.NoError(err)

			r.NoError(pkg.RemoveAll(pt.String()))
			r.NoError(pkg.MkdirAll(filepath.Dir(pt.Name), 0755))

			f, err := pkg.Create(tt.in)
			r.NoError(err)

			_, err = io.Copy(f, strings.NewReader("!"+pt.String()))
			r.NoError(err)
			r.NoError(f.Close())

			info, err := pkg.Stat(tt.in)
			r.NoError(err)
			r.Equal(pt.Name, info.Name())
		})
	}
}

func (s Suite) Test_Walk(t *testing.T) {
	r := require.New(t)

	pkg, err := s.Make()
	r.NoError(err)

	r.NoError(s.LoadFolder(pkg))

	app, err := App()
	r.NoError(err)

	table := []struct {
		in  string
		exp []string
	}{
		{in: "/", exp: app.Paths.Root},
		{in: "/public", exp: app.Paths.Public},
	}

	for _, tt := range table {
		s.Run(t, tt.in, func(st *testing.T) {

			tdir, err := ioutil.TempDir("", "")
			r.NoError(err)
			defer os.RemoveAll(tdir)
			r.NoError(s.WriteFolder(tdir))

			var goact []string
			err = filepath.Walk(filepath.Join(tdir, tt.in), func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				path = strings.TrimPrefix(path, tdir)

				if path == "" || path == "." {
					path = "/"
				}

				pt, err := pkg.Parse(path)
				if err != nil {
					return err
				}
				goact = append(goact, pt.String())
				return nil
			})
			r.NoError(err)
			r.Equal(tt.exp, goact)

			r := require.New(st)
			var act []string
			err = pkg.Walk(tt.in, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				act = append(act, path)
				return nil
			})
			r.NoError(err)

			r.Equal(tt.exp, act)
		})
	}

}

func (s Suite) Test_Remove(t *testing.T) {
	r := require.New(t)

	pkg, err := s.Make()
	r.NoError(err)

	cur, err := pkg.Current()
	r.NoError(err)

	ip := cur.ImportPath

	table := []struct {
		in string
	}{
		{in: "/public/images/img1.png"},
		{in: ip + ":/public/images/img1.png"},
	}

	for _, tt := range table {
		s.Run(t, tt.in, func(st *testing.T) {
			r := require.New(st)

			pkg, err := s.Make()
			r.NoError(err)
			r.NoError(s.LoadFolder(pkg))

			_, err = pkg.Stat(tt.in)
			r.NoError(err)

			r.NoError(pkg.Remove(tt.in))

			_, err = pkg.Stat(tt.in)
			r.Error(err)

			r.Error(pkg.Remove("unknown"))
		})
	}

}

func (s Suite) Test_RemoveAll(t *testing.T) {
	r := require.New(t)

	pkg, err := s.Make()
	r.NoError(err)

	cur, err := pkg.Current()
	r.NoError(err)

	ip := cur.ImportPath

	table := []struct {
		in string
	}{
		{in: "/public"},
		{in: ip + ":/public"},
	}

	for _, tt := range table {
		s.Run(t, tt.in, func(st *testing.T) {
			r := require.New(st)

			pkg, err := s.Make()
			r.NoError(err)
			r.NoError(s.LoadFolder(pkg))

			_, err = pkg.Stat(tt.in)
			r.NoError(err)

			r.NoError(pkg.RemoveAll(tt.in))

			_, err = pkg.Stat(tt.in)
			r.Error(err)
		})
	}

}
