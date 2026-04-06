package utils

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
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

func TestGetUsernamePasswordFromStdin(t *testing.T) {
	runWithStdin(t, "secret\n", func() {
		username, password, err := GetUsernamePassword("test-user", "", true)
		assert.NoError(t, err)
		assert.Equal(t, "test-user", username)
		assert.Equal(t, "secret", password)
	})
}

func TestGetUsernamePasswordFromEmptyStdin(t *testing.T) {
	runWithStdin(t, "\n", func() {
		_, _, err := GetUsernamePassword("test-user", "", true)
		assert.EqualError(t, err, "password required")
	})
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
	assert.Greater(t, len(fileNames), 0, "expected tar to have greater than 0 files")
	for _, fileName := range fileNames {
		flag, _ := filepath.Match(getNewPattern("*.mod"), fileName)
		assert.Equal(t, flag, false, "expected *.mod to be excluded")
	}
	_, err = os.Stat(tarPath)
	assert.Equal(t, err, nil)
	os.Remove(tarPath)

	_ = TarDir(testSrcDir, tarPath, []string{"*/*.lock", "*.mod"}, []string{})
	fileNames, _ = getTarFileNames(tarPath)
	assert.Greater(t, len(fileNames), 0, "expected tar to have greater than 0 files")
	for _, fileName := range fileNames {
		matchedLockPattern, _ := filepath.Match(getNewPattern("*/*.lock"), fileName)
		matchedModPattern, _ := filepath.Match(getNewPattern("*.mod"), fileName)
		assert.True(t, matchedLockPattern || matchedModPattern, fmt.Sprintf("expected \"%s\" to match one of */*.lock or *.mod", fileName))
	}
	_, err = os.Stat(tarPath)
	assert.Equal(t, err, nil)
	os.Remove(tarPath)
}

func runWithStdin(t *testing.T, input string, fn func()) {
	t.Helper()

	oldStdin := os.Stdin
	reader, writer, err := os.Pipe()
	assert.NoError(t, err)

	_, err = writer.WriteString(input)
	assert.NoError(t, err)
	assert.NoError(t, writer.Close())

	os.Stdin = reader
	defer func() {
		os.Stdin = oldStdin
		assert.NoError(t, reader.Close())
	}()

	fn()
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
	assert.Equal(t, JoinPath("", "elem"), "elem")
	assert.Equal(t, JoinPath("", "elem"), "elem")
	assert.Equal(t, JoinPath("base/", ""), "base")
	assert.Equal(t, JoinPath("base", ""), "base")
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

func TestIsTgz(t *testing.T) {
	assert.Equal(t, IsTgz("invalid tgz"), false)
	assert.Equal(t, IsTgz("xxx.tgz"), true)
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

func TestAbsPkgArchivePath(t *testing.T) {
	testDir := t.TempDir()
	tarPath := filepath.Join(testDir, "test.tar")
	tgzPath := filepath.Join(testDir, "test.tgz")
	assert.NoError(t, os.WriteFile(tarPath, []byte("tar"), 0644))
	assert.NoError(t, os.WriteFile(tgzPath, []byte("tgz"), 0644))

	abs, err := AbsPkgArchivePath(tarPath)
	assert.NoError(t, err)
	assert.Equal(t, abs, tarPath)

	abs, err = AbsPkgArchivePath(tgzPath)
	assert.NoError(t, err)
	assert.Equal(t, abs, tgzPath)

	abs, err = AbsPkgArchivePath(filepath.Join(testDir, "invalid.zip"))
	assert.Error(t, err)
	assert.Equal(t, abs, "")
}

func gzipTarFile(t *testing.T, srcPath, dstPath string) {
	t.Helper()

	src, err := os.Open(srcPath)
	assert.NoError(t, err)
	defer src.Close()

	dst, err := os.Create(dstPath)
	assert.NoError(t, err)
	defer dst.Close()

	gzipWriter := gzip.NewWriter(dst)
	_, err = io.Copy(gzipWriter, src)
	assert.NoError(t, err)
	assert.NoError(t, gzipWriter.Close())
}

func TestExtractPkgArchive(t *testing.T) {
	testDir := getTestDir("test_un_tar")
	tarPath := filepath.Join(testDir, "test.tar")

	tarDest := filepath.Join(t.TempDir(), "tar")
	assert.NoError(t, ExtractPkgArchive(tarPath, tarDest))
	assert.True(t, DirExists(tarDest))

	tgzPath := filepath.Join(t.TempDir(), "test.tgz")
	gzipTarFile(t, tarPath, tgzPath)
	tgzDest := filepath.Join(t.TempDir(), "tgz")
	assert.NoError(t, ExtractPkgArchive(tgzPath, tgzDest))
	assert.True(t, DirExists(tgzDest))
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

func TestFindPackage(t *testing.T) {
	testDir := getTestDir("test_find_package")
	correctAddress := filepath.Join(testDir, "test_2")
	foundAddress, _ := FindPackage(testDir, "test_find_package")
	assert.Equal(t, foundAddress, correctAddress)
}

func TestMatchesPackageName(t *testing.T) {
	address := filepath.Join(getTestDir("test_find_package"), "test_2", "kcl.mod")
	assert.Equal(t, MatchesPackageName(address, "test_find_package"), true)
}

func TestShortHash(t *testing.T) {
	hash, err := ShortHash(JoinPath("ghcr.io", "kcl-lang"))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, hash, "9ebd0ad063dba405")
}
