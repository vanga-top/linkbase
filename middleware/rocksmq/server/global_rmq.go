package server

import (
	"github.com/cockroachdb/errors"
	"github.com/linkbase/middleware/log"
	"go.uber.org/zap"
	"os"
	"sync"
)

// Rmq is global rocksmq instance that will be initialized only once
var Rmq *RocketMQServer

// once is used to init global rocksmq
var once sync.Once

// InitRocksMQ init global rocksmq single instance
func InitRocksMQ(path string) error {
	var finalErr error
	once.Do(func() {
		log.Debug("initializing global rmq", zap.String("path", path))
		var fi os.FileInfo
		fi, finalErr = os.Stat(path)
		if os.IsNotExist(finalErr) {
			finalErr = os.MkdirAll(path, os.ModePerm)
			if finalErr != nil {
				return
			}
		} else {
			if !fi.IsDir() {
				errMsg := "can't create a directory because there exists a file with the same name"
				finalErr = errors.New(errMsg)
				return
			}
		}
		Rmq, finalErr = NewRocksMQ(path, nil)
	})
	return finalErr
}

// CloseRocksMQ is used to close global rocksmq
func CloseRocksMQ() {
	log.Debug("Close Rocksmq!")
	if Rmq != nil && Rmq.store != nil {
		Rmq.Close()
	}
}
