package envload

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func (e *sysenv) Clearenv() {
	os.Clearenv()
}

func (e *sysenv) Setenv(k, v string) {
	os.Setenv(k, v)
}

func SystemEnvironment() Environment {
	return &sysenv{}
}

func New(environ ...string) *Loader {
	if len(environ) == 0 {
		environ = os.Environ()
	}

	var envdir string
	original := make([]iterItem, 0, len(environ))
	for _, v := range environ {
		i := strings.IndexByte(v, '=')
		if i <= 0 || i >= len(v)-1 {
			continue
		}
		original = append(original, iterItem{
			key:   v[:i],
			value: v[i+1:],
		})
		if v[:i] == "ENVDIR" {
			envdir = v[i+1:]
		}
	}

	return &Loader{
		original: original,
		envdir:   envdir,
	}
}

func (l *Loader) Restore(options ...Option) error {
	var ctx context.Context = context.Background()
	var e Environment = SystemEnvironment()
	var loadEnvdir bool
	for _, o := range options {
		switch o.Name() {
		case ContextKey:
			ctx = o.Value().(context.Context)
		case EnvironmentKey:
			e = o.Value().(Environment)
		case LoadEnvdirKey:
			loadEnvdir = o.Value().(bool)
		}
	}
	return l.Apply(ctx, e, WithLoadEnvdir(loadEnvdir))
}

func (l *Loader) Apply(octx context.Context, e Environment, options ...Option) error {
	ctx, cancel := context.WithCancel(octx)
	defer cancel()

	e.Clearenv()
	iter := l.Iterator(ctx, options...)
	for iter.Next() {
		k, v := iter.KV()
		e.Setenv(k, v)
	}

	return nil
}

func (l *Loader) Environ(octx context.Context, options ...Option) []string {
	ctx, cancel := context.WithCancel(octx)
	defer cancel()

	var environ []string
	it := l.Iterator(ctx, options...)
	for it.Next() {
		k, v := it.KV()
		environ = append(environ, k+`=`+v)
	}
	return environ
}

func (l *Loader) Iterator(ctx context.Context, options ...Option) *Iterator {
	loadEnvdir := true
	for _, o := range options {
		switch o.Name() {
		case LoadEnvdirKey:
			loadEnvdir = o.Value().(bool)
		}
	}

	ch := make(chan *iterItem)

	var ex chan *iterItem
	if loadEnvdir && l.envdir != "" {
		if fi, err := os.Stat(l.envdir); err == nil && fi.IsDir() {
			ex = make(chan *iterItem)
		}
	}

	go func(m []iterItem, ch, ex chan *iterItem) {
		defer close(ch)
		for _, it := range m {
			select {
			case <-ctx.Done():
				return
			case ch <- &iterItem{key: it.key, value: it.value}:
			}
		}

		if ex == nil {
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			case it, ok := <-ex:
				if !ok {
					return
				}
				select {
				case <-ctx.Done():
					return
				case ch <- it:
				}
			}
		}
	}(l.original, ch, ex)

	// meanwhile, load from envdir, if available
	if ex != nil {
		go func() {
			defer close(ex)
			filepath.Walk(l.envdir, func(path string, fi os.FileInfo, err error) error {
				// Ignore errors
				if err != nil {
					return nil
				}

				// Do not recurse into directories
				if fi.IsDir() && l.envdir != path {
					return filepath.SkipDir
				}

				buf, err := ioutil.ReadFile(path)
				if err != nil {
					return nil
				}

				ex <- &iterItem{
					key:   filepath.Base(path),
					value: string(bytes.TrimSpace(buf)),
				}
				return nil
			})
		}()
	}

	return &Iterator{
		ch: ch,
	}
}

func (iter *Iterator) Next() bool {
	iter.nextK = ""
	iter.nextV = ""
	pair, ok := <-iter.ch
	if !ok {
		return false
	}
	iter.nextK = pair.key
	iter.nextV = pair.value
	return true
}

func (iter *Iterator) KV() (string, string) {
	return iter.nextK, iter.nextV
}
