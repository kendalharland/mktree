package main

import (
	"errors"
	"flag"
	"fmt"
	"strings"
)

type repeatedFlag struct {
	value  func() flag.Value
	values []flag.Value
}

func (f *repeatedFlag) Set(value string) error {
	v := f.value()
	if err := v.Set(value); err != nil {
		return err
	}
	f.values = append(f.values, v)
	return nil
}

func (f *repeatedFlag) String() string {
	ss := make([]string, 0, len(f.values))
	for _, v := range f.values {
		ss = append(ss, v.String())
	}
	return "[" + strings.Join(ss, ",") + "]"
}

func (f *repeatedFlag) Get() interface{} {
	return f.values
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
