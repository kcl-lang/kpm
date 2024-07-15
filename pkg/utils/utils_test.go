package utils

import (
	"archive/tar"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testDataDir = "test_data"

func getTestDir(subDir string) string {
	pwd, _ := os.Getwd()
	testDir := filepath.Join(pwd, testDataDir)
	testDir = filepath.Join(testDir, subDir)

	return testDir
}

func TestParseRepoNameFromGitUrl(t *testing.T) {
	assert.Equal(t, ParseRepoNameFromGitUrl("test"), "test", "ParseRepoNameFromGitUrl failed.")
	assert.Equal(t, ParseRepoNameFromGitUrl("test.git"), "test", "ParseRepoNameFromGitUrl failed.")
	assert.Equal(t, ParseRepoNameFromGitUrl("test.git.git"), "test.git", "ParseRepoNameFromGitUrl failed.")
	assert.Equal(t, ParseRepoNameFromGitUrl("https://test.git"), "test", "ParseRepoNameFromGitUrl failed.")
	assert.Equal(t, ParseRepoNameFromGitUrl("https://test.git.git"), "test.git", "ParseRepoNameFromGitUrl failed.")
	assert.Equal(t, ParseRepoNameFromGitUrl("httfsdafps://test.git.git"), "test.git", "ParseRepoNameFromGitUrl failed.")
}

type TestPath struct {
	FilePath string
}

func (tp *TestPath) TestStore() error {
	return StoreToFile(tp.FilePath, "test")
}

func TestCreateFileIfNotExist(t *testing.T) {
	test_path := getTestDir("test_exist.txt")
	isExist, _ := Exists(test_path)
	assert.Equal(t, isExist, false)

	tp := TestPath{
		FilePath: test_path,
	}
	err := CreateFileIfNotExist(tp.FilePath, tp.TestStore)
	assert.Equal(t, err, nil)

	isExist, _ = Exists(test_path)
	assert.Equal(t, isExist, true)

	_ = os.Remove(test_path)
	isExist, _ = Exists(test_path)
	assert.Equal(t, isExist, false)
}

func TestHashDir(t *testing.T) {
	test_path := filepath.Join(getTestDir("test_hash"), "test_hash.txt")
	tp := TestPath{
		FilePath: test_path,
	}

	_ = CreateFileIfNotExist(tp.FilePath, tp.TestStore)
	res, err := HashDir(filepath.Dir(tp.FilePath))
	assert.Equal(t, err, nil)
	assert.Equal(t, res, "n4bQgYhMfWWaL+qgxVrQFaO/TxsrC4Is0V1sFbDwCgg=")
}

func TestTarDir(t *testing.T) {
	testDir := getTestDir("test_tar")
	tarPath := filepath.Join(testDir, "test.tar")

	_, err := os.Stat(tarPath)
	if !os.IsNotExist(err) {
		os.Remove(tarPath)
	}

	testSrcDir := filepath.Join(testDir, "test_src")

	getTarFileNames := func(filePath string) ([]string, error) {
		file, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		reader := tar.NewReader(file)
		filePaths := []string{}

		for {
			header, err := reader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}

			if header.Typeflag != tar.TypeReg {
				continue
			}

			fullPath := path.Join(header.Name)
			fullPath = path.Join(filePath, fullPath)
			fullPath = strings.Replace(fullPath, "test.tar", "test_src", 1)

			filePaths = append(filePaths, fullPath)
		}

		return filePaths, nil
	}

	getNewPattern := func(ex string) string {
		return testSrcDir + "/" + ex
	}

	err = TarDir(testSrcDir, tarPath, []string{}, []string{})
	assert.Equal(t, err, nil)
	_, err = os.Stat(tarPath)
	assert.Equal(t, err, nil)
	os.Remove(tarPath)

	_ = TarDir(testSrcDir, tarPath, []string{}, []string{"*.mod"})
	fileNames, _ := getTarFileNames(tarPath)
	for _, fileName := range fileNames {
		flag, _ := filepath.Match(getNewPattern("*.mod"), fileName)
		assert.Equal(t, flag, false)
	}
	_, err = os.Stat(tarPath)
	assert.Equal(t, err, nil)
	os.Remove(tarPath)

	_ = TarDir(testSrcDir, tarPath, []string{"*/*.lock", "*.mod"}, []string{})
	fileNames, _ = getTarFileNames(tarPath)
	for _, fileName := range fileNames {
		flag, _ := filepath.Match(getNewPattern("*/*.lock"), fileName)
		assert.Equal(t, flag, true)
	}
	_, err = os.Stat(tarPath)
	assert.Equal(t, err, nil)
	os.Remove(tarPath)
}

func TestUnTarDir(t *testing.T) {
	testDir := getTestDir("test_un_tar")
	tarPath := filepath.Join(testDir, "test.tar")
	testSrc := filepath.Join(testDir, "test_src")

	err := UnTarDir(tarPath, testSrc)
	assert.Equal(t, err, nil)

	_, err = os.Stat(testSrc)
	assert.Equal(t, err, nil)
	_ = os.RemoveAll(testSrc)
}

func TestDefaultKpmHome(t *testing.T) {
	homeDir, _ := os.UserHomeDir()

	filePath := filepath.Join(homeDir, ".kpm")

	kpmHome, err := CreateSubdirInUserHome(".kpm")
	assert.Equal(t, err, nil)
	assert.Equal(t, kpmHome, filePath)
	assert.Equal(t, DirExists(kpmHome), true)
}

func TestJoinPath(t *testing.T) {
	assert.Equal(t, JoinPath("base", "elem"), "base/elem")
	assert.Equal(t, JoinPath("base/", "elem"), "base/elem")
	assert.Equal(t, JoinPath("base", "/elem"), "base/elem")
	assert.Equal(t, JoinPath("", "/elem"), "/elem")
	assert.Equal(t, JoinPath("", "elem"), "/elem")
	assert.Equal(t, JoinPath("base/", ""), "base/")
	assert.Equal(t, JoinPath("base", ""), "base/")
}

func TestIsUrl(t *testing.T) {
	assert.Equal(t, IsURL("invalid url"), false)
	assert.Equal(t, IsURL("https://url/xxx"), true)
	assert.Equal(t, IsURL("https://url"), true)
	assert.Equal(t, IsURL("https://"), false)
}

func TestIsGitRepoUrl(t *testing.T) {
	assert.Equal(t, IsGitRepoUrl("invalid url"), false)
	assert.Equal(t, IsGitRepoUrl("ftp://github.com/user/project.git"), false)
	assert.Equal(t, IsGitRepoUrl("file:///path/to/repo.git/"), false)
	assert.Equal(t, IsGitRepoUrl("file://~/path/to/repo.git/"), false)
	assert.Equal(t, IsGitRepoUrl("path/to/repo.git/"), false)
	assert.Equal(t, IsGitRepoUrl("~/path/to/repo.git"), false)
	assert.Equal(t, IsGitRepoUrl("rsync://host.xz/path/to/repo.git/"), false)
	assert.Equal(t, IsGitRepoUrl("host.xz:path/to/repo.git"), false)
	assert.Equal(t, IsGitRepoUrl("user@host.xz:path/to/repo.git"), false)
	assert.Equal(t, IsGitRepoUrl("C:\\path\\to\\repo.git"), false)
	assert.Equal(t, IsGitRepoUrl("/path/to/repo.git"), false)
	assert.Equal(t, IsGitRepoUrl("./path/to/repo.git"), false)
	assert.Equal(t, IsGitRepoUrl("oci://host.xz/path/to/repo.git/"), false)
	assert.Equal(t, IsGitRepoUrl("https://github.com/user/project"), true)
	assert.Equal(t, IsGitRepoUrl("git@github.com:user/project.git"), true)
	assert.Equal(t, IsGitRepoUrl("https://github.com/user/project.git"), true)
	assert.Equal(t, IsGitRepoUrl("https://github.com/user/project.git"), true)
	assert.Equal(t, IsGitRepoUrl("git@192.168.101.127:user/project.git"), true)
	assert.Equal(t, IsGitRepoUrl("https://192.168.101.127/user/project.git"), true)
	assert.Equal(t, IsGitRepoUrl("http://192.168.101.127/user/project.git"), true)
	assert.Equal(t, IsGitRepoUrl("ssh://user@host.xz:port/path/to/repo.git/"), true)
	assert.Equal(t, IsGitRepoUrl("ssh://user@host.xz/path/to/repo.git/"), true)
	assert.Equal(t, IsGitRepoUrl("ssh://host.xz:port/path/to/repo.git/"), true)
	assert.Equal(t, IsGitRepoUrl("ssh://host.xz/path/to/repo.git/"), true)
	assert.Equal(t, IsGitRepoUrl("ssh://user@host.xz/path/to/repo.git/"), true)
	assert.Equal(t, IsGitRepoUrl("ssh://user@host.xz/~user/path/to/repo.git/"), true)
	assert.Equal(t, IsGitRepoUrl("ssh://host.xz/~user/path/to/repo.git/"), true)
	assert.Equal(t, IsGitRepoUrl("ssh://user@host.xz/~/path/to/repo.git"), true)
	assert.Equal(t, IsGitRepoUrl("git://host.xz/path/to/repo.git/"), true)
	assert.Equal(t, IsGitRepoUrl("http://host.xz/path/to/repo.git/"), true)
	assert.Equal(t, IsGitRepoUrl("https://host.xz/path/to/repo.git/"), true)
}

func TestIsRef(t *testing.T) {
	assert.Equal(t, IsRef("invalid ref"), false)
	assert.Equal(t, IsRef("ghcr.io/xxx/xxx"), true)
	assert.Equal(t, IsRef("ghcr.io/xxx"), true)
	assert.Equal(t, IsRef("ghcr.io/xxx:0.0.1"), true)
	assert.Equal(t, IsRef("ghcr.io/"), false)
}

func TestIsTar(t *testing.T) {
	assert.Equal(t, IsTar("invalid tar"), false)
	assert.Equal(t, IsTar("xxx.tar"), true)
}

func TestIsKfile(t *testing.T) {
	assert.Equal(t, IsKfile("invalid kfile"), false)
	assert.Equal(t, IsKfile("xxx.k"), true)
}

func TestAbsTarPath(t *testing.T) {
	pkgPath := getTestDir("test_check_tar_path")
	expectAbsTarPath, _ := filepath.Abs(filepath.Join(pkgPath, "test.tar"))

	abs, err := AbsTarPath(filepath.Join(pkgPath, "test.tar"))
	assert.Equal(t, err, nil)
	assert.Equal(t, abs, expectAbsTarPath)

	abs, err = AbsTarPath(filepath.Join(pkgPath, "no_exist.tar"))
	assert.NotEqual(t, err, nil)
	assert.Equal(t, abs, "")

	abs, err = AbsTarPath(filepath.Join(pkgPath, "invalid_tar"))
	assert.NotEqual(t, err, nil)
	assert.Equal(t, abs, "")
}

func TestIsSymlinkExist(t *testing.T) {
	testPath := filepath.Join(getTestDir("test_link"), "is_link_exist")

	link_target_not_exist := filepath.Join(testPath, "link_target_not_exist")

	linkExist, targetExist, err := IsSymlinkValidAndExists(link_target_not_exist)
	assert.Equal(t, err, nil)
	assert.Equal(t, linkExist, true)
	assert.Equal(t, targetExist, false)

	linkExist, targetExist, err = IsSymlinkValidAndExists("invalid_link")
	assert.Equal(t, err, nil)
	assert.Equal(t, linkExist, false)
	assert.Equal(t, targetExist, false)

	filename := filepath.Join(testPath, "test.txt")
	validLink := filepath.Join(testPath, "valid_link")
	err = CreateSymlink(filename, validLink)
	assert.Equal(t, err, nil)

	linkExist, targetExist, err = IsSymlinkValidAndExists(validLink)
	assert.Equal(t, err, nil)
	assert.Equal(t, linkExist, true)
	assert.Equal(t, targetExist, true)

	anotherValidLink := filepath.Join(testPath, "another_valid_link")
	err = CreateSymlink(filename, anotherValidLink)
	assert.Equal(t, err, nil)

	linkExist, targetExist, err = IsSymlinkValidAndExists(anotherValidLink)
	assert.Equal(t, err, nil)
	assert.Equal(t, linkExist, true)
	assert.Equal(t, targetExist, true)
	// Defer the removal of the symlink
	defer func() {
		err := os.Remove(anotherValidLink)
		assert.Equal(t, err, nil)
		err = os.Remove(validLink)
		assert.Equal(t, err, nil)
	}()
}

func TestIsModRelativePath(t *testing.T) {
	assert.Equal(t, IsModRelativePath("${KCL_MOD}/aaa"), true)
	assert.Equal(t, IsModRelativePath("${helloworld:KCL_MOD}/aaa"), true)
	assert.Equal(t, IsModRelativePath("xxx/xxx/xxx"), false)
}
