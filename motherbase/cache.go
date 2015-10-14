package motherbase

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/yangzhao28/phantom/commonlog"
)

var cacheLogger = commonlog.NewLogger("cache", "log", commonlog.DEBUG)

func Md5Sum(content []byte) string {
	md5Ctx := md5.New()
	md5Ctx.Write(content)
	cipher := md5Ctx.Sum(nil)
	return hex.EncodeToString(cipher)
}

type CacheItem struct {
	id         string
	body       string
	md5sum     string
	updateTime int64
}

func parseName(baseName string) (string, string, int64, error) {
	matcher := regexp.MustCompile(`([a-zA-Z0-9]+)_([0-9a-fA-F]+)_([0-9]+)`)
	result := matcher.FindStringSubmatch(baseName)
	if len(result) < 4 {
		return "", "", 0, errors.New("invalid file name: " + baseName)
	}
	updateTime, err := strconv.ParseInt(result[3], 10, 64)
	if err != nil {
		return "", "", 0, err
	}
	return result[1], result[2], updateTime, nil
}

func fromFile(fileName string) (*CacheItem, error) {
	baseName := filepath.Base(fileName)
	id, md5sum, updateTime, err := parseName(baseName)
	if err != nil {
		return nil, err
	}
	configFile, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer configFile.Close()
	body, err := ioutil.ReadAll(configFile)
	if err != nil {
		return nil, err
	}
	// 检查MD5
	if Md5Sum(body) != md5sum {
		return nil, errors.New("unmatched md5sum")
	}
	return &CacheItem{
		id:         id,
		body:       string(body),
		md5sum:     md5sum,
		updateTime: updateTime,
	}, nil
}

func (item *CacheItem) GetFileName() string {
	return item.id + "_" + item.md5sum + "_" + strconv.FormatInt(item.updateTime, 10)
}

func (item *CacheItem) toFile(directory string) error {
	fileName := item.GetFileName()
	fullPath := filepath.Join(directory, fileName)
	configFile, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer configFile.Close()
	configFile.WriteString(item.body)
	return nil
}

type PersistCache struct {
	items            map[string]*CacheItem
	persistDirectory string
	mutex            sync.RWMutex
}

func NewPersistCache(persistDirectory string) *PersistCache {
	return &PersistCache{
		items:            make(map[string]*CacheItem),
		persistDirectory: persistDirectory,
	}
}

func (cache *PersistCache) Get(name string) (string, error) {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()
	if item, ok := cache.items[name]; ok {
		cacheLogger.Debug(name + " found")
		return item.body, nil
	} else {
		cacheLogger.Debug(name + " not found")
		return "", errors.New(name + " not exist")
	}
}

func (cache *PersistCache) save(name string, body string) error {
	item := &CacheItem{
		id:         name,
		body:       body,
		md5sum:     Md5Sum([]byte(body)),
		updateTime: time.Now().Unix(),
	}
	err := item.toFile(cache.persistDirectory)
	if err != nil {
		return err
	}
	cache.items[name] = item
	return nil
}

func (cache *PersistCache) Save(name string, body string) error {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	return cache.save(name, body)
}

func (cache *PersistCache) remove(name string) error {
	if item, ok := cache.items[name]; ok {
		fullFileName := filepath.Join(cache.persistDirectory, item.GetFileName())
		if file, err := os.Open(fullFileName); err == nil {
			file.Close()
			err := os.Remove(fullFileName)
			if err != nil {
				return err
			}
		}
		delete(cache.items, name)
	}
	return nil
}

func (cache *PersistCache) Remove(name string) error {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	return cache.remove(name)
}

func (cache *PersistCache) Replace(name string, body string) error {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	if err := cache.remove(name); err != nil {
		return err
	}
	return cache.save(name, body)
}

func (cache *PersistCache) Clean() error {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	fullPath, err := filepath.Abs(cache.persistDirectory)
	if err != nil {
		return err
	}
	files, err := ioutil.ReadDir(fullPath)
	if err != nil {
		return err
	}
	toDelete := make([]string, 0)
	type temp struct {
		FileName  string
		Timestamp int64
	}
	filter := make(map[string]temp)
	// find old file to delete
	for _, fileInfo := range files {
		if !fileInfo.IsDir() {
			id, _, updateTime, err := parseName(fileInfo.Name())
			if err != nil || fileInfo.Size() == 0 {
				toDelete = append(toDelete, fileInfo.Name())
			}
			value, ok := filter[id]
			if ok {
				if value.Timestamp > updateTime {
					toDelete = append(toDelete, fileInfo.Name())
					continue
				} else {
					toDelete = append(toDelete, value.FileName)
				}
			}
			filter[id] = temp{
				FileName:  fileInfo.Name(),
				Timestamp: updateTime,
			}
		}
	}
	// delete
	for _, fileName := range toDelete {
		os.Remove(fileName)
	}
	return nil
}

type CacheItemInfo struct {
	id              string
	md5sum          string
	updateTime      string
	updateTimestamp int64
}

func (cache *PersistCache) List() ([]CacheItemInfo, error) {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()
	itemList := make([]CacheItemInfo, 0)
	for _, value := range cache.items {
		itemList = append(itemList, CacheItemInfo{
			id:              value.id,
			md5sum:          value.md5sum,
			updateTime:      fmt.Sprintf("%v", time.Unix(value.updateTime, 0)),
			updateTimestamp: value.updateTime,
		})
		cacheLogger.Debug("  list -- " + value.id)
	}
	return itemList, nil
}

func (cache *PersistCache) Reload() error {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	fullPath, err := filepath.Abs(cache.persistDirectory)
	if err != nil {
		cacheLogger.Warning(err.Error())
		return err
	}
	cacheLogger.Notice(fmt.Sprintf("reload from %s", fullPath))
	files, err := ioutil.ReadDir(fullPath)
	if err != nil {
		cacheLogger.Warning(err.Error())
		return err
	}
	count := 0
	for _, fileInfo := range files {
		if !fileInfo.IsDir() {
			fullFileName := filepath.Join(fullPath, fileInfo.Name())
			item, err := fromFile(fullFileName)
			if err == nil {
				cache.items[item.id] = item
				count++
				cacheLogger.Debug(fmt.Sprintf("loaded -> file:%v", fullFileName))
			} else {
				cacheLogger.Debug(fmt.Sprintf("ignore -> file:%v error:%v", fullFileName, err.Error()))
			}
		}
	}
	cacheLogger.Notice(fmt.Sprintf("%v item(s) loaded, totally %v now", count, len(cache.items)))
	return nil
}
