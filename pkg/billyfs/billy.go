package billyfs

import (
	"io/fs"

	"github.com/go-git/go-billy/v5"
)

// BillyFS is the minimum interface from the billy package required
// to implement an fs.FS implementation.
type BillyFS interface {
	billy.Basic
	billy.Dir
}

// FS is an implementation of fs.FS which wraps a billy filesystem abtraction.
type FS struct {
	fs BillyFS
}

func New(fs BillyFS) *FS {
	return &FS{fs}
}

// ReadDir reads the named directory
// and returns a list of directory entries sorted by filename.
func (f *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	infos, err := f.fs.ReadDir(name)
	if err != nil {
		return nil, err
	}

	entries := make([]fs.DirEntry, 0, len(infos))
	for _, info := range infos {
		entries = append(entries, fs.FileInfoToDirEntry(info))
	}

	return entries, nil
}

// Open opens the named file.
// When Open returns an error, it should be of type *PathError
// with the Op field set to "open", the Path field set to name,
// and the Err field describing the problem.
//
// Open should reject attempts to open names that do not satisfy
// ValidPath(name), returning a *PathError with Err set to
// ErrInvalid or ErrNotExist.
func (f *FS) Open(name string) (fs.File, error) {
	fi, err := f.fs.Open(name)
	if err != nil {
		return nil, err
	}

	return &File{fi, name, f}, nil
}

// File is an implementation of fs.File which wraps a billy.File instance.
type File struct {
	billy.File

	name string
	fs   *FS
}

// ReadDir reads the contents of the directory and returns
// a slice of up to n DirEntry values in directory order.
// Subsequent calls on the same file will yield further DirEntry values.
//
// If n > 0, ReadDir returns at most n DirEntry structures.
// In this case, if ReadDir returns an empty slice, it will return
// a non-nil error explaining why.
// At the end of a directory, the error is io.EOF.
// (ReadDir must return io.EOF itself, not an error wrapping io.EOF.)
//
// If n <= 0, ReadDir returns all the DirEntry values from the directory
// in a single slice. In this case, if ReadDir succeeds (reads all the way
// to the end of the directory), it returns the slice and a nil error.
// If it encounters an error before the end of the directory,
// ReadDir returns the DirEntry list read until that point and a non-nil error.
func (f *File) ReadDir(n int) ([]fs.DirEntry, error) {
	return f.fs.ReadDir(f.name)
}

// Stat returns a FileInfo describing the file.
// If there is an error, it should be of type *PathError.
func (f *File) Stat() (fs.FileInfo, error) {
	return f.fs.fs.Stat(f.name)
}
