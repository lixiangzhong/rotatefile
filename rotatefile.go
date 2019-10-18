package rotatefile

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/robfig/cron"
)

var (
	stdlog         = log.New(os.Stderr, "rotatefile", log.Lshortfile|log.LstdFlags)
	dailyspec      = "@daily"
	dailycron      *cron.Cron
	DateTimeFormat = "2006-01-02"
)

func init() {
	dailycron = cron.New()
	dailycron.Start()
}

func New(name string, rotate int, daily bool) *RotateFile {
	return &RotateFile{
		filename: name,
		rotate:   rotate,
		daily:    daily,
	}
}

type RotateFile struct {
	filename string
	rotate   int
	daily    bool
	// compress bool
	*os.File
	once  sync.Once
	mutex sync.Mutex
}

func (r *RotateFile) Write(b []byte) (int, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.once.Do(func() {
		if r.daily {
			dailycron.AddFunc(dailyspec, r.dailyrotate)
		}
	})

	if r.File == nil {
		if err := r.openfile(); err != nil {
			return 0, err
		}
	}
	return r.File.Write(b)
}

func (r *RotateFile) Close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.close()
}

func (r *RotateFile) close() error {
	if r.File == nil {
		return nil
	}
	err := r.File.Close()
	r.File = nil
	return err
}

func (r *RotateFile) openfile() error {
	if err := os.MkdirAll(filepath.Dir(r.filename), 0744); err != nil {
		return err
	}
	f, err := os.OpenFile(r.filename, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	r.File = f
	return nil
}

func (r *RotateFile) dailyrotate() {
	now := time.Now()
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if err := r.close(); err != nil {
		stdlog.Println(err)
	}
	yesterday := now.AddDate(0, 0, -1)

	if err := os.Rename(r.filename, fmt.Sprintf("%v.%v", r.filename, yesterday.Format(DateTimeFormat))); err != nil {
		stdlog.Println(err)
	}
	if err := r.openfile(); err != nil {
		stdlog.Println(err)
	}
	os.Remove(fmt.Sprintf("%v.%v", r.filename, yesterday.AddDate(0, 0, -r.rotate).Format(DateTimeFormat)))
}
