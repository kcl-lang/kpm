package kpm

import (
	"testing"
)

func TestInit(t *testing.T) {

	// kpm init test_example
	err := CLI([]string{"kpm", "init", "test_example"}...)
	if err != nil {
		println(err.Error())
		return
	}
}
func TestAdd(t *testing.T) {
	// kpm add -git github.com/orangebees/konfig@v0.0.1
	err := CLI([]string{"kpm", "add", "-git", "github.com/orangebees/konfig@v0.0.1"}...)
	if err != nil {
		println(err.Error())
		return
	}
}
func TestDel(t *testing.T) {
	// kpm del konfig
	err := CLI([]string{"kpm", "del", "konfig"}...)
	if err != nil {
		println(err.Error())
		return
	}
}
func TestAddLatest(t *testing.T) {
	// kpm add -git github.com/orangebees/konfig@v0.0.1
	err := CLI([]string{"kpm", "add", "-git", "github.com/orangebees/konfig"}...)
	if err != nil {
		println(err.Error())
		return
	}
}
func TestDownload(t *testing.T) {

	// kpm download
	err := CLI([]string{"kpm", "download"}...)
	if err != nil {
		println(err.Error())
		return
	}
}

func TestStore(t *testing.T) {
	// kpm store add -git github.com/orangebees/konfig@v0.0.1
	err := CLI([]string{"kpm", "store", "add", "-git", "github.com/orangebees/konfig@v0.0.1"}...)
	if err != nil {
		println(err.Error())
		return
	}

}
