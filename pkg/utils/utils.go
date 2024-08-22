package utils

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	goerrors "errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/distribution/reference"
	"github.com/moby/term"
	"github.com/otiai10/copy"

	"kcl-lang.io/kcl-go/pkg/utils"
	"kcl-lang.io/kpm/pkg/constants"
	"kcl-lang.io/kpm/pkg/errors"
	"kcl-lang.io/kpm/pkg/reporter"
)

// HashDir computes the checksum of a directory by concatenating all files and
// hashing them by sha256.
func HashDir(dir string) (string, error) {
	hasher := sha256.New()

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// files in the ".git "directory will cause the same repository, cloned at different times,
		// has different checksum.
		for _, ignore := range ignores {
			if strings.Contains(path, ignore) {
				return nil
			}
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(hasher, f); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(hasher.Sum(nil)), nil
}

// StoreToFile will store 'data' into toml file under 'filePath'.
func StoreToFile(filePath string, dataStr string) error {
	err := os.WriteFile(filePath, []byte(dataStr), 0644)
	if err != nil {
		reporter.ExitWithReport("failed to write file: ", filePath, err)
		return err
	}
	return nil
}

// ParseRepoNameFromGitUrl get the repo name from git url,
// the repo name in 'https://github.com/xxx/kcl1.git' is 'kcl1'.
func ParseRepoNameFromGitUrl(gitUrl string) string {
	name := filepath.Base(gitUrl)
	return name[:len(name)-len(filepath.Ext(name))]
}

// CreateFileIfNotExist will check whether the file under a certain path 'filePath/fileName' exists,
// and return an error if it exists, and call the method 'storeFunc' to save the file if it does not exist.
func CreateFileIfNotExist(filePath string, storeFunc func() error) error {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		err := storeFunc()
		if err != nil {
			return reporter.NewErrorEvent(reporter.FailedCreateFile, err, fmt.Sprintf("failed to create '%s'", filePath))
		}
	} else {
		return reporter.NewErrorEvent(reporter.FileExists, err, fmt.Sprintf("'%s' already exists", filePath))
	}
	return nil
}

// Whether the file exists
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// todo: Consider using the OCI tarball as the standard tar format.
var ignores = []string{".git", ".tar"}

func TarDir(srcDir string, tarPath string, include []string, exclude []string) error {
	fw, err := os.Create(tarPath)
	if err != nil {
		log.Fatal(err)
	}
	defer fw.Close()

	tw := tar.NewWriter(fw)
	defer tw.Close()

	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		for _, ignore := range ignores {
			if strings.Contains(path, ignore) {
				return nil
			}
		}

		getNewPattern := func(ex string) string {
			newPath := ex
			if !strings.HasPrefix(ex, srcDir+string(filepath.Separator)) {
				newPath = filepath.Join(srcDir, ex)
			}
			return newPath
		}

		for _, ex := range exclude {
			if matched, _ := filepath.Match(getNewPattern(ex), path); matched {
				return nil
			}
		}

		for _, inc := range include {
			if matched, _ := filepath.Match(getNewPattern(inc), path); !matched {
				return nil
			}
		}

		relPath, _ := filepath.Rel(srcDir, path)
		relPath = filepath.ToSlash(relPath)

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = relPath

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		if info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		fr, err := os.Open(path)
		if err != nil {
			return err
		}
		defer fr.Close()

		if _, err := io.Copy(tw, fr); err != nil {
			return err
		}

		return nil
	})

	return err
}

// UnTarDir will extract tar from 'tarPath' to 'destDir'.
func UnTarDir(tarPath string, destDir string) error {
	file, err := os.Open(tarPath)
	if err != nil {
		return reporter.NewErrorEvent(reporter.FailedCreateFile, err, fmt.Sprintf("failed to open '%s'", tarPath))
	}
	defer file.Close()

	tarReader := tar.NewReader(file)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return reporter.NewErrorEvent(reporter.FailedCreateFile, err, fmt.Sprintf("failed to open '%s'", tarPath))
		}

		destFilePath := filepath.Join(destDir, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destFilePath, 0755); err != nil {
				return errors.FailedUnTarKclPackage
			}
		case tar.TypeReg:
			err := os.MkdirAll(filepath.Dir(destFilePath), 0755)
			if err != nil {
				return err
			}
			outFile, err := os.Create(destFilePath)
			if err != nil {
				return reporter.NewErrorEvent(reporter.FailedCreateFile, err, fmt.Sprintf("failed to open '%s'", tarPath))
			}
			defer outFile.Close()

			if _, err := io.Copy(outFile, tarReader); err != nil {
				return reporter.NewErrorEvent(reporter.FailedCreateFile, err, fmt.Sprintf("failed to open '%s'", tarPath))
			}
		default:
			return errors.UnknownTarFormat
		}
	}
	return nil
}

// ExtractTarball support extracting tarball with '.tgz' format.
func ExtractTarball(tarPath, destDir string) error {
	f, err := os.Open(tarPath)
	if err != nil {
		return reporter.NewErrorEvent(reporter.FailedCreateFile, err, fmt.Sprintf("failed to open '%s'", tarPath))
	}
	defer f.Close()

	zip, err := gzip.NewReader(f)
	if err != nil {
		return reporter.NewErrorEvent(reporter.FailedCreateFile, err, fmt.Sprintf("failed to open '%s'", tarPath))
	}
	tarReader := tar.NewReader(zip)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return reporter.NewErrorEvent(reporter.FailedCreateFile, err, fmt.Sprintf("failed to open '%s'", tarPath))
		}

		destFilePath := filepath.Join(destDir, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destFilePath, 0755); err != nil {
				return errors.FailedUnTarKclPackage
			}
		case tar.TypeReg:
			err := os.MkdirAll(filepath.Dir(destFilePath), 0755)
			if err != nil {
				return err
			}
			outFile, err := os.Create(destFilePath)
			if err != nil {
				return reporter.NewErrorEvent(reporter.FailedCreateFile, err, fmt.Sprintf("failed to open '%s'", tarPath))
			}
			defer outFile.Close()

			if _, err := io.Copy(outFile, tarReader); err != nil {
				return reporter.NewErrorEvent(reporter.FailedCreateFile, err, fmt.Sprintf("failed to open '%s'", tarPath))
			}
		default:
			return errors.UnknownTarFormat
		}
	}

	return nil
}

// DirExists will check whether the directory 'path' exists.
func DirExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

const ModRelativePathPattern = `\$\{([a-zA-Z0-9_-]+:)?KCL_MOD\}/`

// If the path preffix is `${KCL_MOD}` or `${KCL_MOD:xxx}`
func IsModRelativePath(s string) bool {
	re := regexp.MustCompile(ModRelativePathPattern)
	return re.MatchString(s)
}

// MoveFile will move the file from 'src' to 'dest'.
// On windows, it will copy the file from 'src' to 'dest', and then delete the file under 'src'.
// On unix-like systems, it will rename the file from 'src' to 'dest'.
func MoveFile(src, dest string) error {
	if utils.DirExists(dest) {
		err := os.RemoveAll(dest)
		if err != nil {
			return err
		}
	}

	destDir := filepath.Dir(dest)
	if !utils.DirExists(destDir) {
		err := os.MkdirAll(destDir, os.ModePerm)
		if err != nil {
			return err
		}
	}

	var err error
	if runtime.GOOS != "windows" {
		err = os.Rename(src, dest)
		if err != nil {
			return err
		}
	} else {
		err = copy.Copy(src, dest)
		if err != nil {
			return err
		}
		err = os.RemoveAll(src)
		if err != nil {
			return err
		}
	}
	return nil
}

// IsSymlinkValidAndExists will check whether the symlink exists and points to a valid target
// return three values: whether the symlink exists, whether it points to a valid target, and any error encountered
// Note: IsSymlinkValidAndExists is only useful on unix-like systems.
func IsSymlinkValidAndExists(symlinkPath string) (bool, bool, error) {
	// check if the symlink exists
	info, err := os.Lstat(symlinkPath)
	if err != nil && os.IsNotExist(err) {
		// symlink does not exist
		return false, false, nil
	} else if err != nil {
		// other error
		return false, false, err
	}

	// check if the file is a symlink
	if info.Mode()&os.ModeSymlink == os.ModeSymlink {
		// get the target of the symlink
		target, err := os.Readlink(symlinkPath)
		if err != nil {
			// can not read the target
			return true, false, err
		}

		// check if the target exists
		_, err = os.Stat(target)
		if err == nil {
			// target exists
			return true, true, nil
		}
		if os.IsNotExist(err) {
			// target does not exist
			return true, false, nil
		}
		return true, false, err
	}

	return false, false, fmt.Errorf("%s exists but is not a symlink", symlinkPath)
}

// DefaultKpmHome create the '.kpm' in the user home and return the path of ".kpm".
func CreateSubdirInUserHome(subdir string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", reporter.NewErrorEvent(reporter.Bug, err, "internal bugs, failed to load user home directory")
	}

	dirPath := filepath.Join(homeDir, subdir)
	if !DirExists(dirPath) {
		err = os.MkdirAll(dirPath, 0755)
		if err != nil {
			return "", reporter.NewErrorEvent(reporter.Bug, err, "internal bugs, failed to create directory")
		}
	}

	return dirPath, nil
}

// CreateSymlink will create symbolic link named 'newName' for 'oldName',
// and if the symbolic link already exists, it will be deleted and recreated.
// Note: CreateSymlink is only useful on unix-like systems.
func CreateSymlink(oldName, newName string) error {
	symExist, _, err := IsSymlinkValidAndExists(newName)

	if err != nil {
		return err
	}

	if symExist {
		err := os.Remove(newName)
		if err != nil {
			return err
		}
	}

	err = os.Symlink(oldName, newName)
	if err != nil {
		return err
	}
	return nil
}

// Copied/Adapted from https://github.com/helm/helm
func GetUsernamePassword(usernameOpt string, passwordOpt string, passwordFromStdinOpt bool) (string, string, error) {
	var err error
	username := usernameOpt
	password := passwordOpt

	if password == "" {
		if username == "" {
			username, err = readLine("Username: ", false)
			if err != nil {
				return "", "", err
			}
			username = strings.TrimSpace(username)
		}
		if username == "" {
			password, err = readLine("Token: ", true)
			if err != nil {
				return "", "", err
			} else if password == "" {
				return "", "", goerrors.New("token required")
			}
		} else {
			password, err = readLine("Password: ", true)
			if err != nil {
				return "", "", err
			} else if password == "" {
				return "", "", goerrors.New("password required")
			}
		}
	}

	return username, password, nil
}

// Copied/adapted from https://github.com/helm/helm
func readLine(prompt string, silent bool) (string, error) {
	fmt.Print(prompt)
	if silent {
		fd := os.Stdin.Fd()
		state, err := term.SaveState(fd)
		if err != nil {
			return "", err
		}
		err = term.DisableEcho(fd, state)
		if err != nil {
			return "", err
		}

		defer func() {
			restoreErr := term.RestoreTerminal(fd, state)
			if err == nil {
				err = restoreErr
			}
		}()
	}

	reader := bufio.NewReader(os.Stdin)
	line, _, err := reader.ReadLine()
	if err != nil {
		return "", err
	}
	if silent {
		fmt.Println()
	}

	return string(line), nil
}

// FindKFiles will find all the '.k' files in the 'path' directory.
func FindKFiles(path string) ([]string, error) {
	var files []string
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		if strings.HasSuffix(path, ".k") {
			files = append(files, path)
		}
		return files, nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".k") {
			files = append(files, filepath.Join(path, entry.Name()))
		}
	}
	return files, nil
}

// RmNewline will remove all the '\r\n' and '\n' in the string 's'.
func RmNewline(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", ""), "\n", "")
}

// JoinPath will join the 'elem' to the 'base' with '/'.
func JoinPath(base, elem string) string {
	base = strings.TrimSuffix(base, "/")
	elem = strings.TrimPrefix(elem, "/")
	return base + "/" + elem
}

// IsUrl will check whether the string 'str' is a url.
func IsURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// IsGitRepoUrl will check whether the string 'str' is a git repo url
func IsGitRepoUrl(str string) bool {
	r := regexp.MustCompile(`((git|ssh|http(s)?)|(git@[\w\.]+))(:(//)?)([\w\.@\:/\-~]+)(\.git)?(/)?`)
	return r.MatchString(str)
}

// IsRef will check whether the string 'str' is a reference.
func IsRef(str string) bool {
	_, err := reference.ParseNormalizedNamed(str)
	return err == nil
}

// IsTar will check whether the string 'str' is a tar path.
func IsTar(str string) bool {
	return strings.HasSuffix(str, constants.TarPathSuffix)
}

// IsKfile will check whether the string 'str' is a k file path.
func IsKfile(str string) bool {
	return strings.HasSuffix(str, constants.KFilePathSuffix)
}

// CheckPackageSum will check whether the 'checkedSum' is equal
// to the hash of the package under 'localPath'.
func CheckPackageSum(checkedSum, localPath string) bool {
	if checkedSum == "" {
		return false
	}

	sum, err := HashDir(localPath)

	if err != nil {
		return false
	}

	return checkedSum == sum
}

// AbsTarPath checks whether path 'tarPath' exists and whether path 'tarPath' ends with '.tar'
// And after checking, absTarPath return the abs path for 'tarPath'.
func AbsTarPath(tarPath string) (string, error) {
	absTarPath, err := filepath.Abs(tarPath)
	if err != nil {
		return "", err
	}

	if filepath.Ext(absTarPath) != ".tar" {
		return "", errors.InvalidKclPacakgeTar
	} else if !DirExists(absTarPath) {
		return "", errors.KclPacakgeTarNotFound
	}

	return absTarPath, nil
}

// FindPackage finds the package with the package name 'targetPackage' under the 'root' directory kcl.mod file.
func FindPackage(root, targetPackage string) (string, error) {
	var result string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			kclModPath := filepath.Join(path, constants.KCL_MOD)
			if _, err := os.Stat(kclModPath); err == nil {
				if matchesPackageName(kclModPath, targetPackage) {
					result = path
					return filepath.SkipAll
				}
			}
		}
		return nil
	})

	if err != nil {
		return "", err
	}
	if result == "" {
		return "", fmt.Errorf("kcl.mod with package '%s' not found", targetPackage)
	}
	return result, nil
}

// MatchesPackageName checks whether the package name in the kcl.mod file under 'kclModPath' is equal to 'targetPackage'.
func matchesPackageName(kclModPath, targetPackage string) bool {
	type Package struct {
		Name string `toml:"name"`
	}
	type ModFile struct {
		Package Package `toml:"package"`
	}

	var modFile ModFile
	_, err := toml.DecodeFile(kclModPath, &modFile)
	if err != nil {
		fmt.Printf("Error parsing kcl.mod file: %v\n", err)
		return false
	}

	return modFile.Package.Name == targetPackage
}
