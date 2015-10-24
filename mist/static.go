package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sync"
	"time"
)

type _escLocalFS struct{}

var _escLocal _escLocalFS

type _escStaticFS struct{}

var _escStatic _escStaticFS

type _escDir struct {
	fs   http.FileSystem
	name string
}

type _escFile struct {
	compressed string
	size       int64
	modtime    int64
	local      string
	isDir      bool

	data []byte
	once sync.Once
	name string
}

func (_escLocalFS) Open(name string) (http.File, error) {
	f, present := _escData[path.Clean(name)]
	if !present {
		return nil, os.ErrNotExist
	}
	return os.Open(f.local)
}

func (_escStaticFS) prepare(name string) (*_escFile, error) {
	f, present := _escData[path.Clean(name)]
	if !present {
		return nil, os.ErrNotExist
	}
	var err error
	f.once.Do(func() {
		f.name = path.Base(name)
		if f.size == 0 {
			return
		}
		var gr *gzip.Reader
		b64 := base64.NewDecoder(base64.StdEncoding, bytes.NewBufferString(f.compressed))
		gr, err = gzip.NewReader(b64)
		if err != nil {
			return
		}
		f.data, err = ioutil.ReadAll(gr)
	})
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (fs _escStaticFS) Open(name string) (http.File, error) {
	f, err := fs.prepare(name)
	if err != nil {
		return nil, err
	}
	return f.File()
}

func (dir _escDir) Open(name string) (http.File, error) {
	return dir.fs.Open(dir.name + name)
}

func (f *_escFile) File() (http.File, error) {
	type httpFile struct {
		*bytes.Reader
		*_escFile
	}
	return &httpFile{
		Reader:   bytes.NewReader(f.data),
		_escFile: f,
	}, nil
}

func (f *_escFile) Close() error {
	return nil
}

func (f *_escFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}

func (f *_escFile) Stat() (os.FileInfo, error) {
	return f, nil
}

func (f *_escFile) Name() string {
	return f.name
}

func (f *_escFile) Size() int64 {
	return f.size
}

func (f *_escFile) Mode() os.FileMode {
	return 0
}

func (f *_escFile) ModTime() time.Time {
	return time.Unix(f.modtime, 0)
}

func (f *_escFile) IsDir() bool {
	return f.isDir
}

func (f *_escFile) Sys() interface{} {
	return f
}

// FS returns a http.Filesystem for the embedded assets. If useLocal is true,
// the filesystem's contents are instead used.
func FS(useLocal bool) http.FileSystem {
	if useLocal {
		return _escLocal
	}
	return _escStatic
}

// Dir returns a http.Filesystem for the embedded assets on a given prefix dir.
// If useLocal is true, the filesystem's contents are instead used.
func Dir(useLocal bool, name string) http.FileSystem {
	if useLocal {
		return _escDir{fs: _escLocal, name: name}
	}
	return _escDir{fs: _escStatic, name: name}
}

// FSByte returns the named file from the embedded assets. If useLocal is
// true, the filesystem's contents are instead used.
func FSByte(useLocal bool, name string) ([]byte, error) {
	if useLocal {
		f, err := _escLocal.Open(name)
		if err != nil {
			return nil, err
		}
		return ioutil.ReadAll(f)
	}
	f, err := _escStatic.prepare(name)
	if err != nil {
		return nil, err
	}
	return f.data, nil
}

// FSMustByte is the same as FSByte, but panics if name is not present.
func FSMustByte(useLocal bool, name string) []byte {
	b, err := FSByte(useLocal, name)
	if err != nil {
		panic(err)
	}
	return b
}

// FSString is the string version of FSByte.
func FSString(useLocal bool, name string) (string, error) {
	b, err := FSByte(useLocal, name)
	return string(b), err
}

// FSMustString is the string version of FSMustByte.
func FSMustString(useLocal bool, name string) string {
	return string(FSMustByte(useLocal, name))
}

var _escData = map[string]*_escFile{

	"/public/index.html": {
		local: "public/index.html", size: 3398, modtime: 1444974526,
		compressed: `
H4sIAAAJbogA/6RX3W7buBK+z1NM1eLAbiLJblK0x7Fd5KTBSQ/6h9MssIuiF7Q0tphQpEpSdtwi77X3
+2I71J+lyAHabS5qkZz5ZuabH7LTR68/nF/98fECEpuK+cHU/YBgcjXzUHrzA4Bpgix2H/SZomUQJUwb
tDMvt0v/pdc+SqzNfPya8/XM+93/7cw/V2nGLF8I9CBS0qIkvTcXM4xXWGtabgXOX7//BG8+wmu0GFml
4b/8W6qmYXnYMiFZijNvzXGTKW1bqBse22QW45pH6BeLI+CSW86EbyImcDYORmTzoAQTXN5AonE585zX
kzA0Nsh4ugok2jCKZSj4woQLpayxmmXhcTAKI9PaCVIuA9rxQKOYecZuBZoE0daB/SMTeMvSjIDC6zxd
KKuV9CXTWm16Gw/ZLo0/8n24vHr39jmYhKfAZAz/R5MpGQfXBt5cvASTZ45CUMtKEAWmRKUphFOMOYOv
OWqOBnx/3sB+5ksQliDg31/KXdo3keaZBaOjMlhD0SpjgpTdUqBBpNIyWFdfz8mjNQX7gsJt1uSVN5+G
Jc5Pw+omtHAcOBrrDZejPvL00WeUMV9+cXGVOwWFYLcZVZfFW+ty7dV+LFS8he/VAiBjcczlyrcqm8Cz
UXZ72juiTFmVTuCkdXp3UH2ET+E8N3ReVC/jEjU8DavDoNmr8twynDK94nICI2C5Vaet/duy5CfwYtSx
+BDmnApzD/AxKcOo53BM+VlSmZKfc8haekvC9Q3/hhMwKRNi55Lj0GeCrwhV81Vi94LG1NJcmBZizE0m
2HYCUkns6TyuFIjf2577ZT7GJz3GKfkuu0Wqp2E5z9ynS2uZ/yk5A5Fgxsy8hqs6/a7o64JsyTmcnZA7
zEV9JtmaJtXaz7ig6LJcCL8goSVdDIhankWWr2kiTlk1MB5780uV4jRkVLqC31PryJ0tVG5/RPDcBRbt
EZ2GuWitkuParSKJaW4xJndQCAXVmL5CQzjJcdOoIRFTL5qm6vLVzC+vOa4EeDzzqGFz0eUH4K1irpuC
IGj5Wlg62LvKWlHsAq8dEITmVwXkzR02VCtHSYOY7fW+rlTXPFRuN2jJsXbyOxUkfLHyx8+66U5OilvO
oF6jNrBVuXZc0sSg4A0kzEBuMCZeTzp6NUWt2ncDrcV4j4iWM2Xftj3N5v+KVLY9hTNzA/+5/utPLeGS
SYOShtn4GAYNd9W1tdlsaOJmGepAWro0lAiUpuBTpZHu2KXSKV3ySjoahwWBfac63/fH+kNX47W7f7bh
s+rjoWn+o3C5pIY1EXntbonW0v9F4O4Vfn3/kfALyIlaMRkel78/AkbGLdILglk092TbwrtyeDLwHlfN
NwzcdTzwXHeXbecNdxN9zTRo9zSYwTtmk8B9q3QwhKcwHo1GO8EnwQrt/z59eD+oQ/PgsFQ9BG9PJRFj
Smavin9nr7wj+H531JkDy1xGrsAGMbNs2Br8Hd+KIMg7rxxVzqiTDy6vrj7CYU/H/ZGtog13fVm0oGvL
Rp3OTg962vQIKrwJLuh8n0v0V3l0SC5NF3o+OCMGOOWJ8bjp/g11fWPKYTmOht5pD++u70M/deVq2NW+
a1bDViCkzILOSBwGkeDRzaBhG49sN7InAwws3bVoyR6PcdCx1Mp7yDIersdhuuXZz+fTxdUedlVwu7rm
mQk0uvYdfAePVt6kIBDuhsP7zBFY66VBWCZRm8GDFBUkNds1Ya0mKt4Q5cthGpb/a/o7AAD//wmNHJNG
DQAA
`,
	},

	"/public/js/templates.js": {
		local: "public/js/templates.js", size: 1372, modtime: 1445655689,
		compressed: `
H4sIAAAJbogA/6xUUW/TMBB+Zr/i6qdYmJBSRMXaTEJl0nip0Mob5cFNnM1ScCz7UoSy/HfOCSkTsGxT
+9DW1/vy3efvU04XEE0mE1TfbSlReQ576eBQQgpNuzg71F+Ztp59o7+N+gFX1Y008ZffzajJqlydQ1Gb
DHVlIMqEFZpD03OmeKv9AuNdpFN9d8cY7wq2RLkrFWSl9D7dsr7qvl95dNqqfMsuBvDWMHgJ+vCoe6Dz
Yom3Fx/XG9got1du+ZrKEeSqqg26n4/BrtUNXWwE8GGzfozjklSdVM4Iilr/MUgXEcaePkXUxSlCTlPe
/SRiOk/E2+SdYE0Dbcs4bzB2vstySLYrMDRGM1hiPnQw3vfz+jg+fe6HJpz/UZo/m2a1OgnN9WloKPun
8dz3/0r6y9z8ncGb+VzMZu+fkcG/ogLtWuGRVwssR9scSJ5qcssXWWwrG9GRjJqMOjUlrxLBeoP64ZBV
pbfS0CaZ0d54YMjY63FfwQAL2yggncLaGSBBJbWhFVY61LL057QnBfh6F05AofHF2a8AAAD//wYxBPlc
BQAA
`,
	},

	"/public/js/widget.js": {
		local: "public/js/widget.js", size: 70, modtime: 1386487443,
		compressed: `
H4sIAAAJbogA/0orzUsuyczP09BUqOZSAAKNvNRyBc/cxPRUDU1NveKiZFslff3y8nK93MSCgtQivbyS
goL8/By9/KJ0/bz8vFQla65aLkAAAAD//4QVA35GAAAA
`,
	},

	"/": {
		isDir: true,
		local: "/",
	},

	"/public": {
		isDir: true,
		local: "/public",
	},

	"/public/js": {
		isDir: true,
		local: "/public/js",
	},
}
