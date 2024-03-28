package mvs

import (
	"fmt"

	"golang.org/x/mod/module"
)

type ReqsMap map[module.Version][]module.Version

func (r ReqsMap) Max(_, v1, v2 string) string {
	if v1 == "none" || v2 == "" {
		return v2
	}
	if v2 == "none" || v1 == "" {
		return v1
	}
	if v1 < v2 {
		return v2
	}
	return v1
}

func (r ReqsMap) Upgrade(m module.Version) (module.Version, error) {
	panic("unimplemented")
}

func (r ReqsMap) Previous(m module.Version) (module.Version, error) {
	panic("unimplemented")
}

func (r ReqsMap) Required(m module.Version) ([]module.Version, error) {
	rr, ok := r[m]
	if !ok {
		return nil, fmt.Errorf("missing module: %v", m)
	}
	return rr, nil
}
