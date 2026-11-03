package config

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"
)

// envRef matches ${NAME} references in config string values. $${ is an escape
// for a literal ${.
var envRef = regexp.MustCompile(`\$?\$\{([^}]*)\}`)

// expandEnv replaces ${NAME} in every string value of cfg with the environment
// variable NAME, so a config can be committed with its secrets held in the
// environment (e.g. a private .env file). It runs after TOML decoding, so a
// value can never change the file's syntax. A reference to an unset variable
// is an error naming the variable and the key that used it.
func expandEnv(cfg *Config) error {
	return expandValue(reflect.ValueOf(cfg).Elem(), "")
}

func expandValue(v reflect.Value, path string) error {
	switch v.Kind() {
	case reflect.String:
		s, err := expandString(v.String(), path)
		if err != nil {
			return err
		}
		v.SetString(s)
	case reflect.Pointer:
		if !v.IsNil() {
			return expandValue(v.Elem(), path)
		}
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			name, _, _ := strings.Cut(f.Tag.Get("toml"), ",")
			if name == "" || name == "-" {
				continue
			}
			if err := expandValue(v.Field(i), joinPath(path, name)); err != nil {
				return err
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if err := expandValue(v.Index(i), fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			// Map values aren't addressable: expand a copy and store it back.
			elem := reflect.New(iter.Value().Type()).Elem()
			elem.Set(iter.Value())
			if err := expandValue(elem, joinPath(path, iter.Key().String())); err != nil {
				return err
			}
			v.SetMapIndex(iter.Key(), elem)
		}
	}
	return nil
}

func expandString(s, path string) (string, error) {
	if !strings.Contains(s, "${") {
		return s, nil
	}
	var err error
	out := envRef.ReplaceAllStringFunc(s, func(m string) string {
		if strings.HasPrefix(m, "$$") {
			return m[1:]
		}
		name := m[2 : len(m)-1]
		val, ok := os.LookupEnv(name)
		if !ok && err == nil {
			err = fmt.Errorf("config: %s uses ${%s}, but %s is not set in the environment", path, name, name)
		}
		return val
	})
	return out, err
}

func joinPath(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}
