package main

import(
  "fmt"
  "io"
  "io/ioutil"
  "crypto/sha1"
  "os"
  "os/exec"
  "path/filepath"
  "bytes"
  "runtime"
  "strings"
  "github.com/kr/s3/s3util"
)

const VERSION = "0.1.0"

func fileExists(path string) bool {
  _, err := os.Stat(path)
  return err == nil
}

func open(s string) (io.ReadCloser, error) {
  if isURL(s) {
    return s3util.Open(s, nil)
  }
  return os.Open(s)
}

func create(s string) (io.WriteCloser, error) {
  if isURL(s) {
    return s3util.Create(s, nil, nil)
  }
  return os.Create(s)
}

func isURL(s string) bool {
  return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func s3url(filename string) string {
  format := "https://s3.amazonaws.com/%s/%s"
  url := fmt.Sprintf(format, os.Getenv("S3_BUCKET"), filename)

  return url
}

func sh(command string) (string, error) {
  var output bytes.Buffer
 
  cmd := exec.Command("bash", "-c", command)
 
  cmd.Stdout = &output
  cmd.Stderr = &output
 
  err := cmd.Run()
  return output.String(), err
}

func calculateChecksum(buffer string) string {
  h := sha1.New()
  io.WriteString(h, buffer)
  return fmt.Sprintf("%x", h.Sum(nil))
}

func transferArchive(file string, url string) {
  s3util.DefaultConfig.AccessKey = os.Getenv("S3_ACCESS_KEY")
  s3util.DefaultConfig.SecretKey = os.Getenv("S3_SECRET_KEY")

  r, err := open(file)
  if err != nil {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(1)
  }

  w, err := create(url)
  if err != nil {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(1)
  }

  _, err = io.Copy(w, r)
  if err != nil {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(1)
  }

  err = w.Close()
  if err != nil {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(1)
  }
}

func extractArchive(filename string, path string) bool {
  cmd_mkdir   := fmt.Sprintf("cd %s && mkdir .bundle", path)
  cmd_move    := fmt.Sprintf("mv %s %s/.bundle/bundle_cache.tar.gz", filename, path)
  cmd_extract := fmt.Sprintf("cd %s/.bundle && tar -xzf ./bundle_cache.tar.gz", path)
  cmd_remove  := fmt.Sprintf("rm %s/.bundle/bundle_cache.tar.gz", path)

  if _, err := sh(cmd_mkdir) ; err != nil {
    fmt.Println("Bundle directory '.bundle' already exists")
    return false
  }

  if _, err := sh(cmd_move) ; err != nil {
    fmt.Println("Unable to move file")
    return false
  }

  if out, err := sh(cmd_extract) ; err != nil {
    fmt.Println("Unable to extract:", out)
    return false
  }

  if _, err := sh(cmd_remove) ; err != nil {
    fmt.Println("Unable to remove archive")
    return false
  }

  return true
}

func envDefined(name string) bool {
  result := os.Getenv(name)
  return len(result) > 0
}

func checkS3Credentials() {
  required := [3]string { "S3_ACCESS_KEY", "S3_SECRET_KEY", "S3_BUCKET" }

  for _, v := range required {
    if !envDefined(v) {
      fmt.Printf("Please define %s environment variable\n", v)
      os.Exit(2)
    }
  }
}

func printUsage() {
  fmt.Println("Usage: bundle_cache [download|upload]")
  os.Exit(2)
}

func upload(bundle_path string, archive_path string, archive_url string) {
  if !fileExists(bundle_path) {
    fmt.Println("Bundle path does not exist")
    os.Exit(1)
  }

  cmd := fmt.Sprintf("cd %s && tar -czf %s .", bundle_path, archive_path)

  fmt.Println("Archiving...")
  if out, err := sh(cmd); err != nil {
    fmt.Println("Failed to make archive:", out)
    os.Exit(1)
  }

  fmt.Println("Archived bundle at", archive_path)
  transferArchive(archive_path, archive_url)

  os.Exit(0)
}

func download(path string, bundle_path string, archive_path string, archive_url string) {
  if fileExists(bundle_path) {
    fmt.Println("Bundle path already exists")
    os.Exit(1)
  }

  fmt.Println("Downloading", archive_url)
  transferArchive(archive_url, archive_path)

  fmt.Println("Extracting...")
  extractArchive(archive_path, path)

  os.Exit(0)
}

func main() {
  args := os.Args[1:]

  if len(args) != 1 {
    printUsage()
  }

  action := args[0]

  /* Check if S3 credentials are set */
  checkS3Credentials()
  
  /* Get all path information */
  path, _       := os.Getwd()
  name          := filepath.Base(path)
  bundle_path   := fmt.Sprintf("%s/.bundle", path)
  lockfile_path := fmt.Sprintf("%s/Gemfile.lock", path)

  /* Check if lockfile exists */
  if !fileExists(lockfile_path) {
    fmt.Println("Gemfile.lock does not exist")
    os.Exit(1)
  }

  /* Read contents of lockfile */
  lockfile, err := ioutil.ReadFile(lockfile_path)
  if err != nil {
    fmt.Println("Unable to read Gemfile.lock")
    os.Exit(1)
  }

  /* Calculate SHA1 checksum for Gemfile.lock */
  checksum := calculateChecksum(string(lockfile))

  /* Make archive save filename */
  archive_name := fmt.Sprintf("%s_%s_%s.tar.gz", name, checksum, runtime.GOARCH)
  archive_path := fmt.Sprintf("/tmp/%s", archive_name)
  archive_url  := s3url(archive_name)

  /* Check if archive already exists */
  if fileExists(archive_path) {
    if os.Remove(archive_path) != nil {
      fmt.Println("Failed to remove existing archive")
      os.Exit(1)
    }
  }

  if action == "upload" || action == "up" {
    upload(bundle_path, archive_path, archive_url)
  }

  if action == "download" || action == "down" {
    download(path, bundle_path, archive_path, archive_url)
  }

  printUsage()
}
