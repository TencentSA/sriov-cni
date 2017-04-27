package config

import (
	"fmt"
	"strconv"
)

type UnmarshallableInt int

func (i *UnmarshallableInt) UnmarshalText(data []byte) error {
	s := string(data)
	v, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("Int unmarshal error: %v", err)
	}

	*i = UnmarshallableInt(v)
	return nil
}
