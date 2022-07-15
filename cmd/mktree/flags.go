package main

import (
	"errors"
	"fmt"
	"strings"
)

type variablesFlag struct {
	vars []*keyValueFlag
}

func (f *variablesFlag) Set(value string) error {
	kv := &keyValueFlag{}
	if err := kv.Set(value); err != nil {
		return err
	}
	f.vars = append(f.vars, kv)
	return nil
}

func (f *variablesFlag) String() string {
	ss := make([]string, 0, len(f.vars))
	for _, v := range f.vars {
		ss = append(ss, v.String())
	}
	return "[" + strings.Join(ss, ",") + "]"
}

func (f *variablesFlag) Get() interface{} {
	vars := map[string]string{}
	for _, kv := range f.vars {
		vars[kv.K] = kv.V
	}
	return vars
}

type keyValueFlag struct {
	K, V string
}

func (f *keyValueFlag) Set(value string) error {
	kv := strings.SplitN(value, "=", 2)
	if len(kv) < 2 || len(kv[0]) == 0 {
		return errors.New("key-value pair must not be empty")
	}
	if len(kv[1]) == 0 {
		return errors.New("missing value in key-value pair")
	}
	f.K = kv[0]
	f.V = kv[1]
	return nil
}

func (f *keyValueFlag) String() string {
	return fmt.Sprintf("%s=%v", f.K, f.V)
}
