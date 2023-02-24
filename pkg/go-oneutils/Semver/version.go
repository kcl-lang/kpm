package Semver

import (
	"errors"
	"github.com/KusionStack/kpm/pkg/go-oneutils/Convert"
	"strconv"
)

type VersionString string
type Version struct {
	Major           int    `json:"major"`
	Minor           int    `json:"minor"`
	Patch           int    `json:"patch"`
	PreRelease      int    `json:"pre_release"`
	PreReleaseCount int    `json:"pre_release_count"`
	BuildMetadata   string `json:"build_metadata"`
}

const (
	Alpha = iota
	Beta
	RC
	Release
)

var PreReleaseNames = []string{"alpha", "beta", "rc", "release"}

// 语义化版本号string
func (v *Version) String() string {
	vs := make([]byte, 64)
	vs = vs[:0]
	vs = append(vs, strconv.Itoa(v.Major)...)
	vs = append(vs, '.')
	vs = append(vs, strconv.Itoa(v.Minor)...)
	vs = append(vs, '.')
	vs = append(vs, strconv.Itoa(v.Patch)...)
	if v.PreRelease != Release {
		vs = append(vs, '-')
		vs = append(vs, PreReleaseNames[v.PreRelease]...)
		if v.PreReleaseCount != 0 {
			vs = append(vs, '.')
			vs = append(vs, strconv.Itoa(v.PreReleaseCount)...)
		}
	}
	if v.BuildMetadata != "" {
		vs = append(vs, '+')
		vs = append(vs, v.BuildMetadata...)
	}
	return string(vs)
}

// TagString 语义化版本标签
func (v *Version) TagString() string {
	vs := make([]byte, 64)
	vs = vs[:0]
	vs = append(vs, 'v')
	vs = append(vs, strconv.Itoa(v.Major)...)
	vs = append(vs, '.')
	vs = append(vs, strconv.Itoa(v.Minor)...)
	vs = append(vs, '.')
	vs = append(vs, strconv.Itoa(v.Patch)...)
	if v.PreRelease != Release {
		vs = append(vs, '-')
		vs = append(vs, PreReleaseNames[v.PreRelease]...)
		if v.PreReleaseCount != 0 {
			vs = append(vs, '.')
			vs = append(vs, strconv.Itoa(v.PreReleaseCount)...)
		}
	}
	if v.BuildMetadata != "" {
		vs = append(vs, '+')
		vs = append(vs, v.BuildMetadata...)
	}
	return string(vs)
}
func NewFromString(str string) (*Version, error) {
	v := Version{}
	tmp := make([]byte, 8)
	tmp = tmp[:0]
	status := 0
	v.PreRelease = 3

	for i := 0; i < len(str); i++ {
		s := str[i]
		switch status {
		case 0:
			if s == 'v' || s == 'V' {
				continue
			} else {
				if !isNum(s) {

					return nil, errors.New("faulty data")
				}
			}
			status++
			fallthrough
		case 1:
			if isNum(s) {
				tmp = append(tmp, s)
			} else {
				if s == '.' {
					atoi, err := strconv.Atoi(Convert.B2S(tmp))
					if err != nil {
						return nil, err
					}
					v.Major = atoi
					tmp = tmp[:0]
					status++
				} else {
					return nil, errors.New("faulty data")
				}
			}
		case 2:
			if isNum(s) {
				tmp = append(tmp, s)
			} else {
				if s == '.' {
					atoi, err := strconv.Atoi(Convert.B2S(tmp))
					if err != nil {
						return nil, err
					}
					v.Minor = atoi
					tmp = tmp[:0]
					status++
				} else {
					return nil, errors.New("faulty data")
				}
			}
		case 3:
			if isNum(s) {
				tmp = append(tmp, s)
			} else {
				if s == '-' {
					atoi, err := strconv.Atoi(Convert.B2S(tmp))
					if err != nil {
						return nil, err
					}
					v.Patch = atoi
					tmp = tmp[:0]
					status++
				} else if s == '+' {
					atoi, err := strconv.Atoi(Convert.B2S(tmp))
					if err != nil {
						return nil, err
					}
					v.Patch = atoi
					tmp = tmp[:0]
					status += 3
				} else {
					return nil, errors.New("faulty data")
				}
			}
		case 4:
			//处理先行版字段
			if s != '.' && s != '+' {
				tmp = append(tmp, s)
			} else {
				//println(Convert.B2S(tmp))
				for k := 0; k < len(tmp); k++ {
					tembyte := tmp[k]
					if tembyte >= 'A' && tembyte <= 'Z' {
						tmp[k] += 32
					}
				}
				for j := 0; j < len(PreReleaseNames); j++ {
					if PreReleaseNames[j] == Convert.B2S(tmp) {
						v.PreRelease = j
						break
					} else if j == len(PreReleaseNames)-1 {
						return nil, errors.New("faulty data")
					}
				}
				tmp = tmp[:0]
				status++
				if s == '+' {
					status++
				}
			}
		case 5:
			//处理先行版版本号
			if isNum(s) {
				tmp = append(tmp, s)
			} else {
				if s == '+' {
					atoi, err := strconv.Atoi(Convert.B2S(tmp))
					if err != nil {
						return nil, err
					}
					v.PreReleaseCount = atoi
					tmp = tmp[:0]
					status++
				} else {
					return nil, errors.New("faulty data")
				}
			}
		case 6:
			//编译元信息
			tmp = append(tmp, s)
		}
	}

	switch status {
	case 3:
		atoi, err := strconv.Atoi(Convert.B2S(tmp))
		if err != nil {
			return nil, err
		}
		v.Patch = atoi
	case 4:
		for k := 0; k < len(tmp); k++ {
			tembyte := tmp[k]
			if tembyte >= 'A' && tembyte <= 'Z' {
				tmp[k] += 32
			}
		}
		for j := 0; j < len(PreReleaseNames); j++ {
			if PreReleaseNames[j] == Convert.B2S(tmp) {
				v.PreRelease = j
				break
			} else if j == len(PreReleaseNames)-1 {
				return nil, errors.New("faulty data")
			}
		}
	case 5:
		atoi, err := strconv.Atoi(Convert.B2S(tmp))
		if err != nil {
			return nil, err
		}
		v.PreReleaseCount = atoi
	case 6:
		v.BuildMetadata = Convert.B2S(tmp)
	}

	return &v, nil
}
func (v *Version) Cmp(nv *Version) int {
	if v.Major != nv.Major {
		if v.Major > nv.Major {
			return 1
		} else {
			return -1
		}
	}
	if v.Minor != nv.Minor {
		if v.Minor > nv.Minor {
			return 1
		} else {
			return -1
		}
	}
	if v.Patch != nv.Patch {
		if v.Patch > nv.Patch {
			return 1
		} else {
			return -1
		}
	}
	if v.PreRelease != nv.PreRelease {
		if v.PreRelease > nv.PreRelease {
			return 1
		} else {
			return -1
		}
	}
	if v.PreReleaseCount != nv.PreReleaseCount {
		if v.PreReleaseCount > nv.PreReleaseCount {
			return 1
		} else {
			return -1
		}
	}
	return 0
}

func isNum(s uint8) bool {
	if s >= 48 && s <= 57 {
		return true
	}
	return false
}
