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
  "github.com/jessevdk/go-flags"
)

const VERSION = "0.1.2"

const(
  ERR_WRONG_USAGE    = 2
  ERR_NO_CREDENTIALS = 3
  ERR_NO_BUNDLE      = 4
  ERR_BUNDLE_EXISTS  = 5
  ERR_NO_GEMLOCK     = 6
)

var options struct {
  Prefix    string `long:"prefix"     description:"Custom archive filename (default: current dir)"`
  Path      string `long:"path"       description:"Path to directory with .bundle (default: current)"`
  AccessKey string `long:"access-key" description:"S3 Access key"`
  SecretKey string `long:"secret-key" description:"S3 Secret key"`
  Bucket    string `long:"bucket"     description:"S3 Bucket name"`
}

func terminate(message string, exit_code int) {
  fmt.Fprintln(os.Stderr, message)
  os.Exit(exit_code)
}

func terminateWithError(err error, exit_code int) {
  fmt.Fprintln(os.Stderr, err)
  os.Exit(exit_code)
}

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
  url := fmt.Sprintf(format, options.Bucket, filename)

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
  s3util.DefaultConfig.AccessKey = options.AccessKey
  s3util.DefaultConfig.SecretKey = options.SecretKey

  r, err := open(file)
  if err != nil {
    terminateWithError(err, 1)
  }

  w, err := create(url)
  if err != nil {
    terminateWithError(err, 1)
  }

  _, err = io.Copy(w, r)
  if err != nil {
    terminateWithError(err, 1)
  }

  err = w.Close()
  if err != nil {
    terminateWithError(err, 1)
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
  if len(options.AccessKey) == 0 { 
    terminate("Please provide S3 access key", ERR_NO_CREDENTIALS) 
  }
  
  if len(options.SecretKey) == 0 { 
    terminate("Please provide S3 secret key", ERR_NO_CREDENTIALS) 
  }
  
  if len(options.Bucket) == 0 { 
    terminate("Please provide S3 bucket name", ERR_NO_CREDENTIALS) 
  }
}

func printUsage() {
  terminate("Usage: bundle_cache [download|upload]", ERR_WRONG_USAGE)
}

func upload(bundle_path string, archive_path string, archive_url string) {
  if !fileExists(bundle_path) {
    terminate("Bundle path does not exist", ERR_NO_BUNDLE)
  }

  fmt.Println("Archiving...")
  cmd := fmt.Sprintf("cd %s && tar -czf %s .", bundle_path, archive_path)
  if _, err := sh(cmd); err != nil {
    terminate("Failed to make archive.", 1)
  }

  fmt.Println("Transferring...")
  transferArchive(archive_path, archive_url)

  os.Exit(0)
}

func download(path string, bundle_path string, archive_path string, archive_url string) {
  if fileExists(bundle_path) {
    terminate("Bundle path already exists", ERR_BUNDLE_EXISTS)
  }

  fmt.Println("Downloading...", archive_url)
  transferArchive(archive_url, archive_path)

  fmt.Println("Extracting...")
  extractArchive(archive_path, path)

  os.Exit(0)
}

func main() {
  new_args, err := flags.ParseArgs(&options, os.Args)

  if err != nil {
    fmt.Println(err)
    os.Exit(ERR_WRONG_USAGE)
  }

  if len(options.AccessKey) == 0 && envDefined("S3_ACCESS_KEY") {
    options.AccessKey = os.Getenv("S3_ACCESS_KEY")
  }

  if len(options.SecretKey) == 0 && envDefined("S3_SECRET_KEY") {
    options.SecretKey = os.Getenv("S3_SECRET_KEY")
  }

  if len(options.Bucket) == 0 && envDefined("S3_BUCKET") {
    options.Bucket = os.Getenv("S3_BUCKET")
  }

  args := new_args[1:]

  if len(args) != 1 {
    printUsage()
  }

  action := args[0]

  checkS3Credentials()

  if len(options.Path) == 0 {
    options.Path, _ = os.Getwd()
  }

  if len(options.Prefix) == 0 {
    options.Prefix = filepath.Base(options.Path)
  }

  bundle_path   := fmt.Sprintf("%s/.bundle", options.Path)
  lockfile_path := fmt.Sprintf("%s/Gemfile.lock", options.Path)

  if !fileExists(lockfile_path) {
    terminate("Gemfile.lock does not exist", ERR_NO_GEMLOCK)
  }

  lockfile, err := ioutil.ReadFile(lockfile_path)
  if err != nil {
    terminate("Unable to read Gemfile.lock", 1)
  }

  checksum     := calculateChecksum(string(lockfile))
  archive_name := fmt.Sprintf("%s_%s_%s.tar.gz", options.Prefix, checksum, runtime.GOARCH)
  archive_path := fmt.Sprintf("/tmp/%s", archive_name)
  archive_url  := s3url(archive_name)

  if fileExists(archive_path) {
    if os.Remove(archive_path) != nil {
      terminate("Failed to remove existing archive", 1)
    }
  }

  if action == "upload" || action == "up" {
    upload(bundle_path, archive_path, archive_url)
  }

  if action == "download" || action == "down" {
    download(options.Path, bundle_path, archive_path, archive_url)
  }

  fmt.Println("Invalid command:", action)
  printUsage()
}
