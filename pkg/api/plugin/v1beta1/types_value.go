package v1beta1

import (
	"bytes"
	"encoding/gob"
)

type Values struct {
	Raw    []byte                 `json:"-"`
	Object map[string]interface{} `json:"-"`
}

func init() {
	// https://pkg.go.dev/encoding/gob#Register
	gob.Register(map[string]interface{}{})
}

func (v *Values) DeepCopy() *Values {
	if v == nil {
		return nil
	}
	out := Values{}
	if v.Raw != nil {
		out.Raw = make([]byte, len(v.Raw))
		copy(out.Raw, v.Raw)
	}
	if v.Object != nil {
		buf := new(bytes.Buffer)
		gob.NewEncoder(buf).Encode(v.Object)
		gob.NewDecoder(buf).Decode(&out.Object)
	}
	return &out
}
