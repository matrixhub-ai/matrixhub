package lfs

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
)

var (
	ErrNotOwner = errors.New("Attempt to delete other user's lock")
	errNoBucket = errors.New("Bucket not found")
)

// LockDB implements a metadata storage. It stores user credentials and Meta information
// for objects. The storage is handled by boltdb.
type LockDB struct {
	db *bolt.DB
}

var (
	locksBucket = []byte("locks")
)

// NewLock creates a new MetaStore using the boltdb database at dbFile.
func NewLock(dbFile string) *LockDB {
	err := os.MkdirAll(filepath.Dir(dbFile), 0755)
	if err != nil {
		panic(fmt.Sprintf("Failed to create directory for boltdb file %s: %v", dbFile, err))
	}
	db, err := bolt.Open(dbFile, 0600, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		panic(fmt.Sprintf("Failed to open boltdb file %s: %v", dbFile, err))
	}

	db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(locksBucket); err != nil {
			return err
		}
		return nil
	})

	return &LockDB{db: db}
}

// Add write locks to the store for the repo.
func (s *LockDB) Add(repo string, l ...Lock) error {
	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(locksBucket)
		if bucket == nil {
			return errNoBucket
		}

		var locks []Lock
		data := bucket.Get([]byte(repo))
		if data != nil {
			if err := json.Unmarshal(data, &locks); err != nil {
				return err
			}
		}
		locks = append(locks, l...)
		sort.Slice(locks, func(i, j int) bool {
			return locks[i].LockedAt.Before(locks[j].LockedAt)
		})
		data, err := json.Marshal(&locks)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(repo), data)
	})
	return err
}

// List retrieves locks for the repo from the store
func (s *LockDB) List(repo string) ([]Lock, error) {
	var locks []Lock
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(locksBucket)
		if bucket == nil {
			return errNoBucket
		}

		data := bucket.Get([]byte(repo))
		if data != nil {
			if err := json.Unmarshal(data, &locks); err != nil {
				return err
			}
		}
		return nil
	})
	return locks, err
}

// Filtered return filtered locks for the repo
func (s *LockDB) Filtered(repo, path, cursor, limit string) (locks []Lock, next string, err error) {
	locks, err = s.List(repo)
	if err != nil {
		return
	}

	if cursor != "" {
		lastSeen := -1
		for i, l := range locks {
			if l.Id == cursor {
				lastSeen = i
				break
			}
		}

		if lastSeen > -1 {
			locks = locks[lastSeen:]
		} else {
			err = fmt.Errorf("cursor (%s) not found", cursor)
			return
		}
	}

	if path != "" {
		var filtered []Lock
		for _, l := range locks {
			if l.Path == path {
				filtered = append(filtered, l)
			}
		}

		locks = filtered
	}

	if limit != "" {
		var size int
		size, err = strconv.Atoi(limit)
		if err != nil || size < 0 {
			locks = make([]Lock, 0)
			err = fmt.Errorf("Invalid limit amount: %s", limit)
			return
		}

		size = int(math.Min(float64(size), float64(len(locks))))
		if size+1 < len(locks) {
			next = locks[size].Id
		}
		locks = locks[:size]
	}

	return locks, next, nil
}

// Delete removes lock for the repo by id from the store
func (s *LockDB) Delete(repo, user, id string, force bool) (*Lock, error) {
	var deleted *Lock
	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(locksBucket)
		if bucket == nil {
			return errNoBucket
		}

		var locks []Lock
		data := bucket.Get([]byte(repo))
		if data != nil {
			if err := json.Unmarshal(data, &locks); err != nil {
				return err
			}
		}
		newLocks := make([]Lock, 0, len(locks))

		var lock Lock
		for _, l := range locks {
			if l.Id == id {
				if l.Owner.Name != user && !force {
					return ErrNotOwner
				}
				lock = l
			} else if len(l.Id) > 0 {
				newLocks = append(newLocks, l)
			}
		}
		if lock.Id == "" {
			return nil
		}
		deleted = &lock

		if len(newLocks) == 0 {
			return bucket.Delete([]byte(repo))
		}

		data, err := json.Marshal(&newLocks)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(repo), data)
	})
	return deleted, err
}

// Close closes the underlying boltdb.
func (s *LockDB) Close() {
	s.db.Close()
}

type User struct {
	Name string `json:"name"`
}

type Lock struct {
	Id       string    `json:"id"`
	Path     string    `json:"path"`
	Owner    User      `json:"owner"`
	LockedAt time.Time `json:"locked_at"`
}

type LockRequest struct {
	Path string `json:"path"`
}

type LockResponse struct {
	Lock    *Lock  `json:"lock"`
	Message string `json:"message,omitempty"`
}

type UnlockRequest struct {
	Force bool `json:"force"`
}

type UnlockResponse struct {
	Lock    *Lock  `json:"lock"`
	Message string `json:"message,omitempty"`
}

type LockList struct {
	Locks      []Lock `json:"locks"`
	NextCursor string `json:"next_cursor,omitempty"`
	Message    string `json:"message,omitempty"`
}

type VerifiableLockRequest struct {
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type VerifiableLockList struct {
	Ours       []Lock `json:"ours"`
	Theirs     []Lock `json:"theirs"`
	NextCursor string `json:"next_cursor,omitempty"`
	Message    string `json:"message,omitempty"`
}
