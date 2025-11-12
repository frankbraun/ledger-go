// Package file implements file related utility functions.
package file

import (
  "crypto/sha256"
  "encoding/hex"
  "hash"
  "io"
  "os"
)

// Exists checks if filename exists already.
func Exists(filename string) (bool, error) {
  _, err := os.Stat(filename)
  if err != nil {
    if os.IsNotExist(err) {
      return false, nil
    }
    return false, err
  }
  return true, err
}

func hashSum(hash hash.Hash, filename string) (string, error) {
  fp, err := os.Open(filename)
  if err != nil {
    return "", err
  }
  defer fp.Close()
  if _, err := io.Copy(hash, fp); err != nil {
    return "", err
  }
  return hex.EncodeToString(hash.Sum(nil)), nil
}

func SHA256Sum(filename string) (string, error) {
  return hashSum(sha256.New(), filename)
}
