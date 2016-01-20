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
		local: "public/index.html", size: 3390, modtime: 1453277129,
		compressed: `
H4sIAAAJbogA/5xXW2/buBJ+z6+YssWB3ESSc+lpj2O7yEmDkxz0hm0W2EXRB0aiLSYUqZKUHbfI/9r3
/WM71C1U5ABt/dCI5Mw3M99cyE6fvPlwevnnxzPIbC7mO1P3BwSVyxlhksx3AKYZo6n7wM+cWQpJRrVh
dkZKuwhfEf8os7YI2deSr2bkj/D3k/BU5QW1/EowAomSlknUuzibsXTJWk3LrWDzN+8/wcVHeMMsS6zS
8D/+LVfTuD70TEiasxlZcbYulLYe6pqnNpulbMUTFlaLPeCSW05FaBIq2Gw/GqPNnRpMcHkDmWaLGYlj
Y6OC58tIMhsnqYwFvzLxlVLWWE2L+DAax4nxdqKcywh3CGgmZsTYjWAmY8y2Mf0sOruleYEY8XWZXymr
lQwl1VqtBxuPma3tPglDOL989/YFmIznQGUKvzFTKJlG1wYuzl6BKQtHHKhFI8gEy5FAUwnnLOUUvpZM
c2YgDOcd7Ge+AGERAv7zpd7FfZNoXlgwOpkRl3sziWNlTJTTWww0SlReB+uq6gV6tMJgX2K43Rq9IvNp
XOP8NKzuQov3I0dju+HSM0SePvnMZMoXX1xc9U5FIdhNgTVl2a11aSatH1cq3cD3ZgFQ0DTlchlaVUzg
YFzcHg+OMFNW5RM48k7vdpqP+DmclgbPq5qlXDINz+PmMOr2mjx7hnOql1xOYAy0tOrY27+tC30CL8c9
i49hzrEmtwAfojKMBw6nmJ8Flin6OYfC01sgbmj4NzYBk1Mh7l1yHIZU8CWiar7M7FbQFBuZC+MhptwU
gm4mIJVkA52njQLyeztwv87H/tGAcUy+y26V6mlcTzH36dJa53+KzkAiqDEz0nHVpt8VfVuQnpzDuRdy
h6VozyRd4XxahQUXGF1RChFWJHjS1Wxo5Wli+Qrn4JQ2s+IpmZ+rnE1jiqUr+AO1ntzJlSrtjwieusCS
LaLTuBTeKjts3aqSmJeWpegOE0JBM5wvmUGc7LBr1BiJaRddU/X56uYX6Y4bAZ7OCDZsKfr8ALxV1HVT
FEWer5Wlna2rwoviPvDWAYFoYVNAZO6woVk5SjrEYqv3baW65sFyu2EWHfOT36sgEYpluH/QT3d2VN1t
hukV0wY2qtSOS5wYGLyBjBooDUuR16OeXkuRV/tuoHmMD4jwnKn71ve0mP8rUcXmGE7MDfz3+u+/tIRz
Kg2TOMz2DyHouIvj9XqNw7YomI6kxftCiUhpjDtXmuGlulA6x1tdScfgqOJu6E/v25/oj1yI1+7W2cQH
zcdjM/wHkEqJHWoS9NVdC94y/HXM/nV9/fAt8GugmVpSGR/Wf38EB+1ahg8Fapl5IOsL32f9WUCeNj02
itytGxDXxHV3kdH94F5RDdq9AGZ4meHz7kLaIHhHbRa5XZUHo+cH4/Y32tsfjyKrPlmNSMHhvz2gZ9GS
2f9/+vA+wKgJ7Naou0C2FBTyqGTxuvp39prswfe7vd4kWJQycXUWpNTSkTf6e25X8aHjpB5WzqiTj84v
Lz/C7kDH/dBW1Yj3nVk1oWvMTh3PjncG2vgMqryJzvB8m0v4azzaRZemV3oenCADHFNIedr1/xr7vjPl
sBxHI3I8wLsb+jDMar0a9bXvutXICwSVadQbiqMoETy5CTq22Z7tR/YsYJHF25ZZtMdTFvQs+SmnBY9X
+3G+4cXP59PF5Y+7Jrj7kueFiTRz/Rx8B4IrMqkIhLvR6CFzCOa9NRDLZGodPEpRRVK33RLm9Vf1iqjf
DtO4/t/SPwEAAP//KMmb1j4NAAA=
`,
	},

	"/public/js/templates.js": {
		local: "public/js/templates.js", size: 1372, modtime: 1386147544,
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
