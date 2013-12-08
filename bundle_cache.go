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

const VERSION = "0.2.0"

const(
  ERR_OK             = 0
  ERR_WRONG_USAGE    = 2
  ERR_NO_CREDENTIALS = 3
  ERR_NO_BUNDLE      = 4
  ERR_BUNDLE_EXISTS  = 5
  ERR_NO_GEMLOCK     = 6
)

var options struct {
  Prefix        string `long:"prefix"     description:"Custom archive filename (default: current dir)"`
  Path          string `long:"path"       description:"Path to directory with .bundle (default: current)"`
  AccessKey     string `long:"access-key" description:"S3 Access key"`
  SecretKey     string `long:"secret-key" description:"S3 Secret key"`
  Bucket        string `long:"bucket"     description:"S3 Bucket name"`
  BundlePath    string
  LockFilePath  string
  CacheFilePath string
  ArchiveName   string
  ArchivePath   string
  ArchiveUrl    string
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

func transferArchive(file string, url string, fail_status int) {
  r, err := open(file)
  if err != nil {
    terminateWithError(err, fail_status)
  }

  w, err := create(url)
  if err != nil {
    terminateWithError(err, fail_status)
  }

  _, err = io.Copy(w, r)
  if err != nil {
    terminateWithError(err, fail_status)
  }

  err = w.Close()
  if err != nil {
    terminateWithError(err, fail_status)
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
  if len(options.AccessKey) == 0 && envDefined("S3_ACCESS_KEY") {
    options.AccessKey = os.Getenv("S3_ACCESS_KEY")
  }

  if len(options.SecretKey) == 0 && envDefined("S3_SECRET_KEY") {
    options.SecretKey = os.Getenv("S3_SECRET_KEY")
  }

  if len(options.Bucket) == 0 && envDefined("S3_BUCKET") {
    options.Bucket = os.Getenv("S3_BUCKET")
  }

  if len(options.AccessKey) == 0 { 
    terminate("Please provide S3 access key", ERR_NO_CREDENTIALS) 
  }
  
  if len(options.SecretKey) == 0 { 
    terminate("Please provide S3 secret key", ERR_NO_CREDENTIALS) 
  }
  
  if len(options.Bucket) == 0 { 
    terminate("Please provide S3 bucket name", ERR_NO_CREDENTIALS) 
  }

  s3util.DefaultConfig.AccessKey = options.AccessKey
  s3util.DefaultConfig.SecretKey = options.SecretKey
}

func printUsage() {
  terminate("Usage: bundle_cache [download|upload]", ERR_WRONG_USAGE)
}

func upload() {
  if fileExists(options.CacheFilePath) {
    terminate("Your bundle is cached, skipping.", ERR_OK)
  }

  if !fileExists(options.BundlePath) {
    terminate("Bundle path does not exist", ERR_NO_BUNDLE)
  }

  fmt.Println("Archiving...")
  cmd := fmt.Sprintf("cd %s && tar -czf %s .", options.BundlePath, options.ArchivePath)
  if _, err := sh(cmd); err != nil {
    terminate("Failed to make archive.", 1)
  }

  fmt.Println("Uploading bundle to S3...")
  transferArchive(options.ArchivePath, options.ArchiveUrl, 0)

  fmt.Println("Done")
  os.Exit(0)
}

func download() {
  if fileExists(options.BundlePath) {
    terminate("Bundle path already exists, skipping.", 0)
  }

  fmt.Println("Downloading bundle from S3...", options.ArchiveUrl)
  transferArchive(options.ArchiveUrl, options.ArchivePath, 0)

  /* Extract archive into bundle directory */
  fmt.Println("Extracting...")
  extractArchive(options.ArchivePath, options.Path)

  /* Create a temp file in path to indicate that bundle was cached */
  if !fileExists(options.CacheFilePath) {
    sh(fmt.Sprintf("touch %s", options.CacheFilePath))
  }

  fmt.Println("Done")
  os.Exit(0)
}

func getAction() string {
  new_args, err := flags.ParseArgs(&options, os.Args)

  if err != nil {
    fmt.Println(err)
    os.Exit(ERR_WRONG_USAGE)
  }

  args := new_args[1:]

  if len(args) != 1 {
    printUsage()
  }

  return args[0] 
}

func setOptions() {
  if len(options.Path) == 0 {
    options.Path, _ = os.Getwd()
  }

  if len(options.Prefix) == 0 {
    options.Prefix = filepath.Base(options.Path)
  }

  options.BundlePath    = fmt.Sprintf("%s/.bundle", options.Path)
  options.LockFilePath  = fmt.Sprintf("%s/Gemfile.lock", options.Path)
  options.CacheFilePath = fmt.Sprintf("%s/.cache", options.BundlePath)
}

func setArchiveOptions() {
  lockfile, err := ioutil.ReadFile(options.LockFilePath)
  if err != nil {
    terminate("Unable to read Gemfile.lock", 1)
  }
  
  checksum := calculateChecksum(string(lockfile))

  options.ArchiveName = fmt.Sprintf("%s_%s_%s.tar.gz", options.Prefix, checksum, runtime.GOARCH)
  options.ArchivePath = fmt.Sprintf("/tmp/%s", options.ArchiveName)
  options.ArchiveUrl  = s3url(options.ArchiveName)

  if fileExists(options.ArchivePath) {
    if os.Remove(options.ArchivePath) != nil {
      terminate("Failed to remove existing archive", 1)
    }
  }
}

func checkGemlock() {
  if !fileExists(options.LockFilePath) {
    message := fmt.Sprintf("%s does not exist", options.LockFilePath)
    terminate(message, ERR_NO_GEMLOCK)
  }
}

func main() {
  action := getAction()
  
  checkS3Credentials()
  setOptions()
  checkGemlock()
  setArchiveOptions()

  switch action {
  default:
    fmt.Println("Invalid command:", action)
    printUsage()
  case "upload":
    upload()
  case "download":
    download()
  }
}
